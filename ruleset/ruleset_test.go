package ruleset_test

import (
	"reflect"
	"testing"

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
