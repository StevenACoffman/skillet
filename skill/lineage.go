package skill

import "strings"

// Frontmatter keys the lineage declaration lives under.
//
// **Nested under `metadata`, not top-level**, because speclint's allowlist is the
// agentskills.io published set and its own rule is that anything outside the spec belongs
// under `metadata`. A top-level `lineage:` would be a second deliberate deviation beside
// `tags`, and the value of "one deviation, for the key 163 corpora already use" is that
// there is one.
const (
	MetadataKey = "metadata"
	LineageKey  = "lineage"
)

// LineagePath is the dotted path a diagnostic names, so a reader is sent to the right
// line rather than to a top-level key that would be rejected.
const LineagePath = MetadataKey + "." + LineageKey

// Lineage values, closed. A skill declares one; nothing infers it.
const (
	// LineageUnset is the zero value: the skill declares nothing.
	//
	// **It is treated as strictly as a declared book lineage, and that is deliberate.**
	// The tempting reading is "undeclared means hand-written, so skip the format
	// checks", and it is the reading to refuse: it hands every skill an escape from the
	// RIA-TV++ contract by omission, which is the same hole as a book skill shedding
	// headings until it no longer looks like one. A contract you leave by saying
	// nothing is not a contract.
	//
	// The cost is honest and bounded: a corpus predating this field keeps its
	// diagnostics until each hand-written skill adds one line. That is a migration
	// rather than a false positive, which is the difference this field buys.
	LineageUnset Lineage = ""
	// BookDerived is distilled from a source text, so the RIA-TV++ segments are the
	// contract it was produced against.
	BookDerived Lineage = "book-derived"
	// HandWritten is authored directly. It never claimed the RIA-TV++ format, so
	// grading it against those segments reports a defect in a format the document does
	// not use.
	HandWritten Lineage = "hand-written"
)

// Lineage is how a skill's content came to exist: distilled from a source text, or
// authored directly.
//
// **It is declared, never derived**, and the distinction is the whole point. Content
// cannot tell you a skill's lineage: a hand-written skill has no RIA segments because it
// never claimed the format, and a book-derived one that has lost its segments is
// malformed — the two look identical from the outside and want opposite treatment. So a
// predicate over the current shape would answer the wrong question, however carefully it
// were tuned.
//
// This is the opposite epistemics from the executes-anything predicate skillsaw and adh
// use to scope their boundary checks, and the resemblance is a trap. That one *must* be
// derived, because nobody declares whether an artifact runs commands. This one *must not*
// be, because the document is the only witness to its own origin. A shared mechanism
// would force one of the two to be wrong.
//
// Closed and typed, unlike finding.Category, which stays a bare string because its drift
// is cosmetic and nothing branches on it. A lineage typo changes **which checks run**, so
// where the kernel owns the rules a value selects, it owns the value's vocabulary.
type Lineage string

// ParseLineage reads a declared lineage, reporting whether the value is recognised.
//
// **An unrecognised value is not silently accepted and not silently rejected.** ok is
// false and the returned Lineage is LineageUnset, so a caller gets the strictest
// treatment *and* can report the value it did not understand. That pairing is the point:
// `hand-written` must not quietly buy lenient treatment, and it must not fail closed
// without saying why either.
//
// There are no aliases. A typo and a newer vocabulary want the same handling, and an
// alias table manufactures synonyms that never shrink; vocabulary evolution belongs to a
// format version.
func ParseLineage(s string) (Lineage, bool) {
	switch trimmed := Lineage(strings.TrimSpace(s)); trimmed {
	case BookDerived, HandWritten:
		return trimmed, true
	case LineageUnset:
		// Absent rather than wrong: nothing was declared, so there is nothing to
		// report, and the caller's default applies.
		return LineageUnset, true
	default:
		return LineageUnset, false
	}
}

// ClaimsRIASegments reports whether a skill of this lineage was produced against the
// RIA-TV++ segment contract, and so should be graded on it.
//
// Only a declared hand-written lineage is exempt. Unset and unrecognised are both graded,
// which is what stops the exemption from being reachable by omission or by typo.
func (l Lineage) ClaimsRIASegments() bool { return l != HandWritten }

// UnrecognisedLineage reports whether a skill declared a lineage that is not in the
// vocabulary, and returns it verbatim so a diagnostic can quote what was written.
//
// A caller that never asks still behaves safely -- Lineage is LineageUnset in that case,
// which is graded strictly. What asking adds is the report, and the report is what stops
// a typo from being indistinguishable from silence.
func (s *Skill) UnrecognisedLineage() (string, bool) {
	if s.LineageRaw != "" && s.Lineage == LineageUnset {
		return s.LineageRaw, true
	}
	return "", false
}
