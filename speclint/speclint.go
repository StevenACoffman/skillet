// Package speclint validates an Agent Skill's frontmatter against the
// agentskills.io skill spec. It is the single source of truth for the spec's
// drift-prone data — the length caps, the name rules and the allowed key set — so
// that exegesis (which gates on these findings) and skillsaw (which scores
// them) cannot diverge on the spec by hand.
//
// The name rules live here because the spec states them as one rule set: length,
// character set, hyphen placement, and a match with the skill's directory. exegesis
// and skillsaw used to hold different halves of it (directory match in one, kebab
// case in the other), so each tool accepted names the spec rejects.
//
// Findings carry one of two severities, and only errors block (finding.Severity):
//
//   - errors are spec violations, plus the two Anthropic platform rules the family
//     has always enforced (no angle brackets in a description);
//   - warnings are things a client may tolerate but a reader should know: a key the
//     spec does not define, a description longer than the family's listing target, and
//     the platform's reserved words in a name.
package speclint

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/skill"
)

// The spec's length caps, in runes.
const (
	DescriptionMaxRunes   = 1024
	NameMaxRunes          = 64
	CompatibilityMaxRunes = 500
)

// SkillMaxLines is agentskills.io's recommended ceiling for a SKILL.md: "keep your main
// SKILL.md under 500 lines" and move detail into referenced files. It is a
// recommendation, not a validation rule of the spec; each tool decides what it does with
// it (exegesis gates new skills on it, skillsaw refuses edits that grow past it and flags
// it on a dimension). Held here so those tools cannot disagree on the figure.
const SkillMaxLines = 500

// DescriptionTargetRunes is the family's description length target, decided as D6 in
// stbenjam_improvements.md. It is not a spec limit: every enabled skill's description
// is sent on every request by both Claude Code and Pi, so a description over this is a
// cost paid on every turn. It is reported as a warning and never blocks.
const DescriptionTargetRunes = 300

// reAngle matches the angle brackets a plain-text description must not contain.
var reAngle = regexp.MustCompile(`[<>]`)

// AllowedFrontmatterKey reports whether k is a spec-permitted top-level
// frontmatter key.
//
// The set is the one https://agentskills.io/specification defines: `name` and
// `description` required, `license`, `compatibility`, `metadata` and
// `allowed-tools` optional. Checked against the published table 2026-08-08,
// which corrected three keys this rejected — a skill declaring its license was
// being told the key does not exist.
//
// `tags` is the one deliberate deviation: the spec does not define it, but it is
// the most-used key across the family's corpora (163 of the installed skills, and
// every book2skill output), so rejecting it would report a defect on nearly every
// skill in existence rather than describe one. Anything genuinely outside the spec
// belongs under `metadata`, which the spec provides for exactly that — including
// `author` and `version`, which are *not* top-level keys however often they are
// assumed to be.
func AllowedFrontmatterKey(k string) bool {
	switch k {
	case "name", "description", "license", "compatibility", "metadata", "allowed-tools":
		return true
	case "tags":
		return true
	default:
		return false
	}
}

// ClaudeCodeKey reports whether k is a top-level frontmatter key Claude Code documents
// beyond the spec's own (https://code.claude.com/docs/en/skills, checked 2026-11-02).
//
// It exists so an unknown-key warning can say which kind of unknown a key is. Claude Code
// acts on these, and moving one under `metadata` switches the behaviour off, so the
// warning for one tells the reader that other clients ignore it rather than that it is a
// mistake.
func ClaudeCodeKey(k string) bool {
	switch k {
	case "when_to_use", "argument-hint", "arguments", "disable-model-invocation",
		"user-invocable", "disallowed-tools", "model", "effort", "context", "agent",
		"background", "hooks", "paths", "shell":
		return true
	default:
		return false
	}
}

// Lines counts text's lines the way an editor numbers them: a final line without a
// trailing newline still counts, and empty text has none. It is the counter SkillMaxLines
// is measured with, so every tool reports the same number for the same file.
func Lines(text string) int {
	if text == "" {
		return 0
	}
	n := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}

// ValidSkillName reports whether name satisfies the spec's character rules: 1 to
// NameMaxRunes characters of ASCII `a-z`, `0-9` and `-`, not starting or ending with a
// hyphen, and with no two hyphens in a row.
//
// Names are compared byte for byte with no Unicode normalization (D3): restricting them
// to ASCII is what makes byte comparison and normalization agree.
//
// It is a skill-name predicate only. An Agent Plugins plugin name also allows `.`, so the
// two must not share a predicate.
func ValidSkillName(name string) bool {
	return nameProblem(name) == ""
}

