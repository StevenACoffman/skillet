package ruleset_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset"
)

const v1Doc = "Source: s\nScope:  x\n\n§1.1  [MUST][CODE]  Close it.\n      because reasons\n"

// TestUndeclaredFormatIsVersionOne is the migration story: every ruleset written before
// versioning existed reads as v1 without being touched.
func TestUndeclaredFormatIsVersionOne(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rs.Format != 1 {
		t.Errorf("Format = %d, want 1 for a file declaring none", rs.Format)
	}
}

// TestVersionOneRendersNoBlock is the property the whole change rests on. If Render started
// emitting a block, every stored ruleset would report drift the moment canonizer's
// canonical-form check lands — the new feature manufacturing the failure the next feature
// exists to detect.
func TestVersionOneRendersNoBlock(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out := ruleset.Render(rs)
	if strings.Contains(out, "---") {
		t.Errorf("v1 rendered a version block:\n%s", out)
	}
	if out != v1Doc {
		t.Errorf("v1 did not round-trip byte-identically:\n got %q\nwant %q", out, v1Doc)
	}
	// A hand-built Ruleset that never set Format must render as v1 too, or every Go caller
	// has to remember a field that has one sensible value.
	zero := rs
	zero.Format = 0
	if ruleset.Render(zero) != out {
		t.Error("Format 0 and Format 1 rendered differently")
	}
}

// TestFutureFormatIsRefused is the behaviour being bought. Without it an unknown marker in a
// newer file is folded into a rule's rationale and nothing says so.
func TestFutureFormatIsRefused(t *testing.T) {
	t.Parallel()
	future := "---\nformat: " + strconv.Itoa(ruleset.FormatVersion+1) + "\n---\n" + v1Doc
	_, err := ruleset.Parse(future)
	if err == nil {
		t.Fatal("a newer format parsed without complaint")
	}
	// The message must name both versions: "it failed" does not tell an operator whether to
	// upgrade the tool or fix the file.
	for _, want := range []string{strconv.Itoa(ruleset.FormatVersion + 1), strconv.Itoa(ruleset.FormatVersion)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name version %s", err, want)
		}
	}
}

func TestMalformedFormatIsAnError(t *testing.T) {
	t.Parallel()
	for name, block := range map[string]string{
		"negative":     "---\nformat: -1\n---\n",
		"not a number": "---\nformat: two\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := ruleset.Parse(block + v1Doc); err == nil {
				t.Error("parsed without complaint")
			}
		})
	}
}

// TestBlockWithoutFormatIsVersionOne keeps the door open for other metadata: a block that
// declares no format is a v1 ruleset carrying something else, not a malformed version.
func TestBlockWithoutFormatIsVersionOne(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse("---\nnote: hello\n---\n" + v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rs.Format != 1 {
		t.Errorf("Format = %d, want 1", rs.Format)
	}
}

// TestFormatVersionTracksTheMarkerSet is the check markers()' own doc comment says exists.
// It did not: ruleset_test.go asserts every marker is non-ASCII, format_test.go covers the
// version reader, and nothing tied the two together. A comment describing an intention as
// though it were a check is the thing this table turns back into one.
//
// A new marker is a grammar change, which is FormatVersion's documented bump trigger, and
// the failure it prevents is silent: a v1 reader given an unknown marker rejects the line,
// so a corpus written by a newer tool becomes unreadable with no version to explain why.
func TestFormatVersionTracksTheMarkerSet(t *testing.T) {
	t.Parallel()
	markersAt := map[int]int{
		1: 3, // ✗ ✓ ↦
		2: 4, // + ⚖
	}
	want, recorded := markersAt[ruleset.FormatVersion]
	if !recorded {
		t.Fatalf("FormatVersion is %d and no marker count is recorded for it; "+
			"add one so the next marker cannot be added without deciding this",
			ruleset.FormatVersion)
	}
	if got := len(ruleset.MarkerPrefixesForTest()); got != want {
		t.Errorf("FormatVersion %d expects %d markers, found %d: the body vocabulary "+
			"changed without a version bump", ruleset.FormatVersion, want, got)
	}
}

// TestAWarrantFreeRulesetStillRendersNoBlock is the inert property, and the reason the
// version reader shipped a release before the first marker that needed it. Every ruleset
// written before warrants existed must render byte-identically, or the canonical-form drift
// check reports a change on files nobody touched -- the new feature manufacturing the
// failure the next feature exists to detect.
func TestAWarrantFreeRulesetStillRendersNoBlock(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	got := ruleset.Render(rs)
	if strings.Contains(got, "format:") {
		t.Errorf("a warrant-free ruleset declared a version:\n%s", got)
	}
	if got != v1Doc {
		t.Errorf("round-trip is not byte-identical:\n got %q\nwant %q", got, v1Doc)
	}
}

// TestARulesetWithAWarrantDeclaresVersionTwo is the other half: the declared version is
// derived from what the document uses, so it cannot contradict its own content.
func TestARulesetWithAWarrantDeclaresVersionTwo(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	if len(rs.Rules) == 0 {
		t.Fatal("the fixture has no rules to adjudicate")
	}
	// Format is left at whatever the v1 document said: the point is that content decides.
	rs.Rules[0].Warrant = ruleset.Warrant{
		By: "steve@khanacademy.org", At: "2026-08-27", Rationale: "two MUSTs disagreed",
	}
	got := ruleset.Render(rs)
	if !strings.HasPrefix(got, "---\nformat: 2\n---\n") {
		t.Errorf("a ruleset carrying a warrant did not declare version 2:\n%s", got)
	}
	if !strings.Contains(got, "⚖  steve@khanacademy.org 2026-08-27  two MUSTs disagreed") {
		t.Errorf("the warrant line is not in the canonical shape:\n%s", got)
	}
	back, err := ruleset.Parse(got)
	if err != nil {
		t.Fatalf("a document this package wrote does not parse: %v\n%s", err, got)
	}
	if back.Rules[0].Warrant != rs.Rules[0].Warrant {
		t.Errorf("warrant did not round-trip: %+v, want %+v",
			back.Rules[0].Warrant, rs.Rules[0].Warrant)
	}
}

// TestAHalfRecordedWarrantIsRefused is the rule that a warrant without all three parts is
// worse than none: it is the only record of a decision carrying no other evidence, so a
// partial one looks like provenance while establishing nothing.
func TestAHalfRecordedWarrantIsRefused(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"no rationale":               "⚖  steve 2026-08-27",
		"no date, no reason":         "⚖  steve",
		"nothing at all":             "⚖",
		"date is not a date":         "⚖  steve last-tuesday because",
		"date in the wrong spelling": "⚖  steve 27-08-2026 because",
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := "Source: x\nScope:  y\n\n§1.1  [MUST][CODE]  Close rows\n    " + line + "\n"
			if _, err := ruleset.Parse(doc); err == nil {
				t.Errorf("a half-recorded warrant was accepted:\n%s", doc)
			}
		})
	}
}
