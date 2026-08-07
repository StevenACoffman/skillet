// Package testprompts is skillet's reader and gate for the test-prompts.json
// contract shared by exegesis and skillsaw. It accepts the canonical
// {"tests":[...]} shape, a bare top-level array, and the legacy
// {"test_cases":[...]} shape with "expected_behavior", normalizing all three
// into one form, recording in File.Rewrites how the file differed so a caller that
// writes it back knows it is migrating one. A case carries an activation Type and
// optional judge Checks; ChecksFor bridges to judge.
//
// Validate gates the standard composition; ValidateAgainst takes a Composition, so a
// caller with its own case vocabulary and minimums gates on those without this package
// knowing them. Parsing and validation are pure; only Load and Write touch the
// filesystem.
package testprompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/StevenACoffman/skillet/judge"
)

// Case types (the activation composition).
const (
	TypeShouldTrigger    = "should_trigger"
	TypeShouldNotTrigger = "should_not_trigger"
	TypeEdgeCase         = "edge_case"
)

// Composition minimums: a set without decoys and an edge case only ever looks
// "good", so all three are required.
const (
	MinTrigger = 3
	MinDecoy   = 2
	MinEdge    = 1
)

// Case is one normalized test prompt. Checks reuses judge.Check so a file's
// embedded checks feed judge directly.
type Case struct {
	ID       int           `json:"id"`
	Type     string        `json:"type"`
	Prompt   string        `json:"prompt"`
	Expected string        `json:"expected"`
	Checks   []judge.Check `json:"checks,omitempty"`
}

// File is a parsed, normalized test-prompts.json.
type File struct {
	Skill string `json:"skill,omitempty"`
	Tests []Case `json:"tests"`

	// Rewrites lists how the file as read differed from the canonical form Write
	// emits, one entry per difference, empty when the file was already canonical.
	//
	// Parse accepts three container shapes and several per-case legacy spellings but
	// Write emits only one form, so writing a parsed file back is a silent migration.
	// A caller that writes needs to know: it must either refuse a non-canonical file
	// or say plainly that it is converting one. That is not recoverable from the
	// normalized result, which is why it is recorded here rather than derived.
	//
	// It is excluded from JSON: it describes the file that was read, not the document,
	// and must not appear in what Write emits.
	Rewrites []string `json:"-"`
}

// Counts tallies cases by type.
type Counts struct {
	Trigger int
	Decoy   int
	Edge    int
}

// Composition is a required case mix: each accepted case type mapped to the minimum
// number of cases of that type a set must contain.
//
// One value answers both questions a gate asks — "is this case type legal?" (is it a
// key) and "are there enough of it?" (the value) — which the built-in gate previously
// held as two separate hard-coded lists that could drift apart. A type mapped to 0 is
// accepted but not required.
//
// Callers with their own vocabulary supply their own Composition rather than skillet
// carrying their case types: a merged-skill gate, for instance, adds its
// prefer_merged_over_source category and raises the edge-case floor without this
// package knowing that merging exists.
type Composition map[string]int

// rawCase tolerates every accepted on-disk shape: a numeric or string id, and
// either "expected" or the legacy "expected_behavior".
type rawCase struct {
	ID               json.RawMessage `json:"id"`
	Type             string          `json:"type"`
	Prompt           string          `json:"prompt"`
	Expected         string          `json:"expected"`
	ExpectedBehavior string          `json:"expected_behavior"`
	Checks           []judge.Check   `json:"checks"`
}

// Load reads and parses a test-prompts.json from path.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("testprompts: read %s: %w", path, err)
	}
	f, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("testprompts: parse %s: %w", path, err)
	}
	return f, nil
}

