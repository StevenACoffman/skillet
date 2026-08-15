// Package conflict finds decidable inconsistencies between the rules of a ruleset.
//
// A ruleset's whole claim is that its rules are internally consistent, and nothing checked
// it: canonizer's verify establishes that each rule is executable and anchored, and two
// rules can both pass and still contradict each other.
//
// Everything here is exact. Each predicate is an equality or a collision over the canonical
// form as it stands today, with no tunable constant anywhere -- which is what makes the
// result safe to put under a gate. Near-duplicate and semantic-similarity detection are
// deliberately absent: "these two rules are 0.87 similar" needs a threshold nobody has
// calibrated, and a threshold under a blocking gate is the defect this family keeps
// refusing. Genuine semantic conflict between two prose rules is judge work, and routes to
// canonizer's existing cold-critic prompt.
//
// It emits finding.Diagnostic and never a score. canonizer's charter is findings-based
// precisely so no threshold can become a ship gate, and a "contradiction score" would be
// that threshold wearing a different name. Severity is the caller's: these say what was
// found, not what it costs -- the same boundary skilllens draws.
//
// Equality is textnorm.Fold-normalized rather than byte equality, so two rulesets distilled
// from differently-typeset copies of one source are compared on their words. Case is
// preserved, per that package's decision.
//
// Everything here is pure.
package conflict

import (
	"sort"
	"strconv"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/ruleset"
	"github.com/StevenACoffman/skillet/textnorm"
)

// Categories reported. They are distinct because the responses differ: a severity or level
// divergence is a disagreement about one rule, while a section collision is a provenance
// failure -- after a merge, anchors and cross-references resolve to whichever copy won.
const (
	CategorySeverityDivergence = "severity-divergence"
	CategoryLevelDivergence    = "level-divergence"
	CategorySectionCollision   = "section-collision"
)

// Find reports every decidable inconsistency among rs's rules.
//
// Requires: nothing; a nil or single-rule ruleset yields no diagnostics.
// Ensures:  every diagnostic names both rules involved by section; the result is ordered
//
//	deterministically for identical input; severity is left unset for the caller
//	to assign; it is pure.
func Find(rs ruleset.Ruleset) []finding.Diagnostic {
	var ds []finding.Diagnostic
	ds = append(ds, divergences(rs.Rules)...)
	ds = append(ds, sectionCollisions(rs.Rules)...)
	finding.Sort(ds)
	return ds
}

// divergences reports rules whose statements are the same words but which disagree about
// severity or level -- the same rule asserted MUST here and CONSIDER there, or governing
// CODE here and ARCH there.
//
// Both are found in one pass because both key on the same grouping: rules that say the same
// thing. Splitting them would fold the text twice and let the two drift apart on what
// "the same statement" means.
func divergences(rules []ruleset.Rule) []finding.Diagnostic {
	byStatement := map[string][]ruleset.Rule{}
	for i := range rules {
		r := &rules[i]
		key := textnorm.Fold(r.Statement)
		if key == "" {
			continue // a rule with no statement says nothing to disagree with
		}
		byStatement[key] = append(byStatement[key], *r)
	}
	var ds []finding.Diagnostic
	for _, key := range sortedKeys(byStatement) {
		group := byStatement[key]
		for i := 1; i < len(group); i++ {
			first, other := &group[0], &group[i]
			if first.Severity != other.Severity {
				ds = append(ds, finding.Diagnostic{
					Category: CategorySeverityDivergence,
					Path:     other.Section,
					Message: "the same statement is " + string(first.Severity) + " in " +
						first.Section + " and " + string(other.Severity) + " here",
				})
			}
			if first.Level != other.Level {
				ds = append(ds, finding.Diagnostic{
					Category: CategoryLevelDivergence,
					Path:     other.Section,
					Message: "the same statement governs " + string(first.Level) + " in " +
						first.Section + " and " + string(other.Level) + " here",
				})
			}
		}
	}
	return ds
}

// sectionCollisions reports a section identity claimed by more than one rule.
//
// Not a semantic contradiction but a provenance one, which is why it is reported even when
// the colliding rules agree: after a merge, a source anchor or a cross-reference naming the
// section resolves to whichever copy won, silently.
func sectionCollisions(rules []ruleset.Rule) []finding.Diagnostic {
	seen := map[string]int{}
	for i := range rules {
		if rules[i].Section != "" {
			seen[rules[i].Section]++
		}
	}
	var ds []finding.Diagnostic
	for _, section := range sortedCounts(seen) {
		ds = append(ds, finding.Diagnostic{
			Category: CategorySectionCollision,
			Path:     section,
			Message: "section " + section + " is claimed by " +
				plural(seen[section]) + "; references to it resolve to only one",
		})
	}
	return ds
}

func sortedKeys(m map[string][]ruleset.Rule) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sortedCounts returns the keys counted more than once, in order.
func sortedCounts(m map[string]int) []string {
	var out []string
	for k, n := range m {
		if n > 1 {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func plural(n int) string {
	return strconv.Itoa(n) + " rules"
}
