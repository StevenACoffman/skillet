package judge_test

import (
	"testing"

	"github.com/StevenACoffman/skillet/judge"
)

func TestScoreRequiresChecks(t *testing.T) {
	t.Parallel()
	if _, err := judge.Score("x", nil); err == nil {
		t.Fatal("want an error when no checks are provided")
	}
}

func TestScoreHardSoft(t *testing.T) {
	t.Parallel()
	res, err := judge.Score("hello world", []judge.Check{
		{Op: judge.OpContains, Arg: "hello"},         // pass
		{Op: judge.OpContains, Arg: "absent-phrase"}, // fail
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Hard != 0 {
		t.Errorf("Hard = %v, want 0 (one check failed)", res.Hard)
	}
	if res.Soft != 0.5 {
		t.Errorf("Soft = %v, want 0.5", res.Soft)
	}
	if len(res.Why) != 2 {
		t.Errorf("Why length = %d, want 2", len(res.Why))
	}
}

func TestScoreAllPass(t *testing.T) {
	t.Parallel()
	res, err := judge.Score("# Heading\nabcdef", []judge.Check{
		{Op: judge.OpSectionPresent, Arg: "Heading"},
		{Op: judge.OpMinChars, Arg: "3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Hard != 1.0 || res.Soft != 1.0 {
		t.Errorf("Hard/Soft = %v/%v, want 1/1", res.Hard, res.Soft)
	}
}

func TestChecksOps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		output string
		check  judge.Check
		want   bool
	}{
		{
			"section_present hit",
			"## Setup\nx",
			judge.Check{Op: judge.OpSectionPresent, Arg: "setup"},
			true,
		},
		{
			"section_present miss",
			"no headings",
			judge.Check{Op: judge.OpSectionPresent, Arg: "setup"},
			false,
		},
		{"regex hit", "abc123", judge.Check{Op: judge.OpRegex, Arg: `\d+`}, true},
		{"regex invalid", "abc", judge.Check{Op: judge.OpRegex, Arg: `[`}, false},
		{"max_chars pass", "abc", judge.Check{Op: judge.OpMaxChars, Arg: "5"}, true},
		{"max_chars fail", "abcdef", judge.Check{Op: judge.OpMaxChars, Arg: "3"}, false},
		{"min_chars pass", "abcdef", judge.Check{Op: judge.OpMinChars, Arg: "3"}, true},
		{
			"tool_called",
			"I will call the grep tool",
			judge.Check{Op: judge.OpToolCalled, Arg: "grep"},
			true,
		},
		{"unknown op", "x", judge.Check{Op: judge.Op("nope"), Arg: ""}, false},
		{"boolean yes", "ANSWER: yes", judge.Check{Op: judge.OpBoolean, Arg: "true"}, true},
		{"boolean mismatch", "ANSWER: no", judge.Check{Op: judge.OpBoolean, Arg: "true"}, false},
		{
			"multiple_choice",
			"The ANSWER: B",
			judge.Check{Op: judge.OpMultipleChoice, Arg: "B"},
			true,
		},
		{
			"numeric within oom",
			"ANSWER: 1200",
			judge.Check{Op: judge.OpNumericOOM, Arg: "1000:1"},
			true,
		},
		{
			"numeric out of oom",
			"ANSWER: 5",
			judge.Check{Op: judge.OpNumericOOM, Arg: "1000:1"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := judge.Score(tt.output, []judge.Check{tt.check})
			if err != nil {
				t.Fatal(err)
			}
			if got := res.Hard == 1.0; got != tt.want {
				t.Fatalf(
					"check %+v on %q: pass=%v, want %v (why=%v)",
					tt.check,
					tt.output,
					got,
					tt.want,
					res.Why,
				)
			}
		})
	}
}
