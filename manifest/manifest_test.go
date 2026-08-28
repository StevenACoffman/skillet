package manifest_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/manifest"
)

func TestBuildSortsAndDoesNotMutate(t *testing.T) {
	t.Parallel()
	in := []manifest.Skill{
		{Slug: "zebra", Dir: "d/zebra"},
		{Slug: "alpha", Dir: "d/alpha"},
	}
	m := manifest.Build("exegesis", "tree", in, true)
	if m.Tool != "exegesis" || m.Tree != "tree" || !m.StructureVerified {
		t.Fatalf("header wrong: %+v", m)
	}
	if m.Skills[0].Slug != "alpha" || m.Skills[1].Slug != "zebra" {
		t.Errorf("skills not sorted by slug: %+v", m.Skills)
	}
	if in[0].Slug != "zebra" {
		t.Error("Build must not mutate the input slice order")
	}
}

func TestMarshal(t *testing.T) {
	t.Parallel()
	m := manifest.Build("skillsaw", "tree", []manifest.Skill{
		{Slug: "a", Dir: "d/a", Hash: "abc123"},
		{Slug: "b", Dir: "d/b"}, // no hash, no test-prompts -> omitempty
	}, false)
	b, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		t.Error("Marshal output must end with a newline")
	}
	var round manifest.Manifest
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if round.Tool != "skillsaw" || len(round.Skills) != 2 {
		t.Errorf("round-trip mismatch: %+v", round)
	}
	if bytes.Contains(b, []byte(`"sha256"`)) && bytes.Contains(b, []byte(`"test_prompts"`)) {
		// b's second skill has neither; ensure omitempty dropped them somewhere.
		if bytes.Count(b, []byte(`"sha256"`)) != 1 {
			t.Errorf("omitempty: expected exactly one sha256 field, got:\n%s", b)
		}
	}
}

func TestParseRoundTripsMarshal(t *testing.T) {
	t.Parallel()
	want := manifest.Build("exegesis", "/t", []manifest.Skill{
		{Slug: "b", Dir: "/t/b", Hash: "bbb", TestPrompts: "/t/b/test-prompts.json"},
		{Slug: "a", Dir: "/t/a", Hash: "aaa"},
	}, true)
	b, err := want.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := manifest.Parse(b)
	if err != nil {
		t.Fatalf("Parse rejected our own Marshal output: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, want)
	}
}

func TestParseRejectsAFileThatIsNotAManifest(t *testing.T) {
	t.Parallel()
	// The mistake this guards is aiming --manifest at the wrong JSON file: it
	// unmarshals cleanly into a zero Manifest, and a diff against an empty tree
	// reports every skill as added, which reads like a real answer.
	cases := map[string]string{
		"unrelated json object": `{"name":"something-else","version":2}`,
		"empty object":          `{}`,
		"manifest shape but no tool": `{"tree":"/t","structure_verified":true,` +
			`"skills":[{"slug":"a","dir":"/t/a"}]}`,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := manifest.Parse([]byte(in)); err == nil {
				t.Error("Parse accepted a document with no tool field")
			}
		})
	}
}

func TestParseReportsMalformedJSON(t *testing.T) {
	t.Parallel()
	if _, err := manifest.Parse([]byte(`{"tool":`)); err == nil {
		t.Error("Parse accepted truncated JSON")
	}
}

func TestParseIgnoresUnknownFields(t *testing.T) {
	t.Parallel()
	// A manifest from a newer tool must still read, so the reader is not a
	// version gate on the writer.
	m, err := manifest.Parse([]byte(
		`{"tool":"exegesis","tree":"/t","skills":[],"future_field":{"x":1}}`))
	if err != nil {
		t.Fatalf("Parse rejected a manifest carrying an unknown field: %v", err)
	}
	if m.Tool != "exegesis" || m.Tree != "/t" {
		t.Errorf("known fields lost alongside the unknown one: %+v", m)
	}
}

// tree assembles a scanned-side manifest the way a caller does: a struct literal, since
// Tool and StructureVerified have no meaning before verification has run.
func tree(root string, skills ...manifest.Skill) manifest.Manifest {
	return manifest.Manifest{Tree: root, Skills: skills}
}

