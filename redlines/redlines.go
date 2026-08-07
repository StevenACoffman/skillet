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
	ds = append(ds, checkSegments(body)...)
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
	var quote []string
	flush := func() {
		if n := len(strings.Fields(strings.Join(quote, " "))); n > MaxQuoteWords {
			ds = append(
				ds,
				diagf("redline: a quotation is %d words, over the %d-word limit", n, MaxQuoteWords),
			)
		}
		quote = quote[:0]
	}
	for _, line := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, ">") {
			quote = append(quote, strings.TrimSpace(strings.TrimPrefix(t, ">")))
			continue
		}
		flush()
	}
	flush()
	return ds
}

// checkTrigger flags a description that states no trigger condition — it contains
// none of the cue phrases that signal when to invoke the skill. Heuristic: it
// catches the "a skill about X" anti-pattern without over-flagging.
func checkTrigger(description string) []finding.Diagnostic {
	low := strings.ToLower(description)
	for _, cue := range []string{"when", "whenever", "invoke", "reach for", "before ", "after "} {
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
