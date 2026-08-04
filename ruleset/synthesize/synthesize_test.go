package synthesize_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset/synthesize"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestFillTemplateInjectsOrderedBlocks(t *testing.T) {
	t.Parallel()
	got, err := synthesize.FillTemplate("A\n{{RULESETS}}\nB", []synthesize.Input{
		{Title: "First", Body: "rule one"},
		{Title: "Second", Body: "rule two"},
	})
	if err != nil {
		t.Fatalf("FillTemplate: %v", err)
	}
	want := "A\n" +
		"<ruleset id=\"1\" source=\"First\">\nrule one\n</ruleset>\n" +
		"<ruleset id=\"2\" source=\"Second\">\nrule two\n</ruleset>" +
		"\nB"
	if got != want {
		t.Errorf("FillTemplate mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestFillTemplateMissingMarkerIsError(t *testing.T) {
	t.Parallel()
	_, err := synthesize.FillTemplate("no marker", []synthesize.Input{{Title: "x", Body: "y"}})
	if err == nil || !strings.Contains(err.Error(), synthesize.Marker) {
		t.Fatalf("got %v, want an error naming the missing %s marker", err, synthesize.Marker)
	}
}

func TestFillTemplateNoInputsIsError(t *testing.T) {
	t.Parallel()
	if _, err := synthesize.FillTemplate(synthesize.Marker, nil); err == nil {
		t.Fatal("FillTemplate with no inputs returned nil; want an error")
	}
}

func TestLoadInputsSortsSkipsAndTitles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b_rules.md"), "# Beta Rules\n\nbeta\n")
	writeFile(t, filepath.Join(dir, "a_rules.md"), "no heading here\n")
	writeFile(t, filepath.Join(dir, "notes.md"), "ignored: not a *_rules.md file\n")

	inputs, err := synthesize.LoadInputs(dir)
	if err != nil {
		t.Fatalf("LoadInputs: %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("got %d inputs, want 2 (the non-_rules.md file is skipped)", len(inputs))
	}
	// os.ReadDir is name-sorted, so a_rules.md precedes b_rules.md.
	if inputs[0].Title != "A" { // no H1 -> filename fallback: "a" -> "A"
		t.Errorf("inputs[0].Title = %q, want %q (filename fallback)", inputs[0].Title, "A")
	}
	if inputs[1].Title != "Beta Rules" { // first H1 wins
		t.Errorf("inputs[1].Title = %q, want %q (from H1)", inputs[1].Title, "Beta Rules")
	}
}

func TestLoadInputsMissingDirIsError(t *testing.T) {
	t.Parallel()
	if _, err := synthesize.LoadInputs(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("LoadInputs of a missing directory returned nil; want an error")
	}
}
