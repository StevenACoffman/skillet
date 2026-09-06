package ruleset_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/judge"
	"github.com/StevenACoffman/skillet/ruleset"
)

// checked builds a rule carrying one contains-check plus the two examples.
func checked(bad, good, arg string) ruleset.Rule {
	return ruleset.Rule{
		Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE,
		Statement: "close it", Bad: bad, Good: good,
		Checks: []judge.Check{{Op: judge.OpContains, Arg: arg}},
	}
}

func TestSound(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		rule        ruleset.Rule
		wantUnsound bool
		wantReason  string
	}{
		// The check fires on the ✗ and not on the ✓, which is the whole contract.
		"a discriminating check is sound": {
			rule: checked(
				"conn.Close() // never deferred",
				"defer conn.Close()",
				"// never deferred",
			),
		},
		"a check that misses its own bad example does not flag what it exists to flag": {
			rule:        checked("conn.Close()", "defer conn.Close()", "ABSENT"),
			wantUnsound: true, wantReason: "✗ example",
		},
		// The measured trap: ✓ contains ✗ in the commonest shape a fix takes, so a check
		// written as "contains the bad text" passes on both and discriminates nothing.
		"a check that passes on both does not discriminate": {
			rule:        checked("conn.Close()", "defer conn.Close()", "conn.Close()"),
			wantUnsound: true, wantReason: "discriminate",
		},
		"checks with no examples cannot be run": {
			rule: ruleset.Rule{
				Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE,
				Statement: "close it",
				Checks:    []judge.Check{{Op: judge.OpContains, Arg: "x"}},
			},
			wantUnsound: true, wantReason: "not both examples",
		},
		// Untested is not unsound: a rule with no checks makes no claim this can refute.
		"a rule with no checks is not reported": {
			rule: ruleset.Rule{
				Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE,
				Statement: "close it", Bad: "a", Good: "b",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSound(t, ruleset.Sound(&ruleset.Ruleset{Rules: []ruleset.Rule{tc.rule}}),
				tc.wantUnsound, tc.wantReason)
		})
	}
}

// TestChecksRoundTripAndDeclareVersionThree is the marker's half of the v3 bump.
func TestChecksRoundTripAndDeclareVersionThree(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{
		Source: "s", Scope: "x",
		Rules: []ruleset.Rule{checked("bad text", "good text", "bad")},
	}
	got := ruleset.Render(&rs)
	if !strings.HasPrefix(got, "---\nformat: 3\n---\n") {
		t.Errorf("a ruleset carrying checks did not declare version 3:\n%s", got)
	}
	if !strings.Contains(got, "⊨  contains  bad") {
		t.Errorf("the check line is not in the canonical shape:\n%s", got)
	}
	back, err := ruleset.Parse(got)
	if err != nil {
		t.Fatalf("a document this package wrote does not parse: %v\n%s", err, got)
	}
	if len(back.Rules[0].Checks) != 1 || back.Rules[0].Checks[0] != rs.Rules[0].Checks[0] {
		t.Errorf("checks did not round-trip: %+v", back.Rules[0].Checks)
	}
}

// TestSeveralChecksAccumulate pins the one marker in this form that appends rather than
// assigns: a rule's checks are a conjunction and each is written on its own line.
func TestSeveralChecksAccumulate(t *testing.T) {
	t.Parallel()
	doc := "Source: s\nScope:  x\n\n§1.1  [MUST][CODE]  Close it.\n" +
		"      ⊨  contains  alpha\n      ⊨  regex  beta.*\n"
	rs, err := ruleset.Parse(doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rs.Rules[0].Checks) != 2 {
		t.Fatalf("checks = %+v, want both lines kept", rs.Rules[0].Checks)
	}
	if rs.Rules[0].Checks[0].Op != judge.OpContains || rs.Rules[0].Checks[1].Op != judge.OpRegex {
		t.Errorf("checks are not in document order: %+v", rs.Rules[0].Checks)
	}
}

func TestAMalformedCheckIsRefused(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		// A wrong operator would otherwise become a check that silently never passes.
		// Written as a plausible near-miss of a real op ("regex") rather than as a
		// dictionary misspelling: `misspell --fix` rewrites a typo inside a fixture and
		// quietly turned this case into the valid one on the first run.
		"unknown operator":     "⊨  regexp  alpha",
		"operator with no arg": "⊨  contains",
		"nothing at all":       "⊨",
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := "Source: s\nScope:  x\n\n§1.1  [MUST][CODE]  Close it.\n      " + line + "\n"
			if _, err := ruleset.Parse(doc); err == nil {
				t.Errorf("a malformed check was accepted:\n%s", doc)
			}
		})
	}
}

// assertSound checks the single finding, or that there is none. Extracted to keep the table
// flat: the sound branch plus three assertions inside a subtest is what pushes it over the
// complexity cap.
func assertSound(t *testing.T, got []ruleset.Unsound, wantUnsound bool, wantReason string) {
	t.Helper()
	if !wantUnsound {
		if len(got) != 0 {
			t.Fatalf("Sound = %+v, want none", got)
		}
		return
	}
	if len(got) != 1 {
		t.Fatalf("Sound = %+v, want one finding", got)
	}
	if got[0].Section != "1.1" {
		t.Errorf("Section = %q, want the rule identified", got[0].Section)
	}
	if !strings.Contains(got[0].Reason, wantReason) {
		t.Errorf("Reason = %q, want it to mention %q", got[0].Reason, wantReason)
	}
}
