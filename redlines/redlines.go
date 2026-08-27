// Package redlines enforces book2skill's mechanical Quality Red Lines over an
// already-loaded skill: the six RIA-TV++ body segments, a per-quotation word
// ceiling, and a description that states when to invoke the skill.
//
// These are distinct from speclint, which encodes the agentskills.io specification.
// The spec changes when agentskills.io changes; the red lines change when the
// book2skill methodology changes, so they are separate packages rather than one.
//
// Every check is pure — a loaded skill in, diagnostics out — so the caller owns the
// file I/O and decides the exit code. It lives in skillet so exegesis (which gates a
// tree on the way in) and skillsaw (which must not let an optimization regress
// structure) enforce one definition rather than two.
package redlines

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/skill"
)

// MaxQuoteWords is the per-quotation word ceiling. It is exported so a second
// consumer enforces this ceiling by reference instead of repeating the literal —
// the drift that splitting these checks out of exegesis is meant to prevent.
const MaxQuoteWords = 150

var reFence = regexp.MustCompile("(?s)```.*?```|~~~.*?~~~")

// Check returns every Quality Red Line diagnostic for s. An empty result means s
// passes them.
//
// Requires: s is a loaded skill (Body and Description populated from its SKILL.md).
// Ensures:  the result covers the missing RIA-TV++ segments, every over-long
//
//	quotation, and a description with no trigger condition; fenced code is
//	ignored; it is pure.
//
// The fourth red line — that test-prompts.json exists — needs the filesystem, so it
// stays with the caller.
func Check(s *skill.Skill) []finding.Diagnostic {
	body := reFence.ReplaceAllString(s.Body, "")
	var ds []finding.Diagnostic
	// The RIA-TV++ segments are asked of a skill that claims that format, which the
	// skill declares (skill.Lineage) rather than the checker inferring. Measured over a
	// 233-skill corpus, an unguarded contract reported **six diagnostics each for 48
	// hand-written skills** -- `gh-cli`, `vale`, `unconventional-commits` and the rest --
	// about a format those documents never used. That is not drift from the contract; it
	// is a document that was never produced against it.
	//
	// Inferring from the body was considered and refused: a hand-written skill with no
	// segments and a malformed book skill that lost its segments look identical, and want
	// opposite treatment. See skill.Lineage for why this must be declared where
	// skillsaw's executes-anything predicate must be derived.
	if s.Lineage.ClaimsRIASegments() {
		ds = append(ds, checkSegments(body)...)
	}
	// A value outside the vocabulary is reported, and was already graded strictly above
	// because ParseLineage leaves Lineage unset for it. Both halves matter: a typo must
	// not buy lenience, and it must not fail closed in silence either.
	if raw, bad := s.UnrecognisedLineage(); bad {
		ds = append(ds, diagf(
			"redline: unknown %s %q (want %q or %q); graded as if it claimed the "+
				"RIA-TV++ format", skill.LineagePath, raw, skill.BookDerived,
			skill.HandWritten))
	}
	ds = append(ds, checkQuotes(body)...)
	// Only ask for a trigger when there was a description to read; see the note on
	// skill.Skill.FrontmatterErr. An unparsed block leaves Description empty, and
	// demanding a trigger of it reports a consequence of the YAML error as a defect
	// in prose the author did write.
	//
	// The two checks above are not guarded: they read the body, which
	// splitFrontmatter produces before the parse is attempted, so their findings are
	// real either way.
	if s.FrontmatterErr == nil {
		ds = append(ds, checkTrigger(s.Description)...)
	}
	return ds
}

