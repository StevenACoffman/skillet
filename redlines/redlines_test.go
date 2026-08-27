package redlines_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/redlines"
	"github.com/StevenACoffman/skillet/skill"
)

// allSegments is a body carrying every RIA-TV++ segment heading, so a case can vary
// one thing at a time.
const allSegments = "## R\n\n## I\n\n## A1\n\n## A2\n\n## E\n\n## B\n"

// triggerDesc states a trigger condition, so it never contributes a diagnostic.
const triggerDesc = "Use when the reader needs the thing done."

func load(body, description string) *skill.Skill {
	return &skill.Skill{Body: body, Description: description}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		body         string
		description  string
		wantCount    int
		wantContains string
	}{
		"a complete skill passes": {
			body: allSegments, description: triggerDesc,
		},
		"every segment missing is six diagnostics": {
			body: "# Body\n\nNo segments here.\n", description: triggerDesc,
			wantCount: 6, wantContains: `missing the "R" RIA segment`,
		},
		"one segment missing is one diagnostic": {
			body: "## R\n\n## I\n\n## A1\n\n## A2\n\n## E\n", description: triggerDesc,
			wantCount: 1, wantContains: `missing the "B" RIA segment`,
		},
		"segment labels are matched case-insensitively on the first token": {
			body:        "## r — Recognition\n## i\n## a1\n## a2\n## e\n## b\n",
			description: triggerDesc,
		},
		"a description with no trigger is flagged": {
			body: allSegments, description: "A skill about testing strategy.",
			wantCount: 1, wantContains: "should state a trigger condition",
		},
		// The hazard case, and the reason the cues are the declarative forms rather than
		// the bare word. "trigger" is a domain term; this description is the very
		// anti-pattern the check exists to catch and it contains it. Shortening the cue to
		// "trigger" would silently turn a blocking check into one that passes this.
		"the anti-pattern is still flagged when it happens to say trigger": {
			body: allSegments, description: "A skill about database triggers.",
			wantCount: 1, wantContains: "should state a trigger condition",
		},
		// The three declarative forms found in a real 286-skill corpus. Each of these was
		// flagged before the cue list learned them, so each is a blocking false positive
		// on a description that states its trigger in the clearest way available.
		"a description opening Trigger: passes": {
			body:        allSegments,
			description: "Trigger: user is using context.WithValue or asking about context keys.",
		},
		"a description with Triggers on: passes": {
			body:        allSegments,
			description: `Write property-based tests using rapid. Triggers on: "PBT", "rapid tests".`,
		},
		"a description with Trigger signals: passes": {
			body:        allSegments,
			description: `Review a pull request end to end. Trigger signals: - "review PR <n>"`,
		},
		"an over-long quotation is flagged": {
			body:        allSegments + "\n> " + strings.Repeat("word ", redlines.MaxQuoteWords+1),
			description: triggerDesc,
			wantCount:   1, wantContains: "over the 150-word limit",
		},
		"a quotation at the limit passes": {
			body:        allSegments + "\n> " + strings.Repeat("word ", redlines.MaxQuoteWords),
			description: triggerDesc,
		},
		"fenced code is not scanned for quotations": {
			body: allSegments + "\n```\n> " +
				strings.Repeat("word ", redlines.MaxQuoteWords+1) + "\n```\n",
			description: triggerDesc,
		},
		"separate quotations are counted separately, not joined": {
			// Two blockquotes of half the limit each must not add up to a violation.
			body: allSegments + "\n> " + strings.Repeat("word ", 100) +
				"\n\nprose\n\n> " + strings.Repeat("word ", 100),
			description: triggerDesc,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := redlines.Check(load(tc.body, tc.description))
			if len(got) != tc.wantCount {
				t.Fatalf("Check returned %d diagnostics, want %d: %+v",
					len(got), tc.wantCount, got)
			}
			if tc.wantContains == "" {
				return
			}
			messages := make([]string, 0, len(got))
			for _, d := range got {
				messages = append(messages, d.Message)
			}
			joined := strings.Join(messages, "\n")
			if !strings.Contains(joined, tc.wantContains) {
				t.Errorf("expected a diagnostic containing %q, got:\n%s",
					tc.wantContains, joined)
			}
		})
	}
}

