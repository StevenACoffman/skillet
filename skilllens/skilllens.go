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
