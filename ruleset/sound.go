package ruleset

import (
	"github.com/StevenACoffman/skillet/judge"
)

// Unsound is one rule whose own examples do not exercise its checks.
type Unsound struct {
	// Section identifies the rule, matching Rule.Section.
	Section string
	// Reason says which half of the known-answer pair failed.
	Reason string
}

// Sound reports the rules whose checks do not discriminate between their own examples.
//
// A rule ships the two answers already: Bad is the case it must flag and Good the case it
// must not. Checks make those runnable, and this is the control that runs them -- every
// check must pass on Bad, and they must not all pass on Good. gnosis validates its pattern
// table this way and it caught a pattern whose own positive example did not match on the
// first run, which is the failure a rule set cannot find by reading itself.
//
// **Soundness before completeness.** A rule that fires on ordinary work gets the tool
// switched off, so the negative case is the one that matters most and the one an author will
// not write unprompted.
//
// **Deliberately not called from Parse, which is the plan's own correction.** Running
// predicates over a document while reading it would make a content defect present as an
// unparseable file: a rule whose regex is valid but whose example stopped matching would
// make the whole ruleset unreadable, and unreadable by the very tools that would report it.
// Parse reads bytes; this judges content; a caller runs both. That keeps gnosis's argument
// -- a callable check on the artifact rather than a unit test somebody has to remember to
// run -- without putting evaluation inside a parser.
//
// A rule carrying no checks is **not** reported. It is untested, which is a different claim
// from unsound and belongs to whatever gate decides that rules must carry checks at all.
//
// Requires: nothing; rs may hold no rules.
// Ensures:  the result is in rule order, holds one entry per failing rule, and is empty
//
//	when every rule with checks discriminates; it is pure.
func Sound(rs *Ruleset) []Unsound {
	var out []Unsound
	for i := range rs.Rules {
		r := &rs.Rules[i]
		if len(r.Checks) == 0 {
			continue
		}
		if u, ok := unsound(r); ok {
			out = append(out, u)
		}
	}
	return out
}

// unsound reports how one rule fails the known-answer pair, and whether it does.
func unsound(r *Rule) (Unsound, bool) {
	switch {
	case r.Bad == "" || r.Good == "":
		return Unsound{
			r.Section,
			"carries checks but not both examples, so nothing can be run against them",
		}, true
	case !allPass(r.Bad, r.Checks):
		return Unsound{
			r.Section,
			"the checks do not all pass on the ✗ example, so the rule does not flag " +
				"the case it exists to flag",
		}, true
	case allPass(r.Good, r.Checks):
		return Unsound{
			r.Section,
			"the checks all pass on the ✓ example too, so the rule does not " +
				"discriminate between them",
		}, true
	default:
		return Unsound{}, false
	}
}

// allPass reports whether every check passes on out.
//
// judge.Score errors only on an empty check set, which the caller has already excluded, so
// the error is folded into "did not pass" -- a rule whose checks cannot be scored has not
// demonstrated anything, which is the same answer for this gate's purposes.
func allPass(out string, checks []judge.Check) bool {
	res, err := judge.Score(out, checks)
	return err == nil && res.Hard == 1.0
}
