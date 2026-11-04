package ruleset_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/judge"
	"github.com/StevenACoffman/skillet/ruleset"
)

// githubRuleset exercises every part of the version-5 layout: references to rules that
// exist and to one that does not, a reference inside a code span, an anchor citing a
// numbered source section, a field ending in a backslash, a warrant and a check.
func githubRuleset() ruleset.Ruleset {
	return ruleset.Ruleset{
		Format: 5,
		Source: "A Book",
		Scope:  "Go",
		Rules: []ruleset.Rule{{
			Section: "1.1", Severity: ruleset.MUST, Level: ruleset.CODE,
			Statement:    "Close what you open (see §1.2, and §10.4, which was deleted).",
			Rationale:    "A leaked descriptor exhausts the table; `§1.2` in code stays literal.",
			Bad:          `path := "C:\temp\"`,
			Good:         "defer f.Close() as §1.2 requires",
			SourceAnchor: `§11.1.2 Pattern 2: "Relying solely on the gateway creates blind spots."`,
		}, {
			Section: "1.2", Severity: ruleset.SHOULD, Level: ruleset.ARCH,
			Statement: `Keep a literal link [§1.1](#r1.1) as written, and end in a backslash\`,
			Warrant:   ruleset.Warrant{By: "steve", At: "2026-11-03", Rationale: "no source says"},
			Checks:    []judge.Check{{Op: judge.OpContains, Arg: "Close"}},
		}},
	}
}

func TestAVersionFiveRulesetRoundTrips(t *testing.T) {
	t.Parallel()
	rs := githubRuleset()
	rendered := ruleset.Render(&rs)
	got, err := ruleset.Parse(rendered)
	if err != nil {
		t.Fatalf("Parse(Render(rs)): %v\n%s", err, rendered)
	}
	if !reflect.DeepEqual(got, rs) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v\nrendered:\n%s", got, rs, rendered)
	}
	if again := ruleset.Render(&got); again != rendered {
		t.Errorf("Render is not stable over a round trip:\n%s\nthen\n%s", rendered, again)
	}
}

// TestTheVersionFiveLayoutReadsInAViewer pins what the layout is for. Each case is a
// substring the rendered document must or must not contain.
func TestTheVersionFiveLayoutReadsInAViewer(t *testing.T) {
	t.Parallel()
	rs := githubRuleset()
	rendered := ruleset.Render(&rs)

	cases := []struct {
		name string
		text string
		want bool
	}{
		{"the format is declared", "format: 5", true},
		{"each header carries its anchor", `<a id="r1.1"></a>`, true},
		{
			"a header followed by a body line ends in a hard break",
			`<a id="r1.1"></a>\` + "\n",
			true,
		},
		{"a reference to a rule here is linked", "[§1.2](#r1.2)", true},
		{"a reference to a missing rule stays plain", "§10.4, which", true},
		{"a missing rule is not linked", "#r10.4", false},
		{"a reference in a code span stays literal", "`§1.2` in code", true},
		{"an anchor's numbered source section is not linked", "[§11.1.2", false},
		{"a rule's last line has no hard break", "Close\n", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := strings.Contains(rendered, c.text); got != c.want {
				t.Errorf("contains %q = %v, want %v\n%s", c.text, got, c.want, rendered)
			}
		})
	}
}

// TestADeclaredFiveSurvivesAVerifiedBlock covers the ordering in formatOf: a verification
// event requires version 4, and must not pull a document that chose 5 back to it.
func TestADeclaredFiveSurvivesAVerifiedBlock(t *testing.T) {
	t.Parallel()
	doc := "---\nformat: 5\nverified:\n  - by: human:steve\n    at: 2026-11-03\n---\n" +
		"Source: x\nScope:  y\n\n§1.1  [MUST][CODE]  Close rows<a id=\"r1.1\"></a>\n"
	rs, err := ruleset.Parse(doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rs.Format != 5 || len(rs.Verified) != 1 {
		t.Fatalf("Format = %d, Verified = %d events; want 5 and 1", rs.Format, len(rs.Verified))
	}
	if out := ruleset.Render(&rs); !strings.HasPrefix(out, "---\nformat: 5\n") {
		t.Errorf("rendered at another version:\n%s", out)
	}
}
