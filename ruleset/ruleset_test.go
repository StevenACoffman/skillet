package ruleset_test

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/StevenACoffman/skillet/ruleset"
)

func TestSeverityLevelValid(t *testing.T) {
	t.Parallel()
	if !ruleset.MUST.Valid() || !ruleset.CONSIDER.Valid() || ruleset.Severity("MAYBE").Valid() {
		t.Error("Severity.Valid wrong")
	}
	if !ruleset.CODE.Valid() || !ruleset.METHOD.Valid() || ruleset.Level("UI").Valid() {
		t.Error("Level.Valid wrong")
	}
}

func TestRenderParseRoundTrip(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{
		Source: "Ben Johnson: Standard Package Layout",
		Scope:  "Go application structure",
		Rules: []ruleset.Rule{
			{
				Section:      "2.3",
				Severity:     ruleset.MUST,
				Level:        ruleset.CODE,
				Statement:    "Never discard an error return without an explicit decision.",
				Rationale:    "Silently dropping errors removes the caller's only failure signal.",
				Bad:          "result, _ = db.Exec(query)",
				Good:         "result, err = db.Exec(query); if err != nil { return err }",
				SourceAnchor: `§Errors: "never ignore the value returned by a function"`,
			},
			{
				Section:   "3.1",
				Severity:  ruleset.SHOULD,
				Level:     ruleset.ARCH,
				Statement: "Keep the root package free of third-party imports.",
			},
		},
	}
	// Parse resolves an undeclared file to version 1, so the fixture states it: a v1 ruleset
	// is what this is, and leaving Format at 0 would make the round-trip compare a resolved
	// value against an unset one.
	rs.Format = 1
	got, err := ruleset.Parse(ruleset.Render(rs))
	if err != nil {
		t.Fatalf("Parse(Render(rs)): %v", err)
	}
	if !reflect.DeepEqual(got, rs) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, rs)
	}
}

func TestRenderDeterministic(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{Source: "s", Scope: "sc", Rules: []ruleset.Rule{
		{Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE, Statement: "do X"},
	}}
	first := ruleset.Render(rs)
	if again := ruleset.Render(rs); first != again {
		t.Error("Render must be deterministic")
	}
}

func TestParseRejectsBadSeverity(t *testing.T) {
	t.Parallel()
	if _, err := ruleset.Parse("§1.1  [WISH][CODE]  do the thing\n"); err == nil {
		t.Fatal("unknown severity should error")
	}
	if _, err := ruleset.Parse("§1.1  [MUST][GUI]  do the thing\n"); err == nil {
		t.Fatal("unknown level should error")
	}
}

// TestAnUnknownMarkerIsNotRationale. The canonical form's body has three markers and a
// default case that is rationale continuation, so a line the parser does not recognise
// used to be appended to the rationale silently — a file written by a newer version
// mis-parsed in an older one while appearing to work.
//
// The cross-version half of that was already closed by the format header, which refuses
// a version newer than this parser understands. This closes the same-format half: a
// typo, a hand-edit, or a paste from a newer document without its header.
func TestAnUnknownMarkerIsNotRationale(t *testing.T) {
	t.Parallel()
	const header = "§4.1  [MUST][CODE]  Close all rows\n"
	cases := map[string]struct {
		body    string
		wantErr bool
		want    string // expected rationale when no error
	}{
		"an unknown symbol marker": {
			"  because leaks are silent\n  ⊕  timeout <= 30s\n", true, "",
		},
		// The case that decided how narrow the rule is. Rejecting all unknown
		// punctuation would reject each of these, and they are prose somebody writes.
		"a rationale opening with an em dash": {
			"  — because leaks are silent\n", false, "— because leaks are silent",
		},
		"a rationale opening with a curly quote": {
			"  “because leaks are silent”\n", false, "“because leaks are silent”",
		},
		"a rationale opening with a parenthesis": {
			"  (see §4) leaks are silent\n", false, "(see §4) leaks are silent",
		},
		"a multi-line rationale": {
			"  because leaks are silent\n  and only visible under load\n", false,
			"because leaks are silent and only visible under load",
		},
		"a known marker still works": {
			"  because leaks are silent\n  ✓ defer rows.Close()\n", false,
			"because leaks are silent",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rs, err := ruleset.Parse("Source: x\n\n" + header + tc.body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error: %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(err.Error(), "unrecognised marker") {
					t.Errorf("the error does not name the problem: %v", err)
				}
				return
			}
			if got := rs.Rules[0].Rationale; got != tc.want {
				t.Errorf("Rationale = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestANewerFormatIsRefused pins the half of this that was already built, because the
// backlog entry describing the defect did not know it existed and a future reader
// should not have to rediscover it.
func TestANewerFormatIsRefused(t *testing.T) {
	t.Parallel()
	_, err := ruleset.Parse("---\nformat: 2\n---\nSource: x\n\n§4.1  [MUST][CODE]  Close rows\n")
	if err == nil {
		t.Fatal("a format newer than this parser understands was accepted")
	}
	if !strings.Contains(err.Error(), "newer than this parser understands") {
		t.Errorf("the error does not say why: %v", err)
	}
}

// TestEveryMarkerIsNonASCII is what makes the rejection rule sound. It rejects a line
// opening with an unknown *symbol*, so a marker added in ASCII would slip past it
// silently. The form should not add one, and this is that intention as a checked claim.
func TestEveryMarkerIsNonASCII(t *testing.T) {
	t.Parallel()
	for _, m := range ruleset.MarkerPrefixesForTest() {
		first, _ := utf8.DecodeRuneInString(m)
		if first < utf8.RuneSelf {
			t.Errorf("marker %q is ASCII; applyBody's rejection cannot see an ASCII marker", m)
		}
		if !unicode.IsSymbol(first) {
			t.Errorf(
				"marker %q does not open with a symbol, so a line using it reads as rationale",
				m,
			)
		}
	}
}
