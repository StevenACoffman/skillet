// Package quotecheck is the fabrication guard: given quotations and the texts they claim
// to come from, it reports which of them cannot be found.
//
// It answers only the mechanical question -- does this run of words appear in the source at
// all. Judging whether a paraphrase is faithful, whether a quotation is used in context, or
// whether a located passage actually supports the claim built on it, stays with the reader;
// a tool that guessed at those would produce verdicts nobody could check. What it does catch
// is the failure that matters most and is hardest to spot by eye: a quotation that was never
// in the book.
//
// # What this package deliberately does not know
//
// It takes quotations already extracted, and knows nothing about where a quotation begins.
// That is not an omission. exegesis finds them as blockquote runs inside a skill's R
// segment; gnosis reads them from `gnosis_evidence` frontmatter on an OKF document. Both
// conventions are right for their own corpus and neither belongs in a shared kernel -- a
// general mechanism carrying one consumer's document conventions is exactly what the second
// consumer has to route around.
//
// So the caller keeps what the caller knows, and this package keeps the comparison. That
// division is the whole reason it can be shared.
//
// Everything here is pure.
package quotecheck

import (
	"regexp"
	"strings"

	"github.com/StevenACoffman/skillet/textnorm"
)

// MinPassageWords is the shortest passage worth a verdict.
//
// A three-word fragment is weak evidence in both directions: it appears in almost any book
// by chance, so finding it proves nothing, and failing to find it is as likely to be a split
// artifact as a fabrication. Reporting those would bury the findings that mean something.
const MinPassageWords = 6

// reSentence matches the sentence terminators a quotation is split on.
var reSentence = regexp.MustCompile(`[.!?]+`)

// Source is one plain-text source a quotation may have come from.
type Source struct {
	Name string // how to name it in a report; usually the file path
	Text string
}

// Finding is one checked passage and where it was found.
type Finding struct {
	Passage string
	Status  Status
	FoundIn string // name of the first source containing it; set only when Found
}

// Missing reports whether the passage was searched for and located in no source.
//
// It is deliberately false for an Unchecked finding. A caller gating on this asks "did the
// guard fire", and a passage nobody searched for has not fired it.
func (f Finding) Missing() bool { return f.Status == Missing }

// Check reports, for each passage of each quotation, whether any source contains it.
//
// Matching is per passage rather than per quotation, because in practice a quotation is
// long: 95% of exegesis's corpus writes a whole segment as one blockquote, median 860
// characters. Requiring all of that to appear verbatim and contiguously means a single
// editorial difference anywhere condemns the entire quotation and says nothing about where
// the problem is. Per passage, a fabricated sentence inserted into an otherwise faithful
// quotation is caught and named -- the subtle case that matters most.
//
// Both sides are folded before matching -- whitespace runs collapsed, typographic characters
// folded to ASCII (textnorm.Fold, shared so that no two guards in this family disagree about
// what counts as the same words) -- because a quotation is line-wrapped in Markdown and its
// source is not, so a literal comparison would report everything missing. That folding is
// the whole of the mechanical latitude given.
//
// Requires: nothing; nil quotes and nil sources are both valid.
// Ensures: one Finding per quotation-passage, in the order given, and the result is never
// nil, so a caller need not distinguish "no quotations" from "no result". A quotation whose
// passages
// are all shorter than MinPassageWords still yields exactly one Finding, Unchecked, carrying
// the quotation itself -- see 'the vanishing quotation' below. With no sources, every
// Finding is Unchecked rather than Missing. It is pure.
//
// # The vanishing quotation
//
// Emitting nothing for a quotation that was too short to split would be the dangerous
// silence: a caller counting findings would see a clean pass over a quotation nobody
// checked. So the short case is reported rather than skipped, which is the difference
// between "we looked and found nothing wrong" and "we did not look".
func Check(quotes []string, sources []Source) []Finding {
	// Returning early rather than falling through the loop: with no quotations there is
	// nothing to match, and folding the sources first would be folding whole books for a
	// result already known to be empty.
	if len(quotes) == 0 {
		return []Finding{}
	}

	// Normalize each source once. A source is a whole book; folding it per passage would
	// multiply that cost by the number of passages for no benefit.
	haystacks := make([]string, len(sources))
	for i, s := range sources {
		haystacks[i] = textnorm.Fold(s.Text)
	}

	findings := make([]Finding, 0, len(quotes))
	for _, q := range quotes {
		passages := Passages(q)
		if len(passages) == 0 {
			findings = append(findings, Finding{Passage: q, Status: Unchecked})
			continue
		}
		for _, p := range passages {
			findings = append(findings, locate(p, sources, haystacks))
		}
	}
	return findings
}

// locate searches the folded sources for one passage.
func locate(passage string, sources []Source, haystacks []string) Finding {
	if len(haystacks) == 0 {
		return Finding{Passage: passage, Status: Unchecked}
	}
	needle := textnorm.Fold(passage)
	for i, h := range haystacks {
		if strings.Contains(h, needle) {
			return Finding{Passage: passage, Status: Found, FoundIn: sources[i].Name}
		}
	}
	return Finding{Passage: passage, Status: Missing}
}

// Support returns how many findings were located in a source: the countable half of
// book2skill's V1, which asks whether the source really contains supporting evidence for the
// unit a claim was distilled from.
//
// Support is measured in passages rather than quotations because a quotation is not a unit
// of evidence here: where a whole segment is one blockquote, a quotation count is 1 for
// nearly every document and a threshold of two could never be met by a faithful one.
//
// It counts located passages, not independent ones. Two passages taken from the same
// paragraph are two here and one to V1, and whether a passage supports the unit at all is a
// judgment no containment check can make. A threshold on this number gates the half that can
// be counted; it does not decide V1.
//
// Ensures: 0 <= result <= len(findings); Support(nil) == 0; it is pure.
func Support(findings []Finding) int {
	located := 0
	for _, f := range findings {
		if f.Status == Found {
			located++
		}
	}
	return located
}

// Passages splits a quotation into the chunks matched individually: sentences, dropped to
// those of at least MinPassageWords.
//
// Terminators are discarded along with the split, so abbreviations and decimals over-split
// ("e.g." becomes two fragments). Both are deliberate and safe: over-splitting leaves the
// substantive text still being checked, and the resulting fragments are too short to survive
// the MinPassageWords filter.
//
// Ensures: every returned passage has at least MinPassageWords words; it is pure.
func Passages(quote string) []string {
	var out []string
	for _, chunk := range reSentence.Split(quote, -1) {
		if chunk = strings.TrimSpace(chunk); len(strings.Fields(chunk)) >= MinPassageWords {
			out = append(out, chunk)
		}
	}
	return out
}
