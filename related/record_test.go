package related_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/related"
)

// TestEdgeMapRoundTripsKindAndTargetAndDropsTheRationale pins the contract in the
// direction it is actually true. The manifest deliberately keeps no rationale, so a test
// asserting a node survives storage unchanged would assert something the storage decision
// rejected — and would fail for the right reason in the wrong words.
func TestEdgeMapRoundTripsKindAndTargetAndDropsTheRationale(t *testing.T) {
	t.Parallel()
	in := []related.Edge{
		{Kind: related.DependsOn, Target: "b", Rationale: "b comes first"},
		{Kind: related.ComposesWith, Target: "c", Rationale: "used together"},
		{Kind: related.DependsOn, Target: "a", Rationale: "and a"},
	}
	want := []related.Edge{
		{Kind: related.ComposesWith, Target: "c"},
		{Kind: related.DependsOn, Target: "a"},
		{Kind: related.DependsOn, Target: "b"},
	}
	got := related.EdgesFrom(related.EdgeMap(in))
	if len(got) != len(want) {
		t.Fatalf("got %d edges %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("edge %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// The input must be readable afterwards: a producer walks a tree once and may hand
	// the same slice to more than one consumer.
	if in[0].Rationale != "b comes first" {
		t.Errorf("EdgeMap mutated its input: %+v", in[0])
	}
}

// renderMap writes a recorded edge map as "kind=a,b;kind=c;", kinds ascending, so a case
// states its expectation as one readable line instead of a nested loop.
func renderMap(m map[string][]string) string {
	kinds := make([]string, 0, len(m))
	for kind := range m {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	var out strings.Builder
	for _, kind := range kinds {
		out.WriteString(kind + "=" + strings.Join(m[kind], ",") + ";")
	}
	return out.String()
}

// TestEdgeMapTable covers the shapes a real manifest reaches.
func TestEdgeMapTable(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in   []related.Edge
		want string
	}{
		"targets are sorted within a kind": {
			[]related.Edge{
				{Kind: related.DependsOn, Target: "z"},
				{Kind: related.DependsOn, Target: "a"},
			},
			"depends-on=a,z;",
		},
		// The reader already dedupes on kind+target; a manifest recording the same
		// relationship twice would make one bullet look like two.
		"a repeated relationship is recorded once": {
			[]related.Edge{
				{Kind: related.DependsOn, Target: "a", Rationale: "one wording"},
				{Kind: related.DependsOn, Target: "a", Rationale: "another"},
			},
			"depends-on=a;",
		},
		// Same target under two kinds is two relationships, not a duplicate.
		"one target under two kinds is two records": {
			[]related.Edge{
				{Kind: related.DependsOn, Target: "a"},
				{Kind: related.Informs, Target: "a"},
			},
			"depends-on=a;informs=a;",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := renderMap(related.EdgeMap(tc.in)); got != tc.want {
				t.Errorf("EdgeMap = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEdgeMapRecordsNoEdgesAsNil is a claim about the JSON, not about the contents, which
// is why it is not a row in the table above: the field is omitempty, and an empty map and
// a nil one render differently. Without this, every entry of a 285-skill manifest whose
// skill declares no edges writes `"edges": {}`.
func TestEdgeMapRecordsNoEdgesAsNil(t *testing.T) {
	t.Parallel()
	for name, in := range map[string][]related.Edge{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := related.EdgeMap(in); got != nil {
				t.Errorf("EdgeMap = %v, want nil", got)
			}
		})
	}
}

// TestEdgesFromIsOrderedIndependentlyOfMapIteration guards the one nondeterminism the
// stored shape introduces. Without an explicit order, two reads of one manifest produce
// differently ordered edges and any comparison against them is flaky for no reason in the
// data.
func TestEdgesFromIsOrderedIndependentlyOfMapIteration(t *testing.T) {
	t.Parallel()
	m := map[string][]string{
		"informs":       {"z", "y"},
		"depends-on":    {"c", "b"},
		"composes-with": {"d"},
	}
	want := []related.Edge{
		{Kind: related.ComposesWith, Target: "d"},
		{Kind: related.DependsOn, Target: "c"},
		{Kind: related.DependsOn, Target: "b"},
		{Kind: related.Informs, Target: "z"},
		{Kind: related.Informs, Target: "y"},
	}
	for range 20 {
		got := related.EdgesFrom(m)
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("edge %d = %+v, want %+v", i, got[i], want[i])
			}
		}
	}
}

// TestEdgesFromKeepsAKindItDoesNotKnow. Dropping it would make the reconstituted graph
// smaller than the document plainly says it is, and the reader would have no way to tell
// that from a manifest that genuinely recorded less. What an unknown kind is worth is the
// ranking consumer's decision, taken where the ranking is.
func TestEdgesFromKeepsAKindItDoesNotKnow(t *testing.T) {
	t.Parallel()
	got := related.EdgesFrom(map[string][]string{"verifies": {"a"}})
	if len(got) != 1 || got[0].Kind != related.Kind("verifies") || got[0].Target != "a" {
		t.Errorf("got %+v, want the unknown kind preserved", got)
	}
	if related.Kind("verifies").Valid() {
		t.Error("this test is vacuous: \"verifies\" has become a known kind")
	}
}
