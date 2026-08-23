// Package markdown parses a SKILL.md body into a small structured view, using
// goldmark (the established Go Markdown parser) instead of hand-rolled regexes.
// Doing the parsing properly means code fences, headings, lists, tables, links,
// and code spans are AST facts — so, for example, a "#" comment inside a
// ```code block``` is no longer mistaken for a heading.
package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Section is a Markdown heading and a count of its concrete body content
// (list items, table rows, and substantial paragraphs) up to the next heading of
// the same or higher level — sub-headings and their content are included.
type Section struct {
	Level int
	Title string
	Units int
}

// Doc is the parsed view of a SKILL.md body.
type Doc struct {
	Sections []Section
	Prose    string   // body text with code blocks and code spans blanked
	Links    []string // link destinations and code-span contents (ref candidates)

	// CodeSpans is the contents of every inline code span, in document order.
	//
	// Links already carries these, mixed with link destinations, because a consumer
	// resolving references wants both. A consumer asking a different question — which
	// of these is a command this document claims its repository supports — cannot use
	// that slice without treating destinations as candidates. Separate rather than
	// splitting Links, which four consumers read.
	CodeSpans      []string
	HasOrderedList bool
	// HasCodeBlock reports a fenced or indented code block, never an inline code span.
	//
	// It is a derived fact, and consumers use it to decide whether a check applies at
	// all -- skillsaw's dim 3 and adh's failure-handling factor both ask it before
	// deducting. An obligation comes with that use, and it is not enforceable here:
	// a consumer that suppresses anything on this predicate must say so and say why.
	// A check that silently declines is indistinguishable from one that found nothing.
	// What gets suppressed is the consumer's choice -- a deduction, a whole check, or
	// nothing at all; the reason is not.
	HasCodeBlock bool
}

// Parse parses a Markdown body. It is pure: same input, same Doc. GFM is enabled
// so tables are recognized.
//
// It parses twice, over the same bytes, and the second parse exists for one reason:
// **a fenced code block written inside an HTML block is not a code block.** A
// `<Good>` tag with no blank line after it opens a CommonMark type-7 HTML block that
// runs to the next blank line, swallowing any fence inside it — so `prose` never sees
// a FencedCodeBlock to blank, and the example's own code reaches Prose as though the
// skill had instructed it. `<Good>`/`<Bad>` wrappers around examples are a documented
// convention, so this is the common case rather than a corner.
//
// The second parse removes the HTML block parser, which turns those fences back into
// fences. It is used for Prose and HasCodeBlock — the two answers about *what code
// this document contains* — and deliberately not for Sections, Links, or
// HasOrderedList, whose answers are about document structure, where an HTML block
// genuinely is one block.
//
// Blanking the whole HTML block was the obvious one-line alternative and is wrong:
// `<Good>Always validate input.</Good>` is a real instruction that reaches Prose
// correctly today, and blanking the block would destroy it. Scanning the block's raw
// lines for fence delimiters was the other, and it is a second fence implementation
// in the package whose first sentence says it does not have one.
func Parse(body string) *Doc {
	src := []byte(body)
	root := goldmark.New(goldmark.WithExtensions(extension.GFM)).
		Parser().Parse(text.NewReader(src))
	code := codeParser().Parser().Parse(text.NewReader(src))
	blocks := children(root)
	return &Doc{
		Sections:       sections(blocks, src),
		Prose:          prose(code, src),
		Links:          links(root, src),
		CodeSpans:      codeSpans(root, src),
		HasOrderedList: hasOrderedList(root),
		HasCodeBlock:   hasCodeBlock(code),
	}
}

// codeParser is goldmark with the HTML block parser removed, so a fence inside an
// HTML block parses as a fence.
//
// Requires: nothing.
// Ensures: a parser producing the same offsets as the default one over the same
// source — it removes a block parser and adds none, so every segment still indexes
// the original bytes. Prose depends on that and TestProsePreservesOffsets pins it.
//
// The parser is **constructed** rather than configured, which is not a style choice:
// `parser.WithBlockParsers` passed to `goldmark.New` *appends* to the defaults, so
// there is no option that removes one. Replacing the parser means restating the
// inline parsers and paragraph transformers, which is why they appear here unchanged.
//
// The HTML block parser is found by pointer identity, because
// `parser.NewHTMLBlockParser` returns a package-level singleton. The reader's first
// guess is a type switch, and that is why this comment exists: the concrete type is
// unexported, so a type switch would have to match on a name string and would
// silently match nothing the day goldmark renames it.
//
// Pointer identity can fail the same way — if a future goldmark stops returning a
// singleton, nothing here removes anything and Prose quietly regresses. The guard is
// TestFenceInsideHTMLBlockIsBlanked, which asserts the *behaviour* rather than the
// mechanism: it fails whether the filter missed, the parser changed, or the fix was
// removed. A length assertion would only catch the first.
func codeParser() goldmark.Markdown {
	htmlBlocks := parser.NewHTMLBlockParser()
	defaults := parser.DefaultBlockParsers()
	kept := make([]util.PrioritizedValue, 0, len(defaults))
	for _, p := range defaults {
		if p.Value != htmlBlocks {
			kept = append(kept, p)
		}
	}
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParser(parser.NewParser(
			parser.WithBlockParsers(kept...),
			parser.WithInlineParsers(parser.DefaultInlineParsers()...),
			parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
		)),
	)
}