func TestCheckTriggerCues(t *testing.T) {
	t.Parallel()
	// Each cue phrase alone is enough to satisfy the trigger red line. Pinning them
	// keeps the heuristic's accepted vocabulary from drifting silently.
	for _, desc := range []string{
		"Use when the thing happens.",
		"Whenever the thing happens.",
		"Invoke this for the thing.",
		"Reach for this during the thing.",
		"Read this before writing the thing.",
		"Run this after the thing.",
	} {
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			if got := redlines.Check(load(allSegments, desc)); len(got) != 0 {
				t.Errorf("%q should satisfy the trigger red line, got %+v", desc, got)
			}
		})
	}
}

func TestCheckIsPure(t *testing.T) {
	t.Parallel()
	// The caller may reuse the skill; Check must not mutate it.
	s := load(allSegments, triggerDesc)
	body, desc := s.Body, s.Description
	redlines.Check(s)
	if s.Body != body || s.Description != desc {
		t.Error("Check mutated the skill it was given")
	}
}

func TestCheckSkipsTheTriggerWhenFrontmatterDidNotParse(t *testing.T) {
	t.Parallel()
	// The description is empty because the YAML failed, not because the author
	// omitted a trigger. The body-derived red lines must still fire: a blanket
	// suppression would hide the over-long quotation, which is a real defect.
	s := load(allSegments+"\n> "+strings.Repeat("word ", redlines.MaxQuoteWords+1), "")
	s.FrontmatterErr = errors.New("[10:45] value is not allowed in this context")

	got := redlines.Check(s)
	if len(got) != 1 {
		t.Fatalf("expected only the quotation defect, got %+v", got)
	}
	if !strings.Contains(got[0].Message, "over the 150-word limit") {
		t.Errorf("the body-derived red line must survive, got %q", got[0].Message)
	}
	for _, d := range got {
		if strings.Contains(d.Message, "trigger condition") {
			t.Error("must not demand a trigger from a description that could not be read")
		}
	}
}

func TestCheckStillDemandsATriggerWhenTheBlockParsed(t *testing.T) {
	t.Parallel()
	// The over-suppression trap: a file with no frontmatter at all parses fine
	// (yaml.Unmarshal("") succeeds), so an author who really did omit the
	// description must still be told. Silencing this would trade a false positive
	// for a false negative.
	got := redlines.Check(load(allSegments, ""))
	found := false
	for _, d := range got {
		if strings.Contains(d.Message, "trigger condition") {
			found = true
		}
	}
	if !found {
		t.Errorf("a genuinely absent description must still be reported, got %+v", got)
	}
}

