package naming_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/StevenACoffman/skillet/naming"
)

func TestTitle(t *testing.T) {
	t.Parallel()
	// The first four are the original cases with the capitals removed. The rest are the
	// cases the 2026-09-07 decision was taken on, kept here so a reader can see *why* the
	// function stopped capitalizing rather than only that it did.
	tests := []struct{ in, want string }{
		{"my-source_file", "my source file"},
		{"crud", "crud"},
		{"wtf-dial", "wtf dial"},
		{"", ""},

		// An acronym. 76 of 342 slugs carry one; the old code wrote "Composite Slo".
		{"composite-slo", "composite slo"},
		// A project name. The old code wrote "Climax Cli Scaffold" -- wrong twice over.
		{"climax-cli-scaffold", "climax cli scaffold"},
		// Internal capitals are unrecoverable from a lowercase stem either way, and the
		// point is that the function no longer guesses: "Skilllens" was never right.
		{"skilllens-dimensions", "skilllens dimensions"},
		// Surrounding and repeated separators. The old split kept empty fields, so this
		// returned " a b " with the spaces attached.
		{"-a__b-", "a b"},
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
	// Lower case, because the fallback goes through Title and Title stopped guessing
	// capitalization. The contrast with the H1 case above is the whole point: an authored
	// title keeps its author's casing, a derived one has none to keep.
	if got, err := naming.TitleFromFile(noH1); err != nil || got != "wtf dial" {
		t.Errorf("TitleFromFile(noH1) = %q, %v; want %q (from stem)", got, err, "wtf dial")
	}
}
