package conflict_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset"
	"github.com/StevenACoffman/skillet/ruleset/conflict"
)

func rule(section, statement string, sev ruleset.Severity, lvl ruleset.Level) ruleset.Rule {
	return ruleset.Rule{Section: section, Statement: statement, Severity: sev, Level: lvl}
}

func TestFindDetectsTheThreePredicates(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		rules []ruleset.Rule
		want  string
	}{
		"severity divergence": {
			[]ruleset.Rule{
				rule("§1.1", "Always close what you open.", "MUST", "CODE"),
				rule("§2.4", "Always close what you open.", "CONSIDER", "CODE"),
			},
			conflict.CategorySeverityDivergence,
		},
		"level divergence": {
			[]ruleset.Rule{
				rule("§1.1", "Always close what you open.", "MUST", "CODE"),
				rule("§2.4", "Always close what you open.", "MUST", "ARCH"),
			},
			conflict.CategoryLevelDivergence,
		},
		"section collision": {
			[]ruleset.Rule{
				rule("§1.1", "Always close what you open.", "MUST", "CODE"),
				rule("§1.1", "Prefer composition.", "MUST", "ARCH"),
			},
			conflict.CategorySectionCollision,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := conflict.Find(ruleset.Ruleset{Rules: tc.rules})
			if len(got) == 0 {
				t.Fatalf("no diagnostic for %s", name)
			}
			found := false
			for _, d := range got {
				if d.Category == tc.want {
					found = true
				}
			}
			if !found {
				t.Errorf("categories = %+v, want one of %q", got, tc.want)
			}
		})
	}
}

// TestEqualityIsFoldedNotByteEqual is the reason this package waited on textnorm: two
// rulesets distilled from differently-typeset copies of one source must be compared on
// their words, or every conflict predicate inherits the disagreement textnorm ended.
func TestEqualityIsFoldedNotByteEqual(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{Rules: []ruleset.Rule{
		rule("§1.1", "Don’t   reuse a closed handle.", "MUST", "CODE"),
		rule("§2.4", "Don't reuse a closed handle.", "CONSIDER", "CODE"),
	}}
	if got := conflict.Find(rs); len(got) == 0 {
		t.Error("a curly apostrophe and a wrapped line hid a severity divergence")
	}
}

// TestAgreeingRulesAreNotConflicts guards the predicate against firing on the ordinary
// case: the same rule stated twice, identically, is duplication and not disagreement.
func TestAgreeingRulesAreNotConflicts(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{Rules: []ruleset.Rule{
		rule("§1.1", "Always close what you open.", "MUST", "CODE"),
		rule("§2.4", "Always close what you open.", "MUST", "CODE"),
	}}
	for _, d := range conflict.Find(rs) {
		if d.Category != conflict.CategorySectionCollision {
			t.Errorf("agreeing rules produced %q: %s", d.Category, d.Message)
		}
	}
}

// TestFindAssignsNoSeverity pins the boundary: these say what was found, not what it costs.
// Whether a severity divergence blocks is canonizer's policy, and a package that decided it
// here would be the ship threshold this family refuses, one level down.
func TestFindAssignsNoSeverity(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{Rules: []ruleset.Rule{
		rule("§1.1", "Always close what you open.", "MUST", "CODE"),
		rule("§2.4", "Always close what you open.", "CONSIDER", "CODE"),
	}}
	got := conflict.Find(rs)
	if len(got) == 0 {
		t.Fatal("expected a diagnostic")
	}
	for _, d := range got {
		if d.Severity != "" {
			t.Errorf("Find assigned severity %q; that is the caller's", d.Severity)
		}
	}
}

func TestFindOnEmptyAndSingleRule(t *testing.T) {
	t.Parallel()
	for name, rs := range map[string]ruleset.Ruleset{
		"empty":       {},
		"single rule": {Rules: []ruleset.Rule{rule("§1.1", "One.", "MUST", "CODE")}},
		"no statement": {Rules: []ruleset.Rule{
			rule("§1.1", "", "MUST", "CODE"), rule("§2.1", "", "CONSIDER", "ARCH"),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := conflict.Find(rs); len(got) != 0 {
				t.Errorf("got %+v, want none", got)
			}
		})
	}
}

// TestNoScoreAnywhere pins the charter in the one place it could erode: this package must
// not grow a total, a ratio, or a count-based verdict that a caller could threshold on.
func TestNoScoreAnywhere(t *testing.T) {
	t.Parallel()
	rs := ruleset.Ruleset{Rules: []ruleset.Rule{
		rule("§1.1", "A.", "MUST", "CODE"), rule("§2.4", "A.", "CONSIDER", "CODE"),
	}}
	for _, d := range conflict.Find(rs) {
		if strings.ContainsAny(d.Message, "%") {
			t.Errorf("message reads like a score: %q", d.Message)
		}
	}
}