func TestDiffPartitionsTheUnion(t *testing.T) {
	t.Parallel()
	base := tree("/t",
		manifest.Skill{Slug: "keep", Dir: "/t/keep", Hash: "h1"},
		manifest.Skill{Slug: "edit", Dir: "/t/edit", Hash: "h2"},
		manifest.Skill{Slug: "gone", Dir: "/t/gone", Hash: "h3"},
	)
	cur := tree("/t",
		manifest.Skill{Slug: "keep", Dir: "/t/keep", Hash: "h1"},
		manifest.Skill{Slug: "edit", Dir: "/t/edit", Hash: "h2-changed"},
		manifest.Skill{Slug: "new", Dir: "/t/new", Hash: "h4"},
	)
	d := manifest.Diff(base, cur)
	for _, tc := range []struct {
		name string
		got  []string
		want []string
	}{
		{"added", d.Added, []string{"new"}},
		{"removed", d.Removed, []string{"gone"}},
		{"changed", d.Changed, []string{"edit"}},
		{"unchanged", d.Unchanged, []string{"keep"}},
		{"stale", d.Stale(), []string{"edit", "new"}},
	} {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	// Totality: every location in either manifest lands in exactly one slice.
	if n := len(d.Added) + len(d.Removed) + len(d.Changed) + len(d.Unchanged); n != 4 {
		t.Errorf("slices hold %d locations, want the 4 in the union: %+v", n, d)
	}
}

func TestDiffMatchesTheSameTreeSpelledDifferently(t *testing.T) {
	t.Parallel()
	// exegesis defaults tree to "." while --tree takes an absolute path, so the same
	// skill is recorded as "foo" by one run and "/t/foo" by another. Matching on the
	// raw Dir would report every skill as both added and removed.
	relative := tree(".", manifest.Skill{Slug: "foo", Dir: "foo", Hash: "h1"})
	absolute := tree("/t", manifest.Skill{Slug: "foo", Dir: "/t/foo", Hash: "h1"})
	d := manifest.Diff(relative, absolute)
	if len(d.Unchanged) != 1 || d.Unchanged[0] != "foo" {
		t.Errorf("same tree spelled two ways did not match: %+v", d)
	}
	if len(d.Stale()) != 0 {
		t.Errorf("nothing changed, but Stale reports %v", d.Stale())
	}
}

func TestDiffKeepsCollidingSlugsApart(t *testing.T) {
	t.Parallel()
	// skill.DiscoverRoots scans several runtime roots, so two distinct skills can
	// share a slug. Matching on slug would collapse them and hide one's edit.
	base := tree(".",
		manifest.Skill{Slug: "foo", Dir: ".claude/skills/foo", Hash: "h1"},
		manifest.Skill{Slug: "foo", Dir: ".cursor/skills/foo", Hash: "h2"},
	)
	cur := tree(".",
		manifest.Skill{Slug: "foo", Dir: ".claude/skills/foo", Hash: "h1"},
		manifest.Skill{Slug: "foo", Dir: ".cursor/skills/foo", Hash: "h2-edited"},
	)
	d := manifest.Diff(base, cur)
	if !reflect.DeepEqual(d.Changed, []string{".cursor/skills/foo"}) {
		t.Errorf("Changed = %v, want only the edited .cursor copy", d.Changed)
	}
	if !reflect.DeepEqual(d.Unchanged, []string{".claude/skills/foo"}) {
		t.Errorf("Unchanged = %v, want the untouched .claude copy", d.Unchanged)
	}
}

func TestDiffTreatsAnUnknownHashAsChanged(t *testing.T) {
	t.Parallel()
	// Hash is omitempty and a writer leaves it empty when the skill would not load.
	// Calling that unchanged would permanently skip a skill that was never hashed.
	cases := map[string]struct{ baseHash, curHash string }{
		"unknown on the base side":  {"", "h1"},
		"unknown on the cur side":   {"h1", ""},
		"unknown on both sides":     {"", ""},
		"known and equal (control)": {"h1", "h1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := manifest.Diff(
				tree("/t", manifest.Skill{Slug: "a", Dir: "/t/a", Hash: tc.baseHash}),
				tree("/t", manifest.Skill{Slug: "a", Dir: "/t/a", Hash: tc.curHash}),
			)
			wantChanged := tc.baseHash == "" || tc.curHash == ""
			if got := len(d.Changed) == 1; got != wantChanged {
				t.Errorf("changed=%t, want %t (base %q, cur %q): %+v",
					got, wantChanged, tc.baseHash, tc.curHash, d)
			}
		})
	}
}

