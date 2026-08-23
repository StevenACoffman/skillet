// Package skilllens locates the evidence behind the three SkillLens quality dimensions:
// whether a skill encodes concrete failure mechanisms, whether its instructions are
// specific rather than softened, and whether it draws a boundary around what not to do.
//
// The dimensions come from microsoft/SkillLens (data/meta_skills/quality_rubric_3dim.md,
// arXiv:2605.23899), each validated at 65-66% predictive accuracy against downstream
// skill utility. They are shared because two consumers already scored them independently
// -- skillsaw as rubric dims 3/5/9, adh as failure-handling / actionable-specificity /
// boundary-section -- and two implementations of one rubric drift, which is what promoting
// speclint and redlines was for.
//
// These return located evidence, not finding.Diagnostic. redlines and speclint each *are*
// a gate with a fixed severity; these are not. skillsaw turns them into 1-10 rubric
// penalties, adh into a 0..1 factor, and a linter would want error-versus-warning per
// check. Returning a Diagnostic would fix a severity here that all three then have to
// unpick, so the caller gets what was found and decides what it means.
//
// The weights, the 1-10 scale and any substance threshold stay with the caller: skillsaw
// weights these three at 35 of 100 and adh at 60, legitimately, because they grade
// different artifacts. Same boundary that keeps speclint's description cap here and the
// cost of exceeding it in skillsaw.
//
// Validity scope: these were validated on one class of skill, and a consumer that runs them
// over another can read an empty result as a pass when it is not one.
//
// The class is the discipline skill -- the failure being "skips a rule under pressure",
// where a boundary section and an absence of hedging are the right signals. superpowers'
// *Match the Form to the Failure* classifies three others, each wanting a different form of
// guidance: output has the wrong shape (wants a positive recipe stating what the output IS,
// in order), a required element is omitted (wants a REQUIRED slot in the template), and
// behaviour depends on a condition (wants a conditional on an observable predicate).
//
// BlacklistSections is the exposed one, and the exposure is not symmetric. For a
// wrong-shape skill a prohibition list is not merely absent evidence, it is the form their
// head-to-head reports as *worse than no guidance at all* -- so a consumer scoring its
// absence as a defect will recommend the change that harms the skill. FailureMechanisms and
// SofteningPhrases are safer: the softening vocabulary is hedges (as appropriate, at your
// discretion, if you prefer), none of which matches a conditional on an observable, so a
// skill written as "if the response has a next cursor, page again" is not flagged.
//
// Nothing here classifies a skill's failure type, and nothing should: which class a skill
// belongs to is a judgement, and a detector that guessed would put an uncalibrated
// heuristic under a scoring dimension. This paragraph is the whole mitigation -- the
// evidence says what it covers, and the caller decides what to do where it does not.
//
// Everything here is pure; the caller parses the document.
package skilllens

import (
	"regexp"
	"strings"

	"github.com/StevenACoffman/skillet/markdown"
)

// The two places evidence appears.
const (
	KindProse   Kind = "prose"   // an inline phrase in the body text
	KindSection Kind = "section" // a heading whose title matches
)

// The finding.Diagnostic categories for this package's three detectors.
//
// They live here because the detector lives here: two consumers reading one detector and
// naming its output differently is drift, and it had already happened -- exegesis emitted
// "skilllens-softening" and canonizer "softening" for the same SofteningPhrases call.
// The rule this generalizes: where the kernel owns the detector, the kernel owns the name.
//
// Unprefixed on purpose. Across the family's thirty category values there is not one
// same-word-different-meaning collision, and the only observed defect is the opposite --
// one concept spelled two ways. A package prefix defends against the hazard that has never
// occurred and manufactures the one that has, since a second mechanism detecting the same
// class would have to spell it differently. It would also put provenance in a field that
// classifies, which is the collapse Severity and Action are kept apart to avoid.
//
// Each names the defect rather than the dimension, and the no-X form follows canonizer's
// existing convention -- no-anchor for never declared, anchor-absent for declared and not
// found. Two of the three detectors fire on absence, so naming them for the dimension read
// as the opposite of what they mean: a "failure" category that means no failure handling
// was written.
//
// What a shared name does NOT make shared, because the two are easy to conflate and the
// package's whole charter is the difference: a category names the finding class, never the
// threshold that decides whether to report one. Consumers of one detector legitimately
// disagree about when its evidence is a defect, and today they do -- exegesis reports
// softening at len >= 3, canonizer at len > 0, from the same SofteningPhrases call.
// FailureMechanisms feeds a bare presence test in exegesis, a kind-filtered count in
// skillsaw gated on Doc.HasCodeBlock, and a 0..1 factor in adh. All four are correct for
// what they grade. Sharing the name is what stops them describing one finding two ways;
// it is not an invitation to reconcile the policies, and a change that made them agree
// would be undoing the deliberate split stated above -- evidence out, policy to the caller.
const (
	// CategoryNoFailureMode is reported when FailureMechanisms finds nothing: the skill
	// says nothing about what to do when the thing it describes goes wrong.
	CategoryNoFailureMode = "no-failure-mode"
	// CategorySoftening is reported when SofteningPhrases finds hedges. It is the one
	// detector of the three that fires on presence.
	CategorySoftening = "softening"
	// CategoryNoBoundary is reported when BlacklistSections finds nothing: the skill
	// draws no boundary around what not to do.
	CategoryNoBoundary = "no-boundary"
)

