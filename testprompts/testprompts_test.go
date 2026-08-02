package testprompts_test

import (
	"path/filepath"
	"reflect"
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