// Parse normalizes any accepted shape into a File. It is pure. Every returned
// Case has a positive ID (position-derived when the source id is non-numeric)
// and Expected populated from either "expected" or "expected_behavior".
func Parse(b []byte) (*File, error) {
	if trimmed := bytes.TrimSpace(b); len(trimmed) > 0 && trimmed[0] == '[' {
		var arr []rawCase
		if err := json.Unmarshal(trimmed, &arr); err != nil {
			return nil, fmt.Errorf("bare array: %w", err)
		}
		cases, rewrites := normalize(arr)
		return &File{
			Tests: cases,
			Rewrites: append(
				[]string{`top-level array; canonical is an object with a "tests" key`},
				rewrites...),
		}, nil
	}
	var obj struct {
		Skill     string    `json:"skill"`
		Tests     []rawCase `json:"tests"`
		TestCases []rawCase `json:"test_cases"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("object: %w", err)
	}
	var rewrites []string
	cases := obj.Tests
	switch {
	case len(cases) == 0 && len(obj.TestCases) > 0:
		cases = obj.TestCases
		rewrites = append(rewrites, `legacy "test_cases" key; canonical is "tests"`)
	case len(cases) > 0 && len(obj.TestCases) > 0:
		// Both keys populated: the reader has always preferred "tests" and dropped the
		// rest, so writing back would delete cases the author can still see on disk.
		rewrites = append(rewrites, fmt.Sprintf(
			`both "tests" and "test_cases" are present; the %d "test_cases" entries are dropped`,
			len(obj.TestCases)))
	}
	normalized, caseRewrites := normalize(cases)
	return &File{
		Skill:    obj.Skill,
		Tests:    normalized,
		Rewrites: append(rewrites, caseRewrites...),
	}, nil
}

// Write marshals f to path in canonical, indented form.
func Write(path string, f *File) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("testprompts: encode: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("testprompts: write %s: %w", path, err)
	}
	return nil
}

// Behavioral returns the cases whose output quality is worth judging:
// should_trigger and edge_case (a decoy has no good output to score).
func (f *File) Behavioral() []Case {
	out := make([]Case, 0, len(f.Tests))
	for _, c := range f.Tests {
		if c.Type == TypeShouldTrigger || c.Type == TypeEdgeCase {
			out = append(out, c)
		}
	}
	return out
}

// Decoys returns the should_not_trigger cases.
func (f *File) Decoys() []Case {
	out := make([]Case, 0, len(f.Tests))
	for _, c := range f.Tests {
		if c.Type == TypeShouldNotTrigger {
			out = append(out, c)
		}
	}
	return out
}

// Find returns the case with the given id.
func (f *File) Find(id int) (Case, bool) {
	for _, c := range f.Tests {
		if c.ID == id {
			return c, true
		}
	}
	return Case{}, false
}

// ChecksFor returns the checks to judge c against: its embedded Checks when
// present, otherwise checks derived from Expected. derived reports which source
// was used; an empty result means neither was available and the caller must
// treat the case as needing hand-written checks rather than silently passing.
func ChecksFor(c *Case) (checks []judge.Check, derived bool) {
	if len(c.Checks) > 0 {
		return c.Checks, false
	}
	return DeriveChecks(c.Expected), true
}

// Standard returns the composition an ordinary skill's test set must meet: a set
// without decoys and an edge case only ever looks "good", so all three are required.
//
// It returns a fresh map on each call rather than exposing a package-level one. A
// shared map is mutable global state — a caller adding its own case type would silently
// change the rule for every other caller in the process.
func Standard() Composition {
	return Composition{
		TypeShouldTrigger:    MinTrigger,
		TypeShouldNotTrigger: MinDecoy,
		TypeEdgeCase:         MinEdge,
	}
}

// CountOf returns how many cases carry the given type.
func (f *File) CountOf(caseType string) int {
	n := 0
	for _, c := range f.Tests {
		if c.Type == caseType {
			n++
		}
	}
	return n
}

// Tally returns the per-type case counts for the three standard types. Sets using a
// wider vocabulary should ask CountOf, which is not limited to them.
func (f *File) Tally() Counts {
	return Counts{
		Trigger: f.CountOf(TypeShouldTrigger),
		Decoy:   f.CountOf(TypeShouldNotTrigger),
		Edge:    f.CountOf(TypeEdgeCase),
	}
}

// Validate gates the set against Standard: one problem string per composition or
// per-case defect, empty when it passes. Use ValidateAgainst to supply a different mix.
func (f *File) Validate() []string {
	return f.ValidateAgainst(Standard())
}

// ValidateAgainst returns one problem string per composition or per-case defect
// measured against want; an empty slice means the set passes that gate.
//
// A case whose type is not a key of want is reported as unknown, so the vocabulary and
// the minimums cannot disagree — they are the same value. Per-case defects (empty
// prompt, empty expected, duplicate id) are gate-independent and always checked.
//
// Ensures: the problems are deterministically ordered; it is pure and reads nothing
//
//	outside f and want.
func (f *File) ValidateAgainst(want Composition) []string {
	var problems []string
	seen := map[int]bool{}
	for _, tc := range f.Tests {
		if _, ok := want[tc.Type]; !ok {
			problems = append(problems, fmt.Sprintf("case %d: unknown type %q", tc.ID, tc.Type))
		}
		if tc.Prompt == "" {
			problems = append(problems, fmt.Sprintf("case %d: empty prompt", tc.ID))
		}
		if tc.Expected == "" {
			problems = append(problems, fmt.Sprintf("case %d: empty expected", tc.ID))
		}
		if seen[tc.ID] {
			problems = append(problems, fmt.Sprintf("duplicate id %d", tc.ID))
		}
		seen[tc.ID] = true
	}
	// Map iteration is randomized; sort so the same set always reports identically.
	for _, caseType := range slices.Sorted(maps.Keys(want)) {
		if n := f.CountOf(caseType); n < want[caseType] {
			problems = append(
				problems,
				fmt.Sprintf("need >=%d %s, have %d", want[caseType], caseType, n),
			)
		}
	}
	return problems
}

// Scaffold returns a minimal passing-shape File for skillName: MinTrigger
// triggers, MinDecoy decoys, and MinEdge edge cases. Each case's Checks are
// seeded from its Expected via DeriveChecks, demonstrating the seam.
func Scaffold(skillName string) *File {
	f := &File{Skill: skillName}
	id := 0
	add := func(typ, prompt, expected string) {
		id++
		f.Tests = append(f.Tests, Case{
			ID:       id,
			Type:     typ,
			Prompt:   prompt,
			Expected: expected,
			Checks:   DeriveChecks(expected),
		})
	}
	for range MinTrigger {
		add(TypeShouldTrigger,
			"TODO: a prompt that SHOULD activate the skill",
			`TODO: describe a good output; e.g. output contains a "Result" section`)
	}
	for range MinDecoy {
		add(TypeShouldNotTrigger,
			"TODO: a plausible decoy prompt that must NOT activate the skill",
			"TODO: describe why the skill should stay silent")
	}
	for range MinEdge {
		add(TypeEdgeCase,
			"TODO: a boundary prompt where activation is genuinely ambiguous",
			"TODO: describe the correct call at the boundary")
	}
	return f
}

// normalize converts raw cases to canonical ones, also reporting each way a case
// differed from the form Write emits. The rewrites are collected here rather than
// recomputed by the caller because the raw spelling is gone once a Case is built.
func normalize(raw []rawCase) (cases []Case, rewrites []string) {
	cases = make([]Case, 0, len(raw))
	for i, r := range raw {
		expected := r.Expected
		if expected == "" && r.ExpectedBehavior != "" {
			expected = r.ExpectedBehavior
			rewrites = append(rewrites, fmt.Sprintf(
				`case %d: legacy "expected_behavior" field; canonical is "expected"`, i+1))
		}
		id, why := caseID(r.ID, i)
		if why != "" {
			rewrites = append(rewrites, fmt.Sprintf("case %d: %s", i+1, why))
		}
		cases = append(cases, Case{
			ID:       id,
			Type:     r.Type,
			Prompt:   r.Prompt,
			Expected: expected,
			Checks:   r.Checks,
		})
	}
	return cases, rewrites
}

// caseID reads a numeric id, falling back to position+1 for string ids like
// "should-trigger-01" so every case has a stable positive integer id. The second
// result names the rewrite that fallback performed, or "" when the on-disk id was
// already a canonical JSON number.
func caseID(raw json.RawMessage, index int) (id int, why string) {
	if len(raw) > 0 {
		var n int
		if err := json.Unmarshal(raw, &n); err == nil {
			return n, ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if n, err := strconv.Atoi(s); err == nil {
				return n, fmt.Sprintf("id %q is a string; canonical is the number %d", s, n)
			}
			return index + 1, fmt.Sprintf("id %q is not a number; renumbered to %d", s, index+1)
		}
	}
	return index + 1, fmt.Sprintf("no id; numbered %d by position", index+1)
}
