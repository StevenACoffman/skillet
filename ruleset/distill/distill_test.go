package distill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset/distill"
)

const tmpl = "Distill this source: <source>{{SOURCE_CONTENT}}</source>\n" +
	"into these rules: <destination>{{DESTINATION_CONTENT}}</destination>\n"

func TestFillTemplateMissingPlaceholder(t *testing.T) {
	t.Parallel()
	if _, err := distill.FillTemplate("no placeholders here", "s", "d"); err == nil {
		t.Fatal("a template missing the placeholders must error")
	}
	if _, err := distill.FillTemplate(
		"only <source>{{SOURCE_CONTENT}}</source>",
		"s",
		"d",
	); err == nil {
		t.Fatal("a template missing the destination placeholder must error")
	}
}

func TestFillTemplate(t *testing.T) {
	t.Parallel()
	got, err := distill.FillTemplate(tmpl, "[T](t.md)", "[T Rules](t_rules.md)")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "{{") {
		t.Errorf("placeholders not replaced: %q", got)
	}
	if !strings.Contains(got, "[T](t.md)") || !strings.Contains(got, "[T Rules](t_rules.md)") {
		t.Errorf("links not substituted: %q", got)
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A source with an H1 title, a source without one, plus files that must be skipped.
	write(t, filepath.Join(root, "crud.md"), "# CRUD Patterns\n\nbody\n")
	write(t, filepath.Join(root, "wtf-dial.md"), "no heading, title from stem\n")
	write(t, filepath.Join(root, "crud_rules.md"), "should be skipped\n")
	write(t, filepath.Join(root, "crud_prompt.md"), "should be skipped\n")

	out := filepath.Join(root, "prompts")
	written, err := distill.Generate(tmpl, root, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("wrote %d prompts, want 2 (rules/prompt files skipped): %v", len(written), written)
	}

	crudPrompt := readFile(t, filepath.Join(out, "crud_prompt.md"))
	if !strings.Contains(crudPrompt, "[CRUD Patterns](") {
		t.Errorf("crud prompt missing H1-derived source link:\n%s", crudPrompt)
	}
	if !strings.Contains(crudPrompt, "[CRUD Patterns Rules](") {
		t.Errorf("crud prompt missing rules link:\n%s", crudPrompt)
	}
	if !strings.Contains(crudPrompt, "crud_rules.md") {
		t.Errorf("crud prompt rules link should point at crud_rules.md:\n%s", crudPrompt)
	}

	dialPrompt := readFile(t, filepath.Join(out, "wtf-dial_prompt.md"))
	if !strings.Contains(dialPrompt, "[Wtf Dial](") { // title derived from the stem
		t.Errorf("dial prompt missing stem-derived title:\n%s", dialPrompt)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
