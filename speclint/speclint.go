// Package speclint validates an Agent Skill's frontmatter against the
// agentskills.io skill spec. It is the single source of truth for the spec's
// drift-prone data — the description length cap and the allowed key set — so
// that exegesis (which gates on these findings) and skillsaw (which scores
// them) cannot diverge on the spec by hand.
//
// Name-format policy is deliberately left to each tool: exegesis requires
// name == directory (a book2skill tree rule), skillsaw requires kebab-case (a
// rubric-scoring rule). Those are genuinely different rules, not one shared
// invariant, so folding them here would be special-purpose code in a
// general-purpose mechanism.
package speclint

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/skill"
)

// DescriptionMaxRunes is the agentskills.io description length cap.
const DescriptionMaxRunes = 1024

// reAngle matches the angle brackets a plain-text description must not contain.
var reAngle = regexp.MustCompile(`[<>]`)

// AllowedFrontmatterKey reports whether k is a spec-permitted top-level
// frontmatter key.
func AllowedFrontmatterKey(k string) bool {
	switch k {
	case "name", "description", "tags", "allowed-tools":
		return true
	default:
		return false
	}
}

// Frontmatter returns the agentskills.io frontmatter-spec violations for s as
// error-severity diagnostics: any disallowed top-level key, and a description
// that is empty, longer than DescriptionMaxRunes, or not plain text. An empty
// result means the frontmatter satisfies the shared spec. It is pure over an
// already-loaded skill.
func Frontmatter(s *skill.Skill) []finding.Diagnostic {
	var ds []finding.Diagnostic
	for _, k := range s.FrontmatterKeys {
		if !AllowedFrontmatterKey(k) {
			ds = append(ds, diag(fmt.Sprintf("frontmatter: disallowed key %q", k)))
		}
	}
	switch n := utf8.RuneCountInString(s.Description); {
	case n == 0:
		ds = append(ds, diag("frontmatter: description is empty"))
	case n > DescriptionMaxRunes:
		ds = append(ds, diag(fmt.Sprintf(
			"frontmatter: description %d runes > %d", n, DescriptionMaxRunes)))
	}
	if reAngle.MatchString(s.Description) {
		ds = append(ds, diag(
			"frontmatter: description contains angle brackets (must be plain text)"))
	}
	return ds
}

// diag builds an error-severity frontmatter diagnostic. Category and Path are
// left empty so the diagnostic marshals as {severity, message} — the shape both
// tools already emit, keeping their machine output stable across this migration.
func diag(message string) finding.Diagnostic {
	return finding.Diagnostic{Severity: finding.SeverityError, Message: message}
}
