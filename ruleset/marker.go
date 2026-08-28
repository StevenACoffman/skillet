package ruleset

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// warrantDate is the only spelling of At the form accepts. One spelling because two tools
// comparing warrants by string would otherwise disagree about dates that are the same day.
const warrantDate = "2006-01-02"

// Warrant records who decided a rule, when, and why, for a rule no source can anchor.
//
// **An adjudicated artifact is sourced differently, not unsourced.** When two rules conflict
// and a person picks one, the decision is knowledge present in neither source, so it can
// carry no source anchor and fails an anchor-requiring provenance check by construction --
// while being the highest-value thing a review produces. gnosis states it most sharply: *"a
// decision that weighed two published positions names both, even though the decision appears
// in neither."* Three repositories derived that independently before it was written down
// once; a check that protects evidence must not reject the one artifact that cannot carry
// any.
//
// The shape is deliberately smaller than the governance models that want it. No tiers, no
// co-signers, no reversal links: those belong to a consumer's authority model, and putting
// them in a shared kernel exports one consumer's governance to every other. The kernel
// carries the datum; the consumer keeps the decision about what to do with it -- the same
// split the manifest's test-prompts hash and skilllens' category names already use.
type Warrant struct {
	// By is who decided, as one whitespace-free token: an address or a handle. The
	// canonical form is positional, so a name with a space in it cannot be represented.
	By string

	// At is when, as a 2006-01-02 date.
	//
	// A string rather than a time.Time, which is the weaker choice by every rule except the
	// one that governs here: parsing and re-formatting would silently rewrite 2026-8-27 as
	// 2026-08-27, and byte-identical round-tripping is what the inert-render property and
	// canonizer's drift check both rest on. Validity is enforced where the text is read --
	// see Valid, and parseWarrant, which refuses a date it cannot parse.
	At string

	// Rationale is why, and it is required. A warrant is the only record of a decision that
	// carries no other evidence, so a half-recorded one is worse than none: it looks like
	// provenance while establishing nothing.
	Rationale string
}

// marker is one body-line prefix and the field it fills.
//
// set returns an error because one marker's value can be malformed: a warrant carries three
// things and a partial one is worse than none. The three single-value markers cannot fail
// and return nil, which is cheaper than a second table for the one that can -- a
// special-purpose branch inside a general mechanism is the shape this form avoids.
type marker struct {
	prefix string
	set    func(*Rule, string) error
}

// markers is the canonical-form's body vocabulary, in one place.
//
// One place because there are two readers: applyBody, and the test asserting that
// FormatVersion was bumped when the set changed. A marker added to a switch and not
// to a list is the second-place-to-remember problem; a marker added here is counted.
func markers() []marker {
	return []marker{
		{"✗", func(r *Rule, v string) error { r.Bad = v; return nil }},
		{"✓", func(r *Rule, v string) error { r.Good = v; return nil }},
		{"↦", func(r *Rule, v string) error { r.SourceAnchor = v; return nil }},
		{"⚖", func(r *Rule, v string) error {
			w, err := parseWarrant(v)
			if err != nil {
				return err
			}
			r.Warrant = w
			return nil
		}},
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
			return m.set(r, strings.TrimSpace(strings.TrimPrefix(trimmed, m.prefix)))
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

// Present reports whether a warrant was recorded at all.
//
// Distinct from Valid on purpose. Absent means the rule was never adjudicated, which is the
// ordinary case and no kind of defect; invalid means somebody recorded a decision badly.
func (w Warrant) Present() bool { return w != Warrant{} }

// Valid reports whether a warrant records all three of who, when and why.
//
// Ensures: pure. False for the zero value, so a caller that checks Valid without checking
// Present treats an unadjudicated rule as a badly adjudicated one -- which is why both exist.
func (w Warrant) Valid() bool {
	if w.By == "" || w.Rationale == "" {
		return false
	}
	_, err := time.Parse(warrantDate, w.At)
	return err == nil
}

// parseWarrant reads the body of a ⚖ line: who, when, then the rest as why.
//
// Positional rather than key=value because every other marker in this form carries free
// text, and a keyed encoding inside a prose document would be the odd one out. The cost is
// that By cannot contain a space.
//
// Requires: v is the line with its marker and surrounding space already removed.
// Ensures:  pure. An error rather than a partial Warrant for every malformed case, so a
// half-recorded decision cannot reach a consumer looking like a whole one.
func parseWarrant(v string) (Warrant, error) {
	by, rest := cutField(strings.TrimSpace(v))
	at, rationale := cutField(rest)
	w := Warrant{By: by, At: at, Rationale: strings.TrimSpace(rationale)}
	if !w.Valid() {
		return Warrant{}, fmt.Errorf(
			"ruleset: malformed warrant %q; want \"who %s why\"", v, warrantDate)
	}
	return w, nil
}

// cutField splits s at its first run of whitespace, returning the field and the remainder.
func cutField(s string) (field, rest string) {
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimLeftFunc(s[i:], unicode.IsSpace)
}
