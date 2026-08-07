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
