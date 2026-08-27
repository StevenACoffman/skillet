// Package skill loads and parses Agent Skills ("SKILL.md") files — the unified
// loader shared by exegesis and skillsaw. A skill is a directory containing a
// SKILL.md in the Anthropic Agent Skills format: YAML frontmatter (name and
// description, optionally other keys) delimited by "---" lines, followed by a
// Markdown body. Parsing is pure once the bytes are read; only Load and the
// Discover functions touch the filesystem.
package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/StevenACoffman/skillet/errs"
	"github.com/StevenACoffman/skillet/frontmatter"
	"github.com/StevenACoffman/skillet/fsutil"
	"github.com/StevenACoffman/skillet/identity"
)

// FileName is the required filename inside a skill directory.
const FileName = "SKILL.md"

// Skill is a parsed SKILL.md and its location.
type Skill struct {
	Dir             string   // skill directory
	Path            string   // <Dir>/SKILL.md
	Name            string   // frontmatter name
	Description     string   // frontmatter description
	FrontmatterKeys []string // top-level frontmatter keys, sorted (for lint)
	Frontmatter     string   // raw YAML frontmatter block (between the --- lines)
	Body            string   // markdown body after the frontmatter

	// Lineage is the recognised `lineage:` declaration, LineageUnset when absent *or*
	// unrecognised, so a caller that ignores LineageRaw gets the strictest treatment.
	//
	// LineageRaw is the value as written, kept so an unrecognised one can be reported
	// verbatim rather than described. Unrecognised is therefore
	// `LineageRaw != "" && Lineage == LineageUnset` — derived rather than a second
	// boolean, because a bool's zero value would say "unrecognised" on the
	// FrontmatterErr path, where nothing was parsed and there is nothing to report.
	// That is the trap the note on FrontmatterErr describes, one field over.
	//
	// A bad value is not a load failure, for the same reason FrontmatterErr is not: one
	// malformed skill must not stop a caller walking a tree.
	Lineage    Lineage
	LineageRaw string
	Raw        string // full file contents
	Bytes      int    // byte size of Raw

	// FrontmatterErr is the error from parsing the YAML frontmatter, or nil when it
	// parsed. When it is non-nil, Name, Description and FrontmatterKeys are zero
	// because nothing could be read out of the block — they are not evidence that
	// those fields are absent from the file. A linter that reports "description is
	// empty" without checking this sends the reader to the wrong line.
	//
	// The underlying YAML error is kept verbatim: it locates the offending line and
	// column, which is the part a reader needs, and nothing classifies on it.
	FrontmatterErr error
}

// Load reads and parses <dir>/SKILL.md. A missing SKILL.md is translated at this
// boundary to an errs.Error with code ENOTFOUND (classify it via errs.ErrorCode);
// the external os.ErrNotExist is not propagated. Any other read error is wrapped
// with Op "skill.Load".
func Load(dir string) (*Skill, error) {
	p := filepath.Join(dir, FileName)
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &errs.Error{Code: errs.ENOTFOUND, Message: "skill not found: " + p}
		}
		return nil, &errs.Error{Op: "skill.Load", Err: err}
	}
	s := &Skill{Dir: dir, Path: p, Raw: string(b), Bytes: len(b)}
	s.parse()
	return s, nil
}

// Hash is the content identity of the skill: identity.Hash of its raw bytes.
// It is byte-identical to the hash exegesis and skillsaw pin manifests with.
func (s *Skill) Hash() string { return identity.Hash(s.Raw) }

// DefaultRoots returns the runtime-neutral directories a multi-root discovery
// scans. The darwin source hard-codes .claude/skills, but runtime neutrality
// requires covering every skills-compatible runtime.
func DefaultRoots() []string {
	return []string{
		".claude/skills",
		".cursor/skills",
		".codex/skills",
		".agents/skills",
	}
}

// Discover returns every immediate subdirectory of tree that contains a
// SKILL.md, sorted by name. It errors if tree cannot be read. This is the
// single-tree walk the exegesis gates iterate over.
func Discover(tree string) ([]string, error) {
	subs, err := fsutil.SubdirsContaining(os.DirFS(tree), ".", FileName)
	if err != nil {
		return nil, fmt.Errorf("discover %s: %w", tree, err)
	}
	return joinFS(tree, subs), nil
}

// DiscoverRoots returns every skill directory under any of roots (relative to
// base), skipping roots absent in this environment and deduplicating. This is
// the multi-runtime scan skillsaw's "--all" performs.
func DiscoverRoots(base string, roots []string) ([]string, error) {
	subs, err := fsutil.SubdirsContainingAny(os.DirFS(base), roots, FileName)
	if err != nil {
		return nil, fmt.Errorf("discover roots under %s: %w", base, err)
	}
	return joinFS(base, subs), nil
}

// Slug normalizes a string to the Agent Skills slug form: lowercase, every run
// of non-[a-z0-9] becomes one hyphen, leading and trailing hyphens are trimmed,
// and the result is capped at 64 runes; an empty result becomes "skill". It is
// idempotent: Slug(Slug(x)) == Slug(x).
//
// It does not NFKD-fold accents (é stays a separator, not "e"), avoiding an
// x/text dependency; skill names are ASCII kebab-case in practice.
func Slug(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if r := []rune(out); len(r) > 64 {
		out = strings.TrimRight(string(r[:64]), "-")
	}
	if out == "" {
		return "skill"
	}
	return out
}

// joinFS turns fs.FS slash paths (relative to base) into OS paths under base.
func joinFS(base string, subs []string) []string {
	dirs := make([]string, 0, len(subs))
	for _, s := range subs {
		dirs = append(dirs, filepath.Join(base, filepath.FromSlash(s)))
	}
	return dirs
}

func (s *Skill) parse() {
	// Raw is left exactly as read: Hash is the content identity over those bytes, and
	// frontmatter.Split normalizes line endings only for what it returns.
	s.Frontmatter, s.Body = frontmatter.Split(s.Raw)

	// Ordered/unordered does not matter for the allowlist check; a map suffices
	// and also yields the top-level keys exegesis lint needs.
	var fields map[string]any
	if err := yaml.Unmarshal([]byte(s.Frontmatter), &fields); err != nil {
		// Record why, rather than leaving Name/Description empty and letting a
		// caller mistake the symptom for the defect. Load still succeeds: one
		// malformed skill must not stop a caller walking a whole tree.
		s.FrontmatterErr = err
		return
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s.FrontmatterKeys = keys
	s.Name = strings.TrimSpace(asString(fields["name"]))
	s.Description = strings.TrimSpace(asString(fields["description"]))
	s.LineageRaw = strings.TrimSpace(asString(nested(fields, MetadataKey, LineageKey)))
	s.Lineage, _ = ParseLineage(s.LineageRaw)
}

// nested reads fields[outer][inner], tolerating either map key type go-yaml may produce
// and any shape that is not a map at all. A declaration in the wrong shape reads as
// absent, which is graded strictly rather than leniently.
func nested(fields map[string]any, outer, inner string) any {
	switch m := fields[outer].(type) {
	case map[string]any:
		return m[inner]
	case map[any]any:
		return m[inner]
	default:
		return nil
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
