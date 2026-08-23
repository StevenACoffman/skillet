package ruleset

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// marker is one body-line prefix and the field it fills.
type marker struct {
	prefix string
	set    func(*Rule, string)
}

// markers is the canonical-form's body vocabulary, in one place.
//
// One place because there are two readers: applyBody, and the test asserting that
// FormatVersion was bumped when the set changed. A marker added to a switch and not
// to a list is the second-place-to-remember problem; a marker added here is counted.
func markers() []marker {
	return []marker{
		{"✗", func(r *Rule, v string) { r.Bad = v }},
		{"✓", func(r *Rule, v string) { r.Good = v }},
		{"↦", func(r *Rule, v string) { r.SourceAnchor = v }},
	}
}

// applyBody attaches one body line to a rule, or reports a line it cannot place.
//
// Requires: trimmed is non-empty and has no surrounding space.
// Ensures: a line opening with a known marker sets that field; any other line is
// rationale, first line or continuation. A line opening with an **unrecognised
// symbol** is an error rather than rationale, which is the whole point of the
// function returning one.
//
// The rejection is deliberately narrow: only a leading rune in Unicode's *symbol*
// categories, and only one that is not a known marker. It is not "any unknown
// punctuation", because a rationale legitimately begins with an em dash, a curly
// quotation mark, or a parenthesis, and rejecting those would reject prose somebody
// will write. Every marker this form uses is a typographic symbol, chosen precisely
// because prose does not begin that way — so "a rationale may not begin with a
// symbol" is the constraint that makes the marker set extensible, and it is stated
// here rather than discovered.
//
// What this cannot catch is an ASCII marker added later. The answer is that the form
// should not add one; TestEveryMarkerIsNonASCII makes that a checked claim rather
// than an intention.
func applyBody(r *Rule, trimmed string) error {
	for _, m := range markers() {
		if strings.HasPrefix(trimmed, m.prefix) {
			m.set(r, strings.TrimSpace(strings.TrimPrefix(trimmed, m.prefix)))
			return nil
		}
	}
	if first, _ := utf8.DecodeRuneInString(trimmed); unicode.IsSymbol(first) {
		return fmt.Errorf(
			"ruleset: unrecognised marker %q in %q; a rationale may not begin with a symbol",
			string(first), trimmed)
	}
	if r.Rationale == "" {
		r.Rationale = trimmed
	} else {
		r.Rationale += " " + trimmed
	}
	return nil
}