func TestDiffDoesNotLetADuplicateLocationHideAChange(t *testing.T) {
	t.Parallel()
	// Two entries disagreeing about one location leave its hash unknowable. Last-wins
	// could pick the one that matches base and report no change at all.
	base := tree("/t", manifest.Skill{Slug: "a", Dir: "/t/a", Hash: "h1"})
	cur := tree("/t",
		manifest.Skill{Slug: "a", Dir: "/t/a", Hash: "h2"},
		manifest.Skill{Slug: "a", Dir: "/t/a", Hash: "h1"},
	)
	if d := manifest.Diff(base, cur); !reflect.DeepEqual(d.Changed, []string{"a"}) {
		t.Errorf("a contradicted location must count as changed, got %+v", d)
	}
}

func TestDiffAgainstAnEmptyBaseReportsEverythingAdded(t *testing.T) {
	t.Parallel()
	cur := tree("/t",
		manifest.Skill{Slug: "b", Dir: "/t/b", Hash: "h2"},
		manifest.Skill{Slug: "a", Dir: "/t/a", Hash: "h1"},
	)
	d := manifest.Diff(manifest.Manifest{}, cur)
	if !reflect.DeepEqual(d.Added, []string{"a", "b"}) {
		t.Errorf("Added = %v, want both, sorted", d.Added)
	}
	if !reflect.DeepEqual(d.Stale(), []string{"a", "b"}) {
		t.Errorf("Stale = %v, want both", d.Stale())
	}
}

func TestDiffOfAManifestWithItselfIsAllUnchanged(t *testing.T) {
	t.Parallel()
	m := manifest.Build("exegesis", "/t", []manifest.Skill{
		{Slug: "a", Dir: "/t/a", Hash: "h1"},
		{Slug: "b", Dir: "/t/b", Hash: "h2"},
	}, true)
	d := manifest.Diff(m, m)
	if len(d.Stale()) != 0 || len(d.Removed) != 0 || len(d.Unchanged) != 2 {
		t.Errorf("a manifest must not differ from itself: %+v", d)
	}
}

// skillAt builds one manifest entry.
func skillAt(slug, hash, prompts, promptsHash string) manifest.Skill {
	return manifest.Skill{
		Slug: slug, Dir: slug, Hash: hash,
		TestPrompts: prompts, TestPromptsHash: promptsHash,
	}
}