// Quotes returns each contiguous blockquote in body, in document order: one string per
// run, with the "> " markers stripped and the run's lines joined by a single space.
//
// Fenced code blocks are removed first, so a "> " line inside a shell transcript or a
// diff is not mistaken for a quotation. A run of bare ">" lines carries no text and is
// not returned.
//
// This is exported because it is the same extraction the MaxQuoteWords red line counts.
// A caller checking quotations against their sources needs to agree with the rule being
// enforced; a second implementation elsewhere would disagree at the margins — over
// fences, over lazy continuation — and the two tools would then dispute what a quote is.
//
// Ensures: every returned string contains at least one non-space character; it is pure.
func Quotes(body string) []string {
	var out, run []string
	flush := func() {
		if joined := strings.Join(run, " "); strings.TrimSpace(joined) != "" {
			out = append(out, joined)
		}
		run = run[:0]
	}
	for _, line := range strings.Split(reFence.ReplaceAllString(body, ""), "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, ">") {
			run = append(run, strings.TrimSpace(strings.TrimPrefix(t, ">")))
			continue
		}
		flush()
	}
	flush()
	return out
}

// checkSegments flags any missing RIA segment. A segment is present when a "## "
// heading's first token (its leading letters/digits, upper-cased) is the label.
func checkSegments(body string) []finding.Diagnostic {
	present := headingLabels(body)
	var ds []finding.Diagnostic
	// The six RIA-TV++ segments, in the order a skill declares them.
	for _, label := range []string{"R", "I", "A1", "A2", "E", "B"} {
		if !present[label] {
			ds = append(ds, diagf(
				"redline: body is missing the %q RIA segment (needs R, I, A1, A2, E, B)", label))
		}
	}
	return ds
}

// headingLabels returns the set of "## " heading first-token labels in body.
func headingLabels(body string) map[string]bool {
	labels := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "## ")
		if !ok {
			continue
		}
		if label := leadingAlnum(strings.TrimSpace(rest)); label != "" {
			labels[strings.ToUpper(label)] = true
		}
	}
	return labels
}

// leadingAlnum returns the leading run of letters and digits in s.
func leadingAlnum(s string) string {
	for i, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return s[:i]
		}
	}
	return s
}

// checkQuotes flags each contiguous blockquote whose word count exceeds the limit.
func checkQuotes(body string) []finding.Diagnostic {
	var ds []finding.Diagnostic
	for _, q := range Quotes(body) {
		if n := len(strings.Fields(q)); n > MaxQuoteWords {
			ds = append(
				ds,
				diagf("redline: a quotation is %d words, over the %d-word limit", n, MaxQuoteWords),
			)
		}
	}
	return ds
}

// checkTrigger flags a description that states no trigger condition — it contains
// none of the cue phrases that signal when to invoke the skill. Heuristic: it
// catches the "a skill about X" anti-pattern without over-flagging.
//
// The trigger cues are the declarative forms — "trigger:", not "trigger". The bare word is
// a domain term, and "a skill about database triggers" is precisely the anti-pattern this
// exists to catch: matching it would trade a false positive for a false negative in a
// blocking check, which is the worse direction, because a redline that lets bad skills
// through is one nobody notices is broken.
//
// Measured over 286 real skills when this was fixed: 122 descriptions contain "trigger" at
// all and only 72 use it declaratively. The other 50 happen to clear on "when" or "invoke",
// so bare and declarative agree on that corpus — a property of the corpus, not of the rule.
func checkTrigger(description string) []finding.Diagnostic {
	low := strings.ToLower(description)
	for _, cue := range []string{
		"when", "whenever", "invoke", "reach for", "before ", "after ",
		"trigger:", "triggers on", "trigger signal",
	} {
		if strings.Contains(low, cue) {
			return nil
		}
	}
	return []finding.Diagnostic{
		diag(
			"redline: description should state a trigger condition (when to use the skill), not just what it is",
		),
	}
}

// diag builds an error-severity red-line diagnostic. Category and Path are left
// empty so the diagnostic marshals as {severity, message} — the shape exegesis
// already emits, keeping its machine output stable across this migration.
func diag(message string) finding.Diagnostic {
	return finding.Diagnostic{Severity: finding.SeverityError, Message: message}
}

// diagf builds an error-severity red-line diagnostic from a format string.
func diagf(format string, a ...any) finding.Diagnostic {
	return diag(fmt.Sprintf(format, a...))
}
