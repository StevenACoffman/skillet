// Package textnorm folds the differences between two copies of the same words that
// nobody means as differences: a run of whitespace where a line happened to wrap, and
// the typographic characters a book, a plain-text extraction and a Markdown file each
// spell their own way.
//
// It exists because several guards need the same folding and must not disagree about it.
// `quotecheck` compares a quotation against the source it claims to come from, `a2check`
// compares a merged skill's language signals against its sources', and canonizer's
// `verify.Provenance` asks whether a rule's anchor appears in its source; a quotation that
// matched in one and not another would be a defect nobody could explain from the output.
//
// That was not hypothetical, and is why this lives here rather than in exegesis. canonizer
// folded whitespace only, so an anchor copied from a source containing a curly apostrophe
// blocked there while the identical passage passed `quotecheck`. 231 of the 233 corpus
// skills contain typographic characters, so the disagreement was the normal case.
//
// Case is deliberately preserved: a quotation differing only in case is a different
// quotation, and folding it would widen every guard that uses this.
//
// Everything here is pure.
package textnorm

import (
	"regexp"
	"strings"

	"github.com/StevenACoffman/skillet/hidden"
)

// reSpace matches any run of whitespace, including the newlines a quotation is wrapped
// across in Markdown but not in an extracted source text.
var reSpace = regexp.MustCompile(`\s+`)

// Fold reduces text to the form two copies of it are compared in: invisible codepoints
// removed, whitespace runs collapsed to one space, typographic characters folded to
// ASCII, ends trimmed.
//
// "Invisible" is whatever hidden.Classify lists, so this and the hidden-character scan
// agree on what a reader cannot see. Folding removed only U+200B before, so a quotation
// carrying a soft hyphen was rejected by the scan yet compared as different text here.
//
// It deliberately does not change case. Case can carry meaning in a quotation, and the
// caller that wants case-insensitivity can say so; a folder that quietly lower-cased
// would leave no way back.
//
// Ensures: Fold(Fold(s)) == Fold(s); it is pure.
func Fold(s string) string {
	visible := hidden.Strip(s).Text
	return strings.TrimSpace(reSpace.ReplaceAllString(typographic().Replace(visible), " "))
}

// typographic folds the characters a book and its plain-text extraction are most likely
// to disagree about. A guard that fired on every curly apostrophe would not get run.
//
// The non-breaking space is written as an escape on purpose: a literal one in source is
// invisible, so a later reader could neither tell what the entry does nor notice if it
// were silently edited away. Zero-width characters are not here; Fold strips them first.
//
// Built per call rather than kept in a package variable: it is microseconds against
// reading whole books off disk, and a package-level Replacer is shared state.
func typographic() *strings.Replacer {
	return strings.NewReplacer(
		"\u2018", "'", "\u2019", "'", // curly single quotes
		"\u201c", `"`, "\u201d", `"`, // curly double quotes
		"\u2013", "-", "\u2014", "-", // en and em dash
		"\u2026", "...", // ellipsis
		"\u00a0", " ", // non-breaking space
	)
}
