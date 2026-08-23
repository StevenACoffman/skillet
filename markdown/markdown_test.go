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

// assertProseAlignsWith checks the one property Prose's blanking must hold: same length,
// and every byte either untouched or replaced by a space, so an index into Prose
// addresses the same byte of body.
func assertProseAlignsWith(t *testing.T, body string) {
	t.Helper()
	prose := markdown.Parse(body).Prose
	if len(prose) != len(body) {
		t.Fatalf("length changed: body %d, prose %d", len(body), len(prose))
	}
	for i := range len(body) {
		if prose[i] != body[i] && prose[i] != ' ' {
			t.Fatalf("byte %d is %q, neither the original %q nor a blank",
				i, prose[i], body[i])
		}
	}
	// Newlines survive blanking, so a line number derived from either agrees.
	if got, want := strings.Count(prose, "\n"), strings.Count(body, "\n"); got != want {
		t.Errorf("newline count changed: body %d, prose %d", want, got)
	}
}

// TestProsePreservesOffsets pins the property Prose's blanking depends on and nothing
// else asserts: prose() copies the source and overwrites in place, so Prose is
// byte-for-byte the same length and an index into it addresses the same byte of the
// original. Three consumers would rely on this the moment skilllens.Span carried
// offsets (see that type's doc), and it is exactly the kind of property a well-meaning
// rewrite of prose() -- building a new buffer instead of masking one -- would silently
// break.
func TestProsePreservesOffsets(t *testing.T) {
	for name, body := range map[string]string{
		"code span":       "# T\n\nIf `go test ./...` fails, stop.\n",
		"fenced block":    "# T\n\ntext\n\n```sh\nif x; then fail; fi\n```\n\nafter\n",
		"multibyte prose": "# T\n\n如果失败，请重试 with `cmd` after.\n",
		"span at end":     "run `x`",
		"no code at all":  "# T\n\nplain prose only.\n",
		"adjacent spans":  "a `one` b `two` c\n",
		"empty":           "",
	} {
		t.Run(name, func(t *testing.T) { assertProseAlignsWith(t, body) })
	}
}

// TestFenceInsideHTMLBlockIsBlanked is the guard on the second parse, and it asserts
// behaviour rather than mechanism so it fails however the fix breaks.
//
// A `<Good>` tag with no blank line after it opens a CommonMark type-7 HTML block that
// runs to the next blank line, so the fence inside it is not a FencedCodeBlock and
// nothing blanks it. `<Good>`/`<Bad>` wrappers around examples are a documented
// convention in the corpus this package reads, and the live hit was a TypeScript
// example throwing `new Error('fail')` scored as the skill's own failure handling.
func TestFenceInsideHTMLBlockIsBlanked(t *testing.T) {
	t.Parallel()
	const code = "throw new Error('fail')"
	cases := map[string]struct {
		body      string
		wantProse bool // should the code survive into Prose?
	}{
		"a bare fence": {
			"# S\n\n```ts\n" + code + "\n```\n", false,
		},
		"a fence inside an HTML block": {
			"# S\n\n<Good>\n```ts\n" + code + "\n```\n</Good>\n", false,
		},
		"a fence inside an HTML block, blank line after the tag": {
			"# S\n\n<Good>\n\n```ts\n" + code + "\n```\n\n</Good>\n", false,
		},
		"a fence inside a nested HTML block": {
			"# S\n\n<div><Good>\n```ts\n" + code + "\n```\n</Good></div>\n", false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := markdown.Parse(tc.body)
			if strings.Contains(got.Prose, code) != tc.wantProse {
				t.Errorf("code in Prose = %v, want %v\nProse: %q",
					!tc.wantProse, tc.wantProse, got.Prose)
			}
			if !got.HasCodeBlock {
				t.Error("HasCodeBlock is false for a document containing a fence")
			}
		})
	}
}

// TestProseInsideAnHTMLBlockSurvives is the case that decided the design. Blanking the
// whole HTML block would have fixed the fence in one line and destroyed this: a
// `<Good>` wrapper around an instruction is an instruction, and it reaches Prose
// correctly today.
func TestProseInsideAnHTMLBlockSurvives(t *testing.T) {
	t.Parallel()
	const instruction = "Always validate input"
	got := markdown.Parse("# S\n\n<Good>\n" + instruction + "\n</Good>\n")

	if !strings.Contains(got.Prose, instruction) {
		t.Errorf("an instruction inside an HTML block was lost from Prose: %q", got.Prose)
	}
}

// TestTheSecondParseKeepsGFM. The parser is replaced rather than configured, so the
// extension has to survive that — a table parsed as paragraphs would change Prose
// without changing any test that looks at Sections.
func TestTheSecondParseKeepsGFM(t *testing.T) {
	t.Parallel()
	got := markdown.Parse("# S\n\n| a | b |\n| - | - |\n| `x y` | 2 |\n")

	// A code span inside a table cell is blanked only if the table parsed as a table.
	if strings.Contains(got.Prose, "x y") {
		t.Errorf("a code span in a table cell was not blanked: %q", got.Prose)
	}
}
