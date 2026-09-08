package ruleset_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset"
	"github.com/StevenACoffman/skillet/verification"
)

// verifiedBody is the rule text every case in this file shares, so a diff between an
// expected and an actual rendering is always about the frontmatter block.
const verifiedBody = "Source: s\nScope:  x\n\n§1.1  [MUST][CODE]  Close it.\n" +
	"      because reasons\n"

// TestAVerifiedRulesetRoundTripsThroughBothDirections is the property the whole slot rests
// on: the block is hand-written and read by a real YAML parser, so writer and reader can
// drift in a way neither side's own tests would notice. Every case renders, parses the
// rendering, and renders again -- the second rendering must equal the first, and the parsed
// events must equal what went in.
func TestAVerifiedRulesetRoundTripsThroughBothDirections(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		events []verification.Event
	}{{
		name:   "an actor and a date",
		events: []verification.Event{{By: "human:steve", At: "2026-09-08"}},
	}, {
		name:   "an undeclared date is omitted rather than emitted empty",
		events: []verification.Event{{By: "check:lint"}},
	}, {
		name: "several events keep the order they were given",
		events: []verification.Event{
			{By: "human:zoe", At: "2026-01-01"},
			{By: "check:lint"},
			{By: "human:abe", At: "2026-12-31"},
		},
	}, {
		name:   "an actor holding a colon, which is the ordinary case",
		events: []verification.Event{{By: "human:steve"}},
	}, {
		name:   "an actor holding a quote",
		events: []verification.Event{{By: `a"b`}},
	}, {
		name:   "an actor holding a backslash",
		events: []verification.Event{{By: `a\b`}},
	}, {
		name:   "an actor holding a colon and a space, which plain YAML would split",
		events: []verification.Event{{By: "x: y"}},
	}, {
		name:   "an actor holding a line break",
		events: []verification.Event{{By: "line\nbreak"}},
	}, {
		// Render is not a validator and returns no error, so it cannot refuse this. An
		// event with no actor records nothing, but dropping the element would be silent
		// data loss; whether it is *valid* belongs to a gate, not to the renderer.
		name:   "an actorless event is carried rather than dropped",
		events: []verification.Event{{At: "2026-09-08"}},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertRoundTrip(t, c.events)
		})
	}
}

// assertRoundTrip renders events, parses that rendering, and renders again: the events must
// survive unchanged and the second rendering must equal the first.
func assertRoundTrip(t *testing.T, events []verification.Event) {
	t.Helper()

	in := ruleset.Ruleset{Source: "s", Scope: "x", Verified: events, Rules: rule()}
	first := ruleset.Render(&in)

	back, err := ruleset.Parse(first)
	if err != nil {
		t.Fatalf("Parse of our own rendering: %v\n%s", err, first)
	}
	if got, want := len(back.Verified), len(events); got != want {
		t.Fatalf("parsed %d events, want %d\n%s", got, want, first)
	}
	for i := range events {
		if back.Verified[i] != events[i] {
			t.Errorf("event %d = %+v, want %+v", i, back.Verified[i], events[i])
		}
	}
	if second := ruleset.Render(&back); second != first {
		t.Errorf("re-rendering differs:\n first=%q\nsecond=%q", first, second)
	}
}

// TestAnUndeclaredDateEmitsNoAtKey pins the asymmetry Event's own json tags describe: a
// verification whose date was never declared is still a verification, so the key is absent
// rather than present and empty.
func TestAnUndeclaredDateEmitsNoAtKey(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{
		Source: "s", Scope: "x",
		Verified: []verification.Event{{By: "check:lint"}},
		Rules:    rule(),
	}
	got := ruleset.Render(&rs)
	// Matched with its indentation: a bare "at:" also occurs inside "format:".
	if strings.Contains(got, "\n    at:") {
		t.Errorf("rendering carries an at: key for an undeclared date:\n%s", got)
	}
	if !strings.Contains(got, `- by: "check:lint"`) {
		t.Errorf("rendering lost the actor:\n%s", got)
	}
}

// TestVerifiedAloneReachesVersionFour is the ordering formatOf depends on. A verified
// ruleset that declares no Limitations: must not fall through to the version-3 clause.
func TestVerifiedAloneReachesVersionFour(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{
		Source: "s", Scope: "x",
		Verified: []verification.Event{{By: "human:steve"}},
		Rules:    rule(),
	}
	if got := ruleset.Render(&rs); !strings.HasPrefix(got, "---\nformat: 4\n") {
		t.Errorf("want a version 4 block, got:\n%s", got)
	}
}

// TestAnUnverifiedRulesetRendersNoVerifiedKey is the inert property, and it is the reason
// the eight stored rulesets keep rendering byte-identically across this bump: a document
// that uses nothing version 4 introduced is not written at version 4.
func TestAnUnverifiedRulesetRendersNoVerifiedKey(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(verifiedBody)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := ruleset.Render(&rs)
	if strings.Contains(got, "verified") {
		t.Errorf("an unverified ruleset gained a verified key:\n%s", got)
	}
	if got != verifiedBody {
		t.Errorf("rendering is not byte-identical:\n got=%q\nwant=%q", got, verifiedBody)
	}
}

// rule returns the one rule every case here shares.
func rule() []ruleset.Rule {
	return []ruleset.Rule{{
		Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE,
		Statement: "Close it.", Rationale: "because reasons",
	}}
}
