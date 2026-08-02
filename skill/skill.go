// Package skill loads and parses Agent Skills ("SKILL.md") files — the unified
// loader shared by exegesis and skillsaw. A skill is a directory containing a
// SKILL.md in the Anthropic Agent Skills format: YAML frontmatter (name and
// description, optionally other keys) delimited by "---" lines, followed by a
// Markdown body. Parsing is pure once the bytes are read; only Load and the
// Discover functions touch the filesystem.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"

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
	Raw             string   // full file contents
	Bytes           int      // byte size of Raw
}

// Load reads and parses <dir>/SKILL.md. The error wraps os.ErrNotExist when the
// file is absent, so callers can errors.Is it.
func Load(dir string) (*Skill, error) {
	p := filepath.Join(dir, FileName)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("load skill %s: %w", dir, err)
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
	text := strings.ReplaceAll(s.Raw, "\r\n", "\n")
	s.Frontmatter, s.Body = splitFrontmatter(text)

	// Ordered/unordered does not matter for the allowlist check; a map suffices
	// and also yields the top-level keys exegesis lint needs.
	var fields map[string]any
	if err := yaml.Unmarshal([]byte(s.Frontmatter), &fields); err != nil {
		return // malformed frontmatter leaves Name/Description empty; lint flags it
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s.FrontmatterKeys = keys
	s.Name = strings.TrimSpace(asString(fields["name"]))
	s.Description = strings.TrimSpace(asString(fields["description"]))
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// splitFrontmatter separates a leading "---"-delimited YAML block from the body.
// It returns ("", text) when there is no frontmatter and ("", rest) when the
// opening delimiter has no matching close.
func splitFrontmatter(text string) (frontmatter, body string) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", rest
	}
	after := rest[end+len("\n---"):]
	if nl := strings.IndexByte(after, '\n'); nl >= 0 {
		body = after[nl+1:]
	}
	return rest[:end], body
}