func TestQuotes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		body string
		want []string
	}{
		"no blockquote": {body: "plain prose\n\nmore prose"},
		"one run joins its lines": {
			body: "intro\n> first line\n> second line\nafter",
			want: []string{"first line second line"},
		},
		"a blank line ends the run": {
			body: "> alpha\n\n> beta",
			want: []string{"alpha", "beta"},
		},
		"runs come back in document order": {
			body: "> one\ntext\n> two\ntext\n> three",
			want: []string{"one", "two", "three"},
		},
		// The red line counts words, so a ">" line inside a shell transcript would
		// otherwise inflate a quotation that is not one.
		"a fenced block is not a quotation": {
			body: "```\n> not a quote\n> still not\n```\n> a real one",
			want: []string{"a real one"},
		},
		"tilde fences count too": {
			body: "~~~\n> nope\n~~~\nprose",
		},
		"bare markers carry no text": {body: ">\n>\n>"},
		"indented markers still count": {
			body: "  > indented quote",
			want: []string{"indented quote"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := redlines.Quotes(tc.body)
			if len(got) != len(tc.want) {
				t.Fatalf("Quotes returned %d runs %q, want %d %q",
					len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("run %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestQuotesAgreesWithTheRedLineItBacks(t *testing.T) {
	t.Parallel()
	// The reason Quotes is exported: a caller must be able to reproduce exactly which
	// runs the MaxQuoteWords red line will complain about.
	long := strings.Repeat("word ", redlines.MaxQuoteWords+1)
	body := "> short quote\n\n> " + long + "\n\nprose"
	quotes := redlines.Quotes(body)
	if len(quotes) != 2 {
		t.Fatalf("expected 2 runs, got %d: %q", len(quotes), quotes)
	}
	over := 0
	for _, q := range quotes {
		if len(strings.Fields(q)) > redlines.MaxQuoteWords {
			over++
		}
	}
	s := &skill.Skill{Body: body, Description: "Use when x. Trigger on y."}
	ds := 0
	for _, d := range redlines.Check(s) {
		if strings.Contains(d.Message, "over the") {
			ds++
		}
	}
	if over != ds {
		t.Errorf("Quotes says %d runs are over the limit, Check reports %d", over, ds)
	}
}

// wantSegmentDiagnostics counts the RIA-segment and unknown-lineage diagnostics a skill
// collects. The helper exists so each case below reads as a claim about a lineage rather
// than as diagnostic filtering.
func wantSegmentDiagnostics(t *testing.T, lineage, body string, wantSeg, wantUnknown int) {
	t.Helper()
	raw, _ := skill.ParseLineage(lineage)
	s := &skill.Skill{
		Description: "Use when the reader needs a worked example.",
		Body:        body,
		Lineage:     raw,
		LineageRaw:  lineage,
	}
	var seg, unknown int
	for _, d := range redlines.Check(s) {
		switch {
		case strings.Contains(d.Message, "RIA segment"):
			seg++
		case strings.Contains(d.Message, "unknown "+skill.LineagePath):
			unknown++
		}
	}
	if seg != wantSeg {
		t.Errorf("lineage %q: %d segment diagnostics, want %d", lineage, seg, wantSeg)
	}
	if unknown != wantUnknown {
		t.Errorf("lineage %q: %d unknown-lineage diagnostics, want %d",
			lineage, unknown, wantUnknown)
	}
}

// TestSegmentContractKeysOnDeclaredLineage. Measured over a 233-skill corpus, the
// unguarded contract reported six diagnostics each for 48 hand-written skills about a
// format they never claimed. The exemption is declared, never inferred: a hand-written
// skill with no segments and a malformed book skill that shed its segments look identical
// from the body and want opposite treatment.
func TestSegmentContractKeysOnDeclaredLineage(t *testing.T) {
	t.Parallel()
	const prose = "# A Tool Skill\n\nRun the thing, then read the output.\n"
	tests := []struct {
		name        string
		lineage     string
		wantSeg     int
		wantUnknown int
	}{{
		name:    "a declared hand-written skill is not graded on segments",
		lineage: "hand-written", wantSeg: 0, wantUnknown: 0,
	}, {
		name:    "a declared book-derived skill is graded on all six",
		lineage: "book-derived", wantSeg: 6, wantUnknown: 0,
	}, {
		// The zero value must not assert. Treating absence as hand-written would hand
		// every skill an escape by omission, which is the same hole as a book skill
		// shedding headings until it looks like something else.
		name:    "an undeclared skill is still graded on all six",
		lineage: "", wantSeg: 6, wantUnknown: 0,
	}, {
		// Strictest treatment *and* a report. A near-miss must not buy lenience, and it
		// must not fail closed in silence either.
		//
		// The value is a missing hyphen rather than a misspelling on purpose: the
		// misspell linter autofixes a literal typo in a string, which silently turned an
		// earlier version of this case into a copy of the passing one above.
		name:    "a near-miss value is graded strictly and reported",
		lineage: "handwritten", wantSeg: 6, wantUnknown: 1,
	}, {
		name:    "an unrecognised newer vocabulary is handled the same way",
		lineage: "machine-distilled", wantSeg: 6, wantUnknown: 1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wantSegmentDiagnostics(t, tt.lineage, prose, tt.wantSeg, tt.wantUnknown)
		})
	}
}

// TestLineageDoesNotExemptTheOtherRedLines. Only the segment contract is lineage-scoped:
// an over-long quotation is a defect whichever way the document was produced, and every
// skill has a description to state a trigger in.
func TestLineageDoesNotExemptTheOtherRedLines(t *testing.T) {
	t.Parallel()
	long := "> " + strings.Repeat("word ", 200) + "\n"
	s := &skill.Skill{
		Description: "a summary with no trigger",
		Body:        "# T\n\n" + long,
		Lineage:     skill.HandWritten,
		LineageRaw:  "hand-written",
	}
	var quote, trigger int
	for _, d := range redlines.Check(s) {
		if strings.Contains(d.Message, "quotation") {
			quote++
		}
		if strings.Contains(d.Message, "trigger") {
			trigger++
		}
	}
	if quote == 0 {
		t.Error("a hand-written skill's over-long quotation went unreported")
	}
	if trigger == 0 {
		t.Error("a hand-written skill's description was not asked for a trigger")
	}
}