// nameProblem returns why name breaks the spec's character rules, or "" when it does not.
func nameProblem(name string) string {
	switch {
	case name == "":
		return "is empty"
	case len(name) > NameMaxRunes:
		// Byte length equals rune length for the only names that can pass.
		return fmt.Sprintf("is %d characters > %d", len(name), NameMaxRunes)
	case strings.HasPrefix(name, "-"), strings.HasSuffix(name, "-"):
		return "starts or ends with a hyphen"
	case strings.Contains(name, "--"):
		return "contains consecutive hyphens"
	case !onlyNameBytes(name):
		return "may contain only a-z, 0-9 and hyphens"
	}
	return ""
}

// onlyNameBytes reports whether every byte of name is ASCII a-z, 0-9 or a hyphen. A
// byte test is exact here: any non-ASCII rune encodes to bytes outside these ranges.
func onlyNameBytes(name string) bool {
	for i := range len(name) {
		c := name[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

// Frontmatter returns the agentskills.io frontmatter findings for s. Errors are spec
// violations and block; warnings are advisory (see the package doc). An empty result
// means the frontmatter satisfies the spec and the family's targets. It is pure over an
// already-loaded skill.
//
// Frontmatter that did not parse is reported as itself and nothing else. Every
// other check reads a field that could not be read, so a block with one YAML
// syntax error would otherwise be reported as an empty description and a missing
// name — symptoms dressed up as independent defects, pointing the reader at lines
// that are perfectly fine.
//
// The name is checked against the skill's directory only when s.Dir is set, so a skill
// assembled in memory is not reported against a directory it does not have.
func Frontmatter(s *skill.Skill) []finding.Diagnostic {
	if errors.Is(s.FrontmatterErr, skill.ErrUnterminatedFrontmatter) {
		return []finding.Diagnostic{errorf("frontmatter: not closed: %v", s.FrontmatterErr)}
	}
	if s.FrontmatterErr != nil {
		return []finding.Diagnostic{errorf("frontmatter: not valid YAML: %v", s.FrontmatterErr)}
	}
	// s.Frontmatter already parsed once in skill.Load, so this cannot fail; a nil map
	// on any error simply reads every field as absent. The typed fields on skill.Skill
	// drop non-string values, which is why the types are read from the map here.
	var fields map[string]any
	_ = yaml.Unmarshal([]byte(s.Frontmatter), &fields)

	var ds []finding.Diagnostic
	for _, k := range s.FrontmatterKeys {
		ds = append(ds, keyFindings(k)...)
	}
	ds = append(ds, nameFindings(s, fields)...)
	ds = append(ds, descriptionFindings(s, fields)...)
	ds = append(ds, optionalFieldFindings(fields)...)
	return ds
}

// keyFindings reports a top-level key the spec does not define, as a warning.
//
// A warning, not an error (D2): neither Claude Code nor Pi fails on an unknown key, and
// the Agent Plugins spec settled the same question for its own manifest by reporting and
// ignoring. An error here would block skills that run in the harness they were written
// for.
func keyFindings(k string) []finding.Diagnostic {
	switch {
	case AllowedFrontmatterKey(k):
		return nil
	case ClaudeCodeKey(k):
		return []finding.Diagnostic{
			warnf(
				"frontmatter: key %q is a Claude Code extension, not part of the Agent Skills spec; other clients ignore it",
				k,
			),
		}
	default:
		return []finding.Diagnostic{
			warnf(
				"frontmatter: key %q is not in the Agent Skills spec; clients ignore it (put custom data under metadata)",
				k,
			),
		}
	}
}

// nameFindings applies the spec's name rules, the directory match, and the platform's
// reserved words.
func nameFindings(s *skill.Skill, fields map[string]any) []finding.Diagnostic {
	raw, present := fields["name"]
	if present && !isString(raw) {
		return []finding.Diagnostic{
			errorf("frontmatter: name must be a string, got %s", yamlType(raw)),
		}
	}
	if problem := nameProblem(s.Name); problem != "" {
		return []finding.Diagnostic{errorf("frontmatter: name %q %s", s.Name, problem)}
	}
	var ds []finding.Diagnostic
	if s.Dir != "" {
		if dir := filepath.Base(s.Dir); s.Name != dir {
			ds = append(ds, errorf("frontmatter: name %q != folder %q", s.Name, dir))
		}
	}
	for _, word := range []string{"anthropic", "claude"} {
		if strings.Contains(s.Name, word) {
			ds = append(ds, warnf(
				"frontmatter: name %q contains the reserved word %q (Anthropic platform rule; the API rejects it on upload)",
				s.Name,
				word,
			))
		}
	}
	return ds
}

// descriptionFindings applies the spec's description rules and the family's target.
func descriptionFindings(s *skill.Skill, fields map[string]any) []finding.Diagnostic {
	if raw, present := fields["description"]; present && !isString(raw) {
		return []finding.Diagnostic{errorf(
			"frontmatter: description must be a string, got %s", yamlType(raw))}
	}
	var ds []finding.Diagnostic
	switch n := utf8.RuneCountInString(s.Description); {
	case n == 0:
		ds = append(ds, errorf("frontmatter: description is empty"))
	case n > DescriptionMaxRunes:
		ds = append(ds, errorf("frontmatter: description %d runes > %d", n, DescriptionMaxRunes))
	case n > DescriptionTargetRunes:
		ds = append(ds, warnf(
			"frontmatter: description %d runes > %d target; it is sent on every request (put the key use case first)",
			n,
			DescriptionTargetRunes,
		))
	}
	if reAngle.MatchString(s.Description) {
		ds = append(ds, errorf(
			"frontmatter: description contains angle brackets (Anthropic platform rule: must be plain text)",
		))
	}
	return ds
}

// optionalFieldFindings checks the types and limits of the spec's optional fields.
func optionalFieldFindings(fields map[string]any) []finding.Diagnostic {
	var ds []finding.Diagnostic
	for _, k := range []string{"license", "allowed-tools"} {
		if raw, present := fields[k]; present && !isString(raw) {
			ds = append(ds, errorf("frontmatter: %s must be a string, got %s", k, yamlType(raw)))
		}
	}
	if raw, present := fields["compatibility"]; present {
		switch v, ok := raw.(string); {
		case !ok:
			ds = append(
				ds,
				errorf("frontmatter: compatibility must be a string, got %s", yamlType(raw)),
			)
		case strings.TrimSpace(v) == "":
			ds = append(ds, errorf("frontmatter: compatibility is empty (omit it instead)"))
		case utf8.RuneCountInString(v) > CompatibilityMaxRunes:
			ds = append(ds, errorf("frontmatter: compatibility %d runes > %d",
				utf8.RuneCountInString(v), CompatibilityMaxRunes))
		}
	}
	if raw, present := fields["metadata"]; present {
		ds = append(ds, metadataFindings(raw)...)
	}
	return ds
}

// metadataFindings requires metadata to be a mapping from string keys to string values,
// as the spec defines it. Keys are reported in sorted order so output is stable.
func metadataFindings(raw any) []finding.Diagnostic {
	m, ok := stringKeyed(raw)
	if !ok {
		return []finding.Diagnostic{
			errorf("frontmatter: metadata must be a mapping, got %s", yamlType(raw)),
		}
	}
	var ds []finding.Diagnostic
	for _, k := range sortedKeys(m) {
		if !isString(m[k]) {
			ds = append(ds, errorf(
				"frontmatter: metadata.%s must be a string, got %s", k, yamlType(m[k])))
		}
	}
	return ds
}

// stringKeyed returns raw as a map with string keys, accepting either map type the YAML
// decoder may produce. A mapping with a non-string key is not one the spec allows.
func stringKeyed(raw any) (map[string]any, bool) {
	switch m := raw.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = v
		}
		return out, true
	default:
		return nil, false
	}
}

func isString(v any) bool {
	_, ok := v.(string)
	return ok
}

// yamlType names a decoded YAML value's kind the way a skill author would.
func yamlType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "a boolean"
	case int, int64, uint64, float64:
		return "a number"
	case []any:
		return "a list"
	case map[string]any, map[any]any:
		return "a mapping"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// errorf builds an error-severity frontmatter diagnostic. Category and Path are left
// empty so the diagnostic marshals as {severity, message} — the shape both tools
// already emit, keeping their machine output stable.
func errorf(format string, a ...any) finding.Diagnostic {
	return finding.Diagnostic{Severity: finding.SeverityError, Message: fmt.Sprintf(format, a...)}
}

// warnf builds a warning-severity frontmatter diagnostic, which never blocks.
func warnf(format string, a ...any) finding.Diagnostic {
	return finding.Diagnostic{Severity: finding.SeverityWarning, Message: fmt.Sprintf(format, a...)}
}
