package skill_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/errs"
	"github.com/StevenACoffman/skillet/identity"
	"github.com/StevenACoffman/skillet/skill"
)

const sampleSkill = "---\nname: my-skill\ndescription: does a thing\ntags: [a, b]\n---\n# Body\n\nHello.\n"

// writeSkill creates <parent>/<name>/SKILL.md with content and returns the dir.
func writeSkill(t *testing.T, parent, name, content string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, skill.FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	return dir
}

func TestLoadAndParse(t *testing.T) {
	t.Parallel()
	dir := writeSkill(t, t.TempDir(), "my-skill", sampleSkill)
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Name != "my-skill" {
		t.Errorf("Name = %q, want my-skill", s.Name)
	}
	if s.Description != "does a thing" {
		t.Errorf("Description = %q, want %q", s.Description, "does a thing")
	}
	if !strings.Contains(s.Body, "Hello.") {
		t.Errorf("Body = %q, want it to contain the body text", s.Body)
	}
	wantKeys := []string{"description", "name", "tags"}
	if !reflect.DeepEqual(s.FrontmatterKeys, wantKeys) {
		t.Errorf("FrontmatterKeys = %v, want %v", s.FrontmatterKeys, wantKeys)
	}
	if s.Hash() != identity.Hash(s.Raw) {
		t.Error("Skill.Hash() must equal identity.Hash of the raw bytes")
	}
	if s.Bytes != len(sampleSkill) {
		t.Errorf("Bytes = %d, want %d", s.Bytes, len(sampleSkill))
	}
}

func TestLoadMissing(t *testing.T) {
	t.Parallel()
	_, err := skill.Load(t.TempDir()) // empty dir, no SKILL.md
	if err == nil {
		t.Fatal("want an error loading a dir with no SKILL.md")
	}
	if code := errs.ErrorCode(err); code != errs.ENOTFOUND {
		t.Fatalf("want ENOTFOUND code, got %q (%v)", code, err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message = %q, want it to mention \"not found\"", err)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	t.Parallel()
	dir := writeSkill(t, t.TempDir(), "plain", "# Just a body\n\nno frontmatter here.\n")
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "" || s.Description != "" {
		t.Errorf("expected empty name/desc, got %q/%q", s.Name, s.Description)
	}
	if !strings.HasPrefix(s.Body, "# Just a body") {
		t.Errorf("Body = %q, want the whole content as body", s.Body)
	}
}

func TestDiscover(t *testing.T) {
	t.Parallel()
	tree := t.TempDir()
	writeSkill(t, tree, "a", sampleSkill)
	writeSkill(t, tree, "b", sampleSkill)
	if err := os.MkdirAll(filepath.Join(tree, "c"), 0o755); err != nil { // dir without SKILL.md
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "top.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := skill.Discover(tree)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := []string{filepath.Join(tree, "a"), filepath.Join(tree, "b")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Discover = %v, want %v", got, want)
	}
}

func TestDiscoverRoots(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeSkill(t, filepath.Join(base, ".claude", "skills"), "alpha", sampleSkill)
	writeSkill(t, filepath.Join(base, ".cursor", "skills"), "beta", sampleSkill)
	// .codex/skills and .agents/skills are absent and must be skipped.
	got, err := skill.DiscoverRoots(base, skill.DefaultRoots())
	if err != nil {
		t.Fatalf("DiscoverRoots: %v", err)
	}
	want := []string{
		filepath.Join(base, ".claude", "skills", "alpha"),
		filepath.Join(base, ".cursor", "skills", "beta"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DiscoverRoots = %v, want %v", got, want)
	}
}

func TestSlug(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"Hello World", "hello-world"},
		{"a_b-c", "a-b-c"},
		{"UPPER", "upper"},
		{"trailing--", "trailing"},
		{"", "skill"},
		{"!!!", "skill"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := skill.Slug(tt.in); got != tt.want {
				t.Errorf("Slug(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if got := skill.Slug(skill.Slug(tt.in)); got != tt.want {
				t.Errorf("Slug not idempotent for %q: got %q", tt.in, got)
			}
		})
	}
}