// TestDiffReportsWhichFileMoved. A manifest used to record that a skill had test
// prompts and nothing about what they said, so a SKILL.md could be rewritten while its
// behavioural assertions still described the previous version and every gate passed.
func TestDiffReportsWhichFileMoved(t *testing.T) {
	t.Parallel()
	const prompts = "test-prompts.json"
	cases := map[string]struct {
		base, cur   manifest.Skill
		wantChanged bool
		want        manifest.Axes
	}{
		"only the prose changed": {
			skillAt("a", "h1", prompts, "p1"), skillAt("a", "h2", prompts, "p1"),
			true,
			manifest.Axes{Skill: true},
		},
		"only the prompts changed": {
			skillAt("a", "h1", prompts, "p1"), skillAt("a", "h1", prompts, "p2"),
			true,
			manifest.Axes{TestPrompts: true},
		},
		"both changed": {
			skillAt("a", "h1", prompts, "p1"), skillAt("a", "h2", prompts, "p2"),
			true,
			manifest.Axes{Skill: true, TestPrompts: true},
		},
		"prompts appeared": {
			skillAt("a", "h1", "", ""), skillAt("a", "h1", prompts, "p1"),
			true,
			manifest.Axes{TestPrompts: true},
		},
		"prompts disappeared": {
			skillAt("a", "h1", prompts, "p1"), skillAt("a", "h1", "", ""),
			true,
			manifest.Axes{TestPrompts: true},
		},
		// The case that decides the zero value. Copying Diff's empty-means-unknown
		// rule for Hash would report every skill without prompts as changed forever.
		"no prompts on either side": {
			skillAt("a", "h1", "", ""), skillAt("a", "h1", "", ""),
			false,
			manifest.Axes{},
		},
		"nothing changed at all": {
			skillAt("a", "h1", prompts, "p1"), skillAt("a", "h1", prompts, "p1"),
			false,
			manifest.Axes{},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := manifest.Diff(
				manifest.Build("t", "tree", []manifest.Skill{tc.base}, true),
				manifest.Build("t", "tree", []manifest.Skill{tc.cur}, true),
			)
			if got := len(d.Changed) == 1; got != tc.wantChanged {
				t.Fatalf("changed = %v, want %v (delta %+v)", got, tc.wantChanged, d)
			}
			if !tc.wantChanged {
				if len(d.ChangedAxes) != 0 {
					t.Errorf("an unchanged location has axes: %+v", d.ChangedAxes)
				}
				return
			}
			if got := d.ChangedAxes[d.Changed[0]]; got != tc.want {
				t.Errorf("axes = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestChangedAxesIsKeyedByExactlyChanged is the totality promise, checked rather than
// commented. A fifth slice would have broken it; a map keyed by a subset cannot, and
// this is what says so.
func TestChangedAxesIsKeyedByExactlyChanged(t *testing.T) {
	t.Parallel()
	base := manifest.Build("t", "tree", []manifest.Skill{
		skillAt("same", "h1", "p", "p1"),
		skillAt("prose", "h1", "p", "p1"),
		skillAt("gone", "h1", "p", "p1"),
	}, true)
	cur := manifest.Build("t", "tree", []manifest.Skill{
		skillAt("same", "h1", "p", "p1"),
		skillAt("prose", "h2", "p", "p1"),
		skillAt("new", "h1", "p", "p1"),
	}, true)

	d := manifest.Diff(base, cur)
	if len(d.ChangedAxes) != len(d.Changed) {
		t.Fatalf("%d axes for %d changed locations", len(d.ChangedAxes), len(d.Changed))
	}
	for _, loc := range d.Changed {
		if _, ok := d.ChangedAxes[loc]; !ok {
			t.Errorf("%q is Changed and has no axes", loc)
		}
	}
	// And the four slices still partition the union.
	total := len(d.Added) + len(d.Removed) + len(d.Changed) + len(d.Unchanged)
	if total != 4 {
		t.Errorf("the four slices hold %d locations, want the 4 in the union", total)
	}
}

// TestEdgesAreRecordedNotDiffed is the double-count guard. A skill's edges live in its
// SKILL.md, so any edit to them already moves Hash and shows up on Axes.Skill; feeding them
// to axes as well would report one change on two axes and make every graph edit look like
// two. The field exists to reconstruct a baseline graph, which is a different question from
// staleness.
//
// The two manifests here differ only in Edges, which is a state a real tree cannot reach —
// and that is the point: it isolates the axis under test from the hash that would otherwise
// mask it.
func TestEdgesAreRecordedNotDiffed(t *testing.T) {
	t.Parallel()
	withEdges := func(edges map[string][]string) manifest.Manifest {
		s := manifest.Skill{Slug: "a", Dir: "a", Hash: "same", Edges: edges}
		return manifest.Build("exegesis", "/t", []manifest.Skill{s}, true)
	}
	base := withEdges(map[string][]string{"composes-with": {"b"}})
	cur := withEdges(map[string][]string{"contrasts-with": {"b"}})

	d := manifest.Diff(base, cur)
	if len(d.Changed) != 0 {
		t.Errorf("an edge change was reported as a content change: %v", d.Changed)
	}
	if len(d.Unchanged) != 1 {
		t.Errorf("Unchanged = %v, want the one skill whose hash did not move", d.Unchanged)
	}
}

// TestEdgesSurviveTheRoundTrip keeps the field usable by the consumer it exists for: a tool
// reading a published manifest whose tree is gone gets the graph back or gets nothing.
func TestEdgesSurviveTheRoundTrip(t *testing.T) {
	t.Parallel()
	want := map[string][]string{"depends-on": {"b", "c"}, "informs": {"d"}}
	m := manifest.Build("exegesis", "/t", []manifest.Skill{
		{Slug: "a", Dir: "a", Hash: "h", Edges: want},
	}, true)
	b, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	back, err := manifest.Parse(b)
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	got := back.Skills[0].Edges
	if len(got) != len(want) {
		t.Fatalf("Edges = %v, want %v", got, want)
	}
	for kind, targets := range want {
		if len(got[kind]) != len(targets) {
			t.Fatalf("Edges[%q] = %v, want %v", kind, got[kind], targets)
		}
		for i, target := range targets {
			if got[kind][i] != target {
				t.Errorf("Edges[%q][%d] = %q, want %q", kind, i, got[kind][i], target)
			}
		}
	}
}

// TestASkillWithoutEdgesOmitsTheField keeps the inert property for every manifest written
// before this field existed: they must marshal identically, or a consumer diffing manifest
// files reports drift on trees nobody touched.
func TestASkillWithoutEdgesOmitsTheField(t *testing.T) {
	t.Parallel()
	m := manifest.Build("exegesis", "/t", []manifest.Skill{{Slug: "a", Dir: "a", Hash: "h"}}, true)
	b, err := m.Marshal()
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if strings.Contains(string(b), "edges") {
		t.Errorf("a skill with no edges wrote the key anyway:\n%s", b)
	}
}

// TestEdgesRecordedDistinguishesUnreadFromEmpty is the reason the flag is on the manifest
// rather than inferred from the skills.
//
// encoding/json omits a nil map and an empty one alike, so a skill that declares no edges
// and a skill whose edges were never read serialise identically. Without the flag a
// consumer cannot tell "compared against nothing" from "compared and found nothing", and
// would report a confident no-change against a baseline it never had.
func TestEdgesRecordedDistinguishesUnreadFromEmpty(t *testing.T) {
	t.Parallel()
	// Two manifests whose skills are byte-identical on the wire, differing only in
	// whether the producer claims to have read the graph.
	skills := []manifest.Skill{{Slug: "a", Dir: "a", Hash: "h"}}
	unread := manifest.Manifest{Tool: "t", Tree: ".", Skills: skills}
	empty := manifest.Manifest{Tool: "t", Tree: ".", Skills: skills, EdgesRecorded: true}

	for _, tc := range []struct {
		name string
		in   manifest.Manifest
		want bool
	}{
		{"a producer that never read the graph", unread, false},
		{"a producer that read it and found none", empty, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := tc.in.Marshal()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			got, err := manifest.Parse(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got.EdgesRecorded != tc.want {
				t.Errorf("EdgesRecorded = %v, want %v", got.EdgesRecorded, tc.want)
			}
			// The skills themselves are indistinguishable, which is the point: the
			// flag carries what the entries cannot.
			if got.Skills[0].Edges != nil {
				t.Errorf("Edges = %v, want nil in both cases", got.Skills[0].Edges)
			}
		})
	}
}

// TestEdgesRecordedIsFalseOnAManifestPredatingTheField. The case the flag exists for: a
// document written before it, which must read as "not recorded" rather than as "none".
func TestEdgesRecordedIsFalseOnAManifestPredatingTheField(t *testing.T) {
	t.Parallel()
	const old = `{"tool":"exegesis","tree":".","structure_verified":true,` +
		`"skills":[{"slug":"a","dir":"a","sha256":"h"}]}`
	got, err := manifest.Parse([]byte(old))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.EdgesRecorded {
		t.Error("a manifest with no edges_recorded key read as having recorded them")
	}
}
