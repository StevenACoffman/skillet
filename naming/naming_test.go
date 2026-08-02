package naming_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/StevenACoffman/skillet/naming"
)

func TestTitle(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"my-source_file", "My Source File"},
		{"crud", "Crud"},
		{"wtf-dial", "Wtf Dial"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := naming.Title(tt.in); got != tt.want {
			t.Errorf("Title(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRulesFilename(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"My-Source File.md", "my_source_file_rules.md"},
		{"crud.md", "crud_rules.md"},
		{"a_b.md", "a_b_rules.md"},
	}
	for _, tt := range tests {
		if got := naming.RulesFilename(tt.in); got != tt.want {
			t.Errorf("RulesFilename(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPromptFilename(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"self-refinement.md", "self-refinement_prompt.md"},
		{"crud.md", "crud_prompt.md"},
	}
	for _, tt := range tests {
		if got := naming.PromptFilename(tt.in); got != tt.want {
			t.Errorf("PromptFilename(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTitleFromMarkdown(t *testing.T) {
	t.Parallel()
	if got := naming.TitleFromMarkdown("# Hello\n\nbody\n"); got != "Hello" {
		t.Errorf("TitleFromMarkdown = %q, want Hello", got)
	}
	if got := naming.TitleFromMarkdown("no heading here\n"); got != "" {
		t.Errorf("TitleFromMarkdown = %q, want empty", got)
	}
	// "## Sub" is not an H1; the first real "# " heading wins.
	if got := naming.TitleFromMarkdown("## Sub\n\n# Real\n"); got != "Real" {
		t.Errorf("TitleFromMarkdown = %q, want Real", got)
	}
}

func TestTitleFromFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	withH1 := filepath.Join(dir, "crud.md")
	if err := os.WriteFile(withH1, []byte("# CRUD Patterns\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := naming.TitleFromFile(withH1); err != nil || got != "CRUD Patterns" {
		t.Errorf("TitleFromFile(withH1) = %q, %v; want %q", got, err, "CRUD Patterns")
	}

	noH1 := filepath.Join(dir, "wtf-dial.md")
	if err := os.WriteFile(noH1, []byte("just prose, no heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := naming.TitleFromFile(noH1); err != nil || got != "Wtf Dial" {
		t.Errorf("TitleFromFile(noH1) = %q, %v; want %q (from stem)", got, err, "Wtf Dial")
	}
}