func children(root ast.Node) []ast.Node {
	var out []ast.Node
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, c)
	}
	return out
}

// sections builds one Section per heading, counting content units until the next
// heading of the same or higher level.
func sections(blocks []ast.Node, src []byte) []Section {
	var out []Section
	for i, n := range blocks {
		h, ok := n.(*ast.Heading)
		if !ok {
			continue
		}
		units := 0
		for j := i + 1; j < len(blocks); j++ {
			if h2, ok := blocks[j].(*ast.Heading); ok && h2.Level <= h.Level {
				break
			}
			units += unitCount(blocks[j], src)
		}
		out = append(out, Section{Level: h.Level, Title: nodeText(n, src), Units: units})
	}
	return out
}

// unitCount returns how many concrete content points a block contributes: one per
// list item, one per table row, and one for a paragraph with substantial text.
func unitCount(n ast.Node, src []byte) int {
	switch n.Kind() {
	case ast.KindList:
		return countChildren(n, ast.KindListItem)
	case extast.KindTable:
		return countChildren(n, extast.KindTableRow) + countChildren(n, extast.KindTableHeader)
	case ast.KindParagraph, ast.KindTextBlock:
		if len([]rune(nodeText(n, src))) > 20 {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func countChildren(n ast.Node, kind ast.NodeKind) int {
	count := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == kind {
			count++
		}
	}
	return count
}

// hasOrderedList reports whether the document contains any ordered list (a signal
// of a step-by-step workflow).
func hasOrderedList(root ast.Node) bool {
	found := false
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if l, ok := n.(*ast.List); ok && l.IsOrdered() {
				found = true
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return found
}

// hasCodeBlock reports whether the document contains a block-level code block,
// fenced or indented.
//
// An inline code span is deliberately not one: a document that merely names `foo`
// in backticks demonstrates nothing runnable, and counting spans would make almost
// every prose document look executable.
//
// What a caller concludes from this is the caller's own: skillsaw and adh read it as
// "this artifact executes something, so a missing failure mechanism is a defect
// rather than a category error", but that reading is theirs, not this package's.
func hasCodeBlock(root ast.Node) bool {
	found := false
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n.Kind() {
			case ast.KindFencedCodeBlock, ast.KindCodeBlock:
				found = true
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return found
}

// links collects Markdown link destinations and code-span contents — the
// candidate resource references a skill may point at.
func links(root ast.Node, src []byte) []string {
	var out []string
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindLink:
			out = append(out, string(n.(*ast.Link).Destination))
		case ast.KindCodeSpan:
			out = append(out, nodeText(n, src))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return out
}

// codeSpans collects inline code-span contents in document order.
func codeSpans(root ast.Node, src []byte) []string {
	var out []string
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindCodeSpan {
			out = append(out, nodeText(n, src))
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return out
}

// prose returns the body with the contents of code blocks and code spans blanked
// (replaced by spaces, newlines preserved), so prose-quality checks do not count
// phrases inside code examples or backtick-quoted anti-patterns.
func prose(root ast.Node, src []byte) string {
	buf := append([]byte(nil), src...)
	blank := func(start, stop int) {
		for i := start; i < stop && i < len(buf); i++ {
			if buf[i] != '\n' {
				buf[i] = ' '
			}
		}
	}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindFencedCodeBlock, ast.KindCodeBlock:
			lines := n.Lines()
			for i := range lines.Len() {
				seg := lines.At(i)
				blank(seg.Start, seg.Stop)
			}
		case ast.KindCodeSpan:
			blankCodeSpan(n, blank)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return string(buf)
}

func blankCodeSpan(n ast.Node, blank func(start, stop int)) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			blank(t.Segment.Start, t.Segment.Stop)
		}
	}
}

// nodeText concatenates the readable text of a subtree: Text/String literals and
// code-span contents. Used for heading titles and paragraph length.
func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := node.(type) {
		case *ast.Text:
			b.Write(v.Segment.Value(src))
		case *ast.String:
			b.Write(v.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