func TestLoadRecordsMalformedFrontmatter(t *testing.T) {
	t.Parallel()
	// A quoted scalar followed by unquoted text — the shape that actually appears in
	// the book trees, and that YAML rejects.
	const malformed = "---\n" +
		"name: my-skill\n" +
		"description: does a thing\n" +
		`source_book: "Site Reliability Engineering" by Betsy Beyer` + "\n" +
		"---\n# Body\n\nHello.\n"

	dir := writeSkill(t, t.TempDir(), "my-skill", malformed)
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatalf("Load must still succeed: one malformed skill cannot stop a tree walk: %v", err)
	}
	if s.FrontmatterErr == nil {
		t.Fatal("FrontmatterErr must record why the frontmatter did not parse")
	}
	// The contract is that the cause is recorded, not that it takes any particular
	// shape — the message format belongs to the YAML library, so asserting on it
	// would couple this test to that choice.
	if s.FrontmatterErr.Error() == "" {
		t.Error("FrontmatterErr must carry a message explaining the failure")
	}
	// The split happens before the parse, so callers that only need the body still
	// get it. exegesis's index relies on this.
	if !strings.Contains(s.Body, "Hello.") {
		t.Errorf("Body must survive a frontmatter parse failure, got %q", s.Body)
	}
	if s.Raw != malformed {
		t.Error("Raw must be the file verbatim regardless of parse outcome")
	}
	// The fields are zero because nothing could be read, not because they are absent.
	if s.Name != "" || s.Description != "" || len(s.FrontmatterKeys) != 0 {
		t.Errorf("expected zero fields alongside FrontmatterErr, got %+v", s)
	}
}

func TestLoadLeavesFrontmatterErrNilWhenValid(t *testing.T) {
	t.Parallel()
	dir := writeSkill(t, t.TempDir(), "my-skill", sampleSkill)
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.FrontmatterErr != nil {
		t.Errorf("valid frontmatter must leave FrontmatterErr nil, got %v", s.FrontmatterErr)
	}
}

func TestParseDoesNotDisturbContentIdentity(t *testing.T) {
	t.Parallel()
	// Hash is the identity manifests are pinned with and must be over the bytes as
	// read. frontmatter.Split normalizes CRLF for what it returns; if that
	// normalization ever reached Raw, every hash in every stored manifest would move.
	dir := t.TempDir()
	raw := "---\r\nname: a\r\ndescription: d\r\n---\r\nbody\r\n"
	if err := os.WriteFile(filepath.Join(dir, skill.FileName), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Raw != raw {
		t.Errorf("Raw was normalized: %q", s.Raw)
	}
	if s.Bytes != len(raw) {
		t.Errorf("Bytes = %d, want %d (the bytes on disk)", s.Bytes, len(raw))
	}
	if got, want := s.Hash(), identity.Hash(raw); got != want {
		t.Errorf("Hash = %q, want %q — identity over the raw bytes", got, want)
	}
}

func TestParseReadsACRLFSkill(t *testing.T) {
	t.Parallel()
	// A Windows-authored skill must not look like one with no frontmatter, which is
	// how it would be reported: "description is empty" on a file that plainly has one.
	dir := t.TempDir()
	raw := "---\r\nname: alpha\r\ndescription: Use when x.\r\n---\r\n## R\r\nbody\r\n"
	if err := os.WriteFile(filepath.Join(dir, skill.FileName), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.FrontmatterErr != nil {
		t.Fatalf("frontmatter did not parse: %v", s.FrontmatterErr)
	}
	if s.Name != "alpha" || s.Description != "Use when x." {
		t.Errorf("got name=%q description=%q, want them read from the CRLF header",
			s.Name, s.Description)
	}
	if strings.Contains(s.Body, "---") {
		t.Errorf("delimiter leaked into the body: %q", s.Body)
	}
}
