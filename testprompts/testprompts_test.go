package testprompts_test

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/judge"
	"github.com/StevenACoffman/skillet/testprompts"
)

func TestParseCanonical(t *testing.T) {
	t.Parallel()
	b := []byte(
		`{"skill":"s","tests":[{"id":7,"type":"should_trigger","prompt":"p","expected":"e"}]}`,
	)
	f, err := testprompts.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if f.Skill != "s" || len(f.Tests) != 1 || f.Tests[0].ID != 7 {
		t.Fatalf("Parse canonical = %+v", f)
	}
}

func TestParseBareArray(t *testing.T) {
	t.Parallel()
	f, err := testprompts.Parse([]byte(`[{"type":"should_trigger","prompt":"p","expected":"e"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Tests) != 1 || f.Tests[0].ID != 1 { // id is position-derived
		t.Fatalf("bare array = %+v", f)
	}
}

func TestParseLegacyTestCases(t *testing.T) {
	t.Parallel()
	b := []byte(
		`{"test_cases":[{"id":"str-01","type":"edge_case","prompt":"p","expected_behavior":"legacy"}]}`,
	)
	f, err := testprompts.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Tests) != 1 {
		t.Fatalf("legacy = %+v", f)
	}
	if f.Tests[0].ID != 1 {
		t.Errorf("string id should fall back to position, got %d", f.Tests[0].ID)
	}
	if f.Tests[0].Expected != "legacy" {
		t.Errorf("expected_behavior fallback, got %q", f.Tests[0].Expected)
	}
}

func TestBehavioralDecoysFind(t *testing.T) {
	t.Parallel()
	f := &testprompts.File{Tests: []testprompts.Case{
		{ID: 1, Type: testprompts.TypeShouldTrigger},
		{ID: 2, Type: testprompts.TypeShouldNotTrigger},
		{ID: 3, Type: testprompts.TypeEdgeCase},
	}}
	if got := len(f.Behavioral()); got != 2 {
		t.Errorf("Behavioral = %d, want 2", got)
	}
	if got := len(f.Decoys()); got != 1 {
		t.Errorf("Decoys = %d, want 1", got)
	}
	if _, ok := f.Find(3); !ok {
		t.Error("Find(3) should be found")
	}
	if _, ok := f.Find(99); ok {
		t.Error("Find(99) should not be found")
	}
}

func TestChecksFor(t *testing.T) {
	t.Parallel()
	embedded := testprompts.Case{Checks: []judge.Check{{Op: judge.OpContains, Arg: "x"}}}
	if got, derived := testprompts.ChecksFor(&embedded); derived || len(got) != 1 {
		t.Errorf("embedded: got %v derived=%v, want 1 check, derived=false", got, derived)
	}
	fromExpected := testprompts.Case{Expected: `output contains "hello"`}
	got, derived := testprompts.ChecksFor(&fromExpected)
	if !derived || len(got) != 1 || got[0].Op != judge.OpContains || got[0].Arg != "hello" {
		t.Errorf("derived: got %v derived=%v, want contains(hello), derived=true", got, derived)
	}
}

func TestScaffoldValidates(t *testing.T) {
	t.Parallel()
	f := testprompts.Scaffold("my-skill")
	if problems := f.Validate(); len(problems) != 0 {
		t.Fatalf("Scaffold must pass Validate, got: %v", problems)
	}
	c := f.Tally()
	if c.Trigger < testprompts.MinTrigger || c.Decoy < testprompts.MinDecoy ||
		c.Edge < testprompts.MinEdge {
		t.Errorf("Scaffold counts short: %+v", c)
	}
}

func TestValidateProblems(t *testing.T) {
	t.Parallel()
	f := &testprompts.File{Tests: []testprompts.Case{
		{ID: 1, Type: "bogus", Prompt: "p", Expected: "e"},
	}}
	if problems := f.Validate(); len(problems) == 0 {
		t.Fatal("expected problems for an ill-formed, under-composed set")
	}
}

func TestWriteRoundTrip(t *testing.T) {
	t.Parallel()
	f := testprompts.Scaffold("round")
	path := filepath.Join(t.TempDir(), "test-prompts.json")
	if err := testprompts.Write(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := testprompts.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, f) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, f)
	}
}

func TestDeriveChecks(t *testing.T) {
	t.Parallel()
	got := testprompts.DeriveChecks(
		`must include a "Result" section and it mentions "done"; at most 500 characters`,
	)
	want := []judge.Check{
		{Op: judge.OpSectionPresent, Arg: "Result"},
		{Op: judge.OpContains, Arg: "done"},
		{Op: judge.OpMaxChars, Arg: "500"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DeriveChecks = %+v\nwant %+v", got, want)
	}
}

func TestParseReportsNoRewritesForACanonicalFile(t *testing.T) {
	t.Parallel()
	f, err := testprompts.Parse([]byte(
		`{"skill":"s","tests":[{"id":1,"type":"should_trigger","prompt":"p","expected":"e"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Rewrites) != 0 {
		t.Errorf("a canonical file must report no rewrites, got %q", f.Rewrites)
	}
}

func TestParseReportsEveryRewrite(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in   string
		want string // substring the rewrite must mention
	}{
		"bare array": {
			in:   `[{"id":1,"type":"should_trigger","prompt":"p","expected":"e"}]`,
			want: "top-level array",
		},
		"legacy test_cases key": {
			in:   `{"test_cases":[{"id":1,"type":"should_trigger","prompt":"p","expected":"e"}]}`,
			want: `legacy "test_cases" key`,
		},
		"legacy expected_behavior": {
			in:   `{"tests":[{"id":1,"type":"should_trigger","prompt":"p","expected_behavior":"e"}]}`,
			want: `legacy "expected_behavior"`,
		},
		"non-numeric string id is renumbered": {
			in:   `{"tests":[{"id":"should-trigger-01","type":"should_trigger","prompt":"p","expected":"e"}]}`,
			want: "renumbered to 1",
		},
		"numeric string id keeps its value but changes type": {
			in:   `{"tests":[{"id":"7","type":"should_trigger","prompt":"p","expected":"e"}]}`,
			want: `is a string`,
		},
		"missing id is numbered by position": {
			in:   `{"tests":[{"type":"should_trigger","prompt":"p","expected":"e"}]}`,
			want: "no id",
		},
		// Silent data loss: the reader has always preferred "tests", so a write-back
		// deletes cases the author can still see on disk.
		"both keys present drops one set": {
			in: `{"tests":[{"id":1,"type":"should_trigger","prompt":"p","expected":"e"}],` +
				`"test_cases":[{"id":2,"type":"edge_case","prompt":"q","expected":"f"}]}`,
			want: "are dropped",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, err := testprompts.Parse([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Rewrites) == 0 {
				t.Fatalf("no rewrite reported for a non-canonical file")
			}
			if !strings.Contains(strings.Join(f.Rewrites, "\n"), tc.want) {
				t.Errorf("rewrites %q do not mention %q", f.Rewrites, tc.want)
			}
		})
	}
}

func TestParseReportsARewritePerCase(t *testing.T) {
	t.Parallel()
	// Two legacy cases and one canonical: the report must name which cases, since
	// its purpose is to tell an author what a write-back would change.
	f, err := testprompts.Parse([]byte(`{"tests":[
		{"id":1,"type":"should_trigger","prompt":"p","expected":"e"},
		{"id":2,"type":"edge_case","prompt":"q","expected_behavior":"f"},
		{"type":"should_not_trigger","prompt":"r","expected":"g"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Rewrites) != 2 {
		t.Fatalf("want 2 rewrites (cases 2 and 3), got %d: %q", len(f.Rewrites), f.Rewrites)
	}
	joined := strings.Join(f.Rewrites, "\n")
	if !strings.Contains(joined, "case 2") || !strings.Contains(joined, "case 3") {
		t.Errorf("rewrites must name the offending cases, got %q", f.Rewrites)
	}
}

func TestWriteDoesNotEmitRewrites(t *testing.T) {
	t.Parallel()
	// Rewrites describes the file that was read, not the document. Leaking it into
	// the output would make every written file non-canonical by its own definition.
	f, err := testprompts.Parse([]byte(
		`[{"id":"x","type":"should_trigger","prompt":"p","expected_behavior":"e"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Rewrites) == 0 {
		t.Fatal("expected rewrites from a bare array with legacy fields")
	}
	path := filepath.Join(t.TempDir(), "test-prompts.json")
	if err := testprompts.Write(path, f); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "Rewrites") || strings.Contains(string(b), "rewrites") {
		t.Errorf("Write leaked the Rewrites field:\n%s", b)
	}
	// The written file must itself be canonical: re-reading it reports nothing.
	back, err := testprompts.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Rewrites) != 0 {
		t.Errorf("Write did not produce a canonical file: %q", back.Rewrites)
	}
}

// set builds a File with n cases of each given type, all otherwise well-formed.
func set(mix map[string]int) *testprompts.File {
	f := &testprompts.File{Skill: "s"}
	for _, typ := range slices.Sorted(maps.Keys(mix)) {
		for range mix[typ] {
			f.Tests = append(f.Tests, testprompts.Case{
				ID: len(f.Tests) + 1, Type: typ, Prompt: "p", Expected: "e",
			})
		}
	}
	return f
}

func TestStandardReturnsAFreshMapEachCall(t *testing.T) {
	t.Parallel()
	// A shared package-level map would let one caller's gate mutate everyone else's.
	a := testprompts.Standard()
	a["injected"] = 99
	if _, leaked := testprompts.Standard()["injected"]; leaked {
		t.Error("Standard hands out a shared map: a caller's edit changed the rule")
	}
}

func TestValidateStillGatesTheStandardThree(t *testing.T) {
	t.Parallel()
	// Validate must behave exactly as before for every existing caller.
	full := set(map[string]int{
		testprompts.TypeShouldTrigger:    testprompts.MinTrigger,
		testprompts.TypeShouldNotTrigger: testprompts.MinDecoy,
		testprompts.TypeEdgeCase:         testprompts.MinEdge,
	})
	if problems := full.Validate(); len(problems) != 0 {
		t.Errorf("a standard-conforming set must pass, got %q", problems)
	}
	short := set(map[string]int{testprompts.TypeShouldTrigger: 1})
	problems := strings.Join(short.Validate(), "\n")
	for _, want := range []string{
		"need >=3 should_trigger, have 1",
		"need >=2 should_not_trigger, have 0",
		"need >=1 edge_case, have 0",
	} {
		if !strings.Contains(problems, want) {
			t.Errorf("Validate no longer reports %q; got:\n%s", want, problems)
		}
	}
}

func TestValidateAgainstAcceptsACallerVocabulary(t *testing.T) {
	t.Parallel()
	// The case this whole item exists for: a merged skill's four-category gate, which
	// the standard vocabulary rejects as an unknown type.
	const preferMerged = "prefer_merged_over_source"
	merged := testprompts.Composition{
		testprompts.TypeShouldTrigger:    3,
		testprompts.TypeShouldNotTrigger: 2,
		testprompts.TypeEdgeCase:         2,
		preferMerged:                     2,
	}
	f := set(map[string]int{
		testprompts.TypeShouldTrigger:    3,
		testprompts.TypeShouldNotTrigger: 2,
		testprompts.TypeEdgeCase:         2,
		preferMerged:                     2,
	})
	if problems := f.ValidateAgainst(merged); len(problems) != 0 {
		t.Errorf("a conforming merged set must pass its own gate, got %q", problems)
	}
	// ...and the standard gate must still reject it, since it does not know the type.
	if problems := f.Validate(); len(problems) == 0 {
		t.Error("the standard gate must not silently accept an unknown case type")
	}
}

func TestValidateAgainstReportsAShortfallInTheCallerVocabulary(t *testing.T) {
	t.Parallel()
	const preferMerged = "prefer_merged_over_source"
	want := testprompts.Composition{testprompts.TypeShouldTrigger: 1, preferMerged: 2}
	f := set(map[string]int{testprompts.TypeShouldTrigger: 1, preferMerged: 1})
	problems := strings.Join(f.ValidateAgainst(want), "\n")
	if !strings.Contains(problems, "need >=2 "+preferMerged+", have 1") {
		t.Errorf("shortfall in a caller type not reported; got:\n%s", problems)
	}
}

func TestValidateAgainstTreatsAZeroMinimumAsAcceptedButNotRequired(t *testing.T) {
	t.Parallel()
	want := testprompts.Composition{testprompts.TypeShouldTrigger: 1, "optional_kind": 0}
	f := set(map[string]int{testprompts.TypeShouldTrigger: 1})
	if problems := f.ValidateAgainst(want); len(problems) != 0 {
		t.Errorf("a zero minimum must not be a requirement, got %q", problems)
	}
	withOne := set(map[string]int{testprompts.TypeShouldTrigger: 1, "optional_kind": 1})
	if problems := withOne.ValidateAgainst(want); len(problems) != 0 {
		t.Errorf("a zero-minimum type must still be accepted, got %q", problems)
	}
}

func TestValidateAgainstIsDeterministic(t *testing.T) {
	t.Parallel()
	// Minimums are read from a map, whose iteration order is randomized.
	want := testprompts.Composition{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1, "f": 1}
	f := set(map[string]int{})
	first := f.ValidateAgainst(want)
	for range 50 {
		if got := f.ValidateAgainst(want); !reflect.DeepEqual(got, first) {
			t.Fatalf("problem order varies between calls:\n%v\nvs\n%v", got, first)
		}
	}
}

func TestCountOf(t *testing.T) {
	t.Parallel()
	f := set(map[string]int{testprompts.TypeShouldTrigger: 3, "custom": 2})
	if n := f.CountOf(testprompts.TypeShouldTrigger); n != 3 {
		t.Errorf("CountOf(should_trigger) = %d, want 3", n)
	}
	if n := f.CountOf("custom"); n != 2 {
		t.Errorf("CountOf(custom) = %d, want 2", n)
	}
	if n := f.CountOf("absent"); n != 0 {
		t.Errorf("CountOf(absent) = %d, want 0", n)
	}
}

func TestTallyStillCountsTheStandardThree(t *testing.T) {
	t.Parallel()
	f := set(map[string]int{
		testprompts.TypeShouldTrigger:    3,
		testprompts.TypeShouldNotTrigger: 2,
		testprompts.TypeEdgeCase:         1,
		"custom":                         5, // must not disturb the three fields
	})
	if c := f.Tally(); c.Trigger != 3 || c.Decoy != 2 || c.Edge != 1 {
		t.Errorf("Tally = %+v, want {3 2 1}", c)
	}
}
