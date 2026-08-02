// Package testprompts is skillet's reader and gate for the test-prompts.json
// contract shared by exegesis and skillsaw. It accepts the canonical
// {"tests":[...]} shape, a bare top-level array, and the legacy
// {"test_cases":[...]} shape with "expected_behavior", normalizing all three
// into one form. A case carries an activation Type and optional judge Checks;
// ChecksFor bridges to judge, and Validate gates the composition. Parsing and
// validation are pure; only Load and Write touch the filesystem.
package testprompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
}

// Counts tallies cases by type.
type Counts struct {
	Trigger int
	Decoy   int
	Edge    int
}

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
		return &File{Tests: normalize(arr)}, nil
	}
	var obj struct {
		Skill     string    `json:"skill"`
		Tests     []rawCase `json:"tests"`
		TestCases []rawCase `json:"test_cases"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("object: %w", err)
	}
	cases := obj.Tests
	if len(cases) == 0 {
		cases = obj.TestCases
	}
	return &File{Skill: obj.Skill, Tests: normalize(cases)}, nil
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

// Tally returns the per-type case counts.
func (f *File) Tally() Counts {
	var c Counts
	for _, tc := range f.Tests {
		switch tc.Type {
		case TypeShouldTrigger:
			c.Trigger++
		case TypeShouldNotTrigger:
			c.Decoy++
		case TypeEdgeCase:
			c.Edge++
		}
	}
	return c
}

// Validate returns one problem string per composition or per-case defect; an
// empty slice means the set passes the gate. The result is empty iff every case
// is well-formed and the counts meet MinTrigger/MinDecoy/MinEdge.
func (f *File) Validate() []string {
	var problems []string
	seen := map[int]bool{}
	for _, tc := range f.Tests {
		switch tc.Type {
		case TypeShouldTrigger, TypeShouldNotTrigger, TypeEdgeCase:
		default:
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
	c := f.Tally()
	if c.Trigger < MinTrigger {
		problems = append(
			problems,
			fmt.Sprintf("need >=%d should_trigger, have %d", MinTrigger, c.Trigger),
		)
	}
	if c.Decoy < MinDecoy {
		problems = append(
			problems,
			fmt.Sprintf("need >=%d should_not_trigger, have %d", MinDecoy, c.Decoy),
		)
	}
	if c.Edge < MinEdge {
		problems = append(problems, fmt.Sprintf("need >=%d edge_case, have %d", MinEdge, c.Edge))
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

func normalize(raw []rawCase) []Case {
	cases := make([]Case, 0, len(raw))
	for i, r := range raw {
		expected := r.Expected
		if expected == "" {
			expected = r.ExpectedBehavior
		}
		cases = append(cases, Case{
			ID:       caseID(r.ID, i),
			Type:     r.Type,
			Prompt:   r.Prompt,
			Expected: expected,
			Checks:   r.Checks,
		})
	}
	return cases
}

// caseID reads a numeric id, falling back to position+1 for string ids like
// "should-trigger-01" so every case has a stable positive integer id.
func caseID(raw json.RawMessage, index int) int {
	if len(raw) > 0 {
		var n int
		if err := json.Unmarshal(raw, &n); err == nil {
			return n
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if n, err := strconv.Atoi(s); err == nil {
				return n
			}
		}
	}
	return index + 1
}