// failureCN and failureEN detect inline "if X fails / when Y errors" branches.
//
// Both languages are carried deliberately: the vocabulary began in a Chinese-language
// source, and a China-only list scores every English skill as defect-free, which is the
// opposite of useful.
var (
	failureCN = regexp.MustCompile(`如果.{0,24}(失败|错误|不可用|超时|冲突|缺失|找不到|异常|没有)`)
	failureEN = regexp.MustCompile(
		`(?i)(if|when)\s.{0,40}(fail|error|missing|unavailable|timeout|not found|goes wrong|blocked|get stuck|gets stuck)`,
	)
)

// Kind says where a span was found, because one detector reports both.
//
// A failure mechanism can be encoded inline ("if the API times out, re-query") or by a
// dedicated section, and a caller that could not tell them apart could not weigh them
// differently -- which every current caller does.
type Kind string

// Span is one piece of located evidence.
type Span struct {
	Kind Kind
	// Text is the matched fragment for a prose span, or the heading title for a section.
	//
	// Do not display it raw when it came from a regex detector. FailureMechanisms matches
	// against Doc.Prose, where code blocks and spans are blanked to spaces, so a branch
	// written "If `go test ./...` fails" arrives here as "If" + thirteen spaces + "fail".
	// The window is also an arbitrary cut -- the pattern allows 40 characters between the
	// conditional and the failure word, so matches routinely end mid-word ("the right
	// fail" from "failures"). Both make it good for counting and debugging, and poor for
	// showing a reader.
	//
	// SofteningPhrases and the section detectors are unaffected: the first reports the
	// vocabulary term it searched for, the second the heading title, and neither is ever
	// blank. Today no consumer in the family reads this field except canonizer, which
	// reads a SofteningPhrases term -- so the hazard is real and currently unmet.
	//
	// If a consumer ever needs displayable evidence the answer is byte offsets on this
	// struct, not a copy of the source carried beside the match: Doc holds no body, so
	// carrying the substring would mean the kernel storing a second copy of every source
	// or taking one as a parameter, and offsets let the caller widen the window to
	// something readable, which a fixed match cannot. Prose is byte-offset-identical to
	// the source, so an offset here indexes the caller's own body -- see
	// markdown.TestProsePreservesOffsets, which pins the property this would depend on.
	Text string
	// Units is the section's count of concrete content (list items, table rows,
	// substantial paragraphs); zero for a prose span.
	//
	// It travels with the span because "enough substance to count" is policy, not
	// detection: skillsaw requires a section to out-weigh the body it sits in, and
	// another caller may want any section at all. Returning the number lets each decide.
	Units int
}

// FailureSectionTitles returns the heading vocabulary that counts as failure-mode
// encoding. A fresh slice per call: an exported slice would be shared mutable state that
// one caller could edit under every other.
func FailureSectionTitles() []string {
	return []string{
		"反例", "边界", "异常", "失败", "回退",
		"boundary", "pitfall", "anti-pattern", "antipattern", "failure mode",
		"common failure", "when not to", "troubleshoot", "if this fails", "limitation",
		"edge case", "recovery", "when it breaks", "gotcha", "fallback", "red line",
	}
}

// SofteningTerms returns the phrases that hedge an instruction into unfollowability.
func SofteningTerms() []string {
	return []string{
		"建议", "可以考虑", "根据情况", "灵活把握", "视情况而定",
		"as appropriate", "it depends", "you might want", "feel free",
		"at your discretion", "where appropriate", "as you see fit", "if you prefer",
	}
}

// BlacklistTitles returns the heading vocabulary for a counter-example or boundary
// section.
func BlacklistTitles() []string {
	return []string{
		"反例", "黑名单", "反模式", "不要做", "不要",
		"don't", "do not", "avoid", "blacklist", "boundary", "pitfall",
		"anti-pattern", "antipattern", "when not to", "common mistake",
		"common failure", "red flag", "red line", "failure mode", "gotcha",
		"caution", "limitation",
	}
}

