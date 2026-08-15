package markdown_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/markdown"
)

func TestParseSections(t *testing.T) {
	t.Parallel()
	body := "# Title\n\nIntro paragraph that is well over twenty characters long.\n\n" +
		"## Steps\n\n1. first\n2. second\n"
	doc := markdown.Parse(body)
	if len(doc.Sections) != 2 {
		t.Fatalf("want 2 sections, got %d: %+v", len(doc.Sections), doc.Sections)
	}
	if doc.Sections[0].Title != "Title" || doc.Sections[0].Level != 1 {
		t.Errorf("section0 = %+v, want Title/level 1", doc.Sections[0])
	}
	if doc.Sections[1].Title != "Steps" || doc.Sections[1].Level != 2 {
		t.Errorf("section1 = %+v, want Steps/level 2", doc.Sections[1])
	}
	if !doc.HasOrderedList {
		t.Error("HasOrderedList = false, want true")
	}
}

func TestParseCodeFenceNotHeading(t *testing.T) {
	t.Parallel()
	// The "#" inside the fence is a shell comment, not a Markdown heading — the
	// whole reason for parsing with an AST instead of regexes.
	body := "# Real Heading\n\n```\n# not a heading, just a shell comment\n```\n"
	doc := markdown.Parse(body)
	if len(doc.Sections) != 1 {
		t.Fatalf("want 1 section, got %d: %+v", len(doc.Sections), doc.Sections)
	}
	if doc.Sections[0].Title != "Real Heading" {
		t.Errorf("section title = %q, want %q", doc.Sections[0].Title, "Real Heading")
	}
	if strings.Contains(doc.Prose, "not a heading") {
		t.Errorf("Prose should blank code content, got: %q", doc.Prose)
	}
}

func TestParseLinks(t *testing.T) {
	t.Parallel()
	doc := markdown.Parse("See [the guide](docs/guide.md) and run `some-tool`.\n")
	var hasLink, hasSpan bool
	for _, l := range doc.Links {
		switch l {
		case "docs/guide.md":
			hasLink = true
		case "some-tool":
			hasSpan = true
		}
	}
	if !hasLink || !hasSpan {
		t.Fatalf("Links = %v, want docs/guide.md and some-tool", doc.Links)
	}
}

func TestParseUnorderedListIsNotOrdered(t *testing.T) {
	t.Parallel()
	if markdown.Parse("- a\n- b\n").HasOrderedList {
		t.Error("unordered list reported as ordered")
	}
}

func TestParseHasCodeBlock(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		body string
		want bool
	}{
		"fenced":   {"Run it:\n\n```sh\nls -l\n```\n", true},
		"indented": {"Run it:\n\n    ls -l\n", true},
		// The distinction the whole field turns on. Consumers read this as "the artifact
		// executes something"; naming `foo` in backticks demonstrates nothing runnable, and
		// counting spans would make nearly every prose document look executable.
		"code span alone is not a block": {"Call `Parse` on the body, then read `Prose`.\n", false},
		"prose only":                     {"## Boundary\n\n- do not do this\n", false},
		"empty":                          {"", false},
		// A fence nested in a list item is still a code block; the walk must not stop at
		// the top level of the tree.
		"fenced inside a list item": {"1. first:\n\n   ```go\n   x := 1\n   ```\n", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := markdown.Parse(tc.body).HasCodeBlock; got != tc.want {
				t.Errorf("HasCodeBlock = %v, want %v", got, tc.want)
			}
		})
	}
}