// FailureMechanisms returns the evidence that a skill encodes what to do when things go
// wrong: inline conditional branches, and sections named for failure.
//
// Both kinds come back from one call because they answer one question, and a caller that
// had to make two would be free to ask only half of it. Kind separates them.
//
// Ensures: it is pure; prose spans carry Units 0.
func FailureMechanisms(d *markdown.Doc) []Span {
	spans := proseSpans(d, failureCN)
	spans = append(spans, proseSpans(d, failureEN)...)
	return append(spans, sectionSpans(d, FailureSectionTitles())...)
}

// SofteningPhrases returns the hedging phrases found in the body text.
//
// Ensures: every span has Kind KindProse; it is pure.
func SofteningPhrases(d *markdown.Doc) []Span {
	var spans []Span
	lower := strings.ToLower(d.Prose)
	for _, term := range SofteningTerms() {
		t := strings.ToLower(term)
		for i := 0; ; {
			j := strings.Index(lower[i:], t)
			if j < 0 {
				break
			}
			spans = append(spans, Span{Kind: KindProse, Text: term})
			i += j + len(t)
		}
	}
	return spans
}

// BlacklistSections returns the sections that draw a boundary around what not to do.
//
// Only headings are considered: the claim being evidenced is that the skill *has* such a
// section, and a passing mention of "anti-pattern" in a paragraph is not that.
//
// Ensures: every span has Kind KindSection and carries the section's Units; it is pure.
func BlacklistSections(d *markdown.Doc) []Span {
	return sectionSpans(d, BlacklistTitles())
}

// proseSpans returns one span per match of re in the document's prose.
//
// It matches Doc.Prose, where markdown has already blanked code blocks and spans, so a
// conditional inside a shell transcript is not read as the skill's own instruction.
func proseSpans(d *markdown.Doc, re *regexp.Regexp) []Span {
	found := re.FindAllString(d.Prose, -1)
	spans := make([]Span, 0, len(found))
	for _, m := range found {
		spans = append(spans, Span{Kind: KindProse, Text: m})
	}
	return spans
}

// sectionSpans returns one span per section whose title matches any of the terms.
//
// Matching is by matchesSignal, not plain substring: an ASCII term must begin at a word
// boundary and its regular English inflections count, so "boundary" matches the standard
// book2skill "## B — Boundaries" heading (a plain Contains misses it, since "boundary"
// is not a substring of "boundaries") while "red flag" does not match "requi[red flag]s".
func sectionSpans(d *markdown.Doc, terms []string) []Span {
	var spans []Span
	for _, sec := range d.Sections {
		title := strings.ToLower(sec.Title)
		for _, term := range terms {
			if matchesSignal(title, strings.ToLower(term)) {
				spans = append(spans, Span{Kind: KindSection, Text: sec.Title, Units: sec.Units})
				break
			}
		}
	}
	return spans
}

// matchesSignal reports whether s contains sig or a regular inflection of it (both
// already lowercased). An ASCII signal must begin at a word boundary, so "red flag" does
// not match "requi[red flag]s" (the "red" is mid-word). Append-only inflections already
// match because the signal is a prefix of the longer word ("mistake" in "mistakes",
// "troubleshoot" in "troubleshooting"); the one regular inflection a prefix match cannot
// reach — a consonant+"y" pluralizing to "ies" — is probed explicitly, so "boundary"
// matches "boundaries". Non-ASCII (CJK) signals have no word boundaries and match as
// plain substrings.
func matchesSignal(s, sig string) bool {
	if !isASCII(sig) {
		return strings.Contains(s, sig)
	}
	if matchesForm(s, sig) {
		return true
	}
	if plural, ok := iesPlural(sig); ok {
		return matchesForm(s, plural)
	}
	return false
}

// matchesForm reports whether form appears in s beginning at a word boundary.
func matchesForm(s, form string) bool {
	for idx := 0; ; {
		p := strings.Index(s[idx:], form)
		if p < 0 {
			return false
		}
		p += idx
		if p == 0 || !isWordByte(s[p-1]) {
			return true // form starts at a word boundary
		}
		idx = p + 1
	}
}

// iesPlural returns sig with a trailing consonant+"y" rewritten to "ies" — the regular
// English plural that rewrites the stem (boundary->boundaries, policy->policies), which
// a prefix match cannot reach. ok is false otherwise; a vowel+"y" ("day"->"days") is
// append-only and already covered by matchesForm.
func iesPlural(sig string) (plural string, ok bool) {
	if len(sig) < 2 || sig[len(sig)-1] != 'y' || isVowel(sig[len(sig)-2]) {
		return "", false
	}
	return sig[:len(sig)-1] + "ies", true
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
