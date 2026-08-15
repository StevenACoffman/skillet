// Package finding is skillet's shared diagnostic type: one Diagnostic that any
// linter or gate emits, carrying a Severity, an optional Category, an optional
// location Path, and a human Message. It replaces the per-tool finding types
// (exegesis lint's {Severity, Message}, modelith lint's four-field form) with
// one shape. A finding is a static diagnostic — not an adjudication hypothesis.
package finding

import (
	"cmp"
	"slices"
)

// Severity levels. Only SeverityError is blocking.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// The known actions. There is deliberately no constant for "unclassified" -- see Diagnostic.
const (
	// ActionAutomatic means a tool can generate the fix without asking.
	ActionAutomatic Action = "automatic"
	// ActionGuided means a tool can propose the fix but a person confirms it.
	ActionGuided Action = "guided"
	// ActionHuman means closing it needs judgment no tool here has.
	ActionHuman Action = "human"
)

// Severity classifies a Diagnostic.
type Severity string

// Action classifies who can close a finding. It is orthogonal to Severity, and the two
// must not be collapsed: Severity answers whether a finding blocks, Action answers who acts.
// A blocking finding may be safely automatic, and an advisory one may need a person.
type Action string

// Diagnostic is one finding about an artifact.
//
// Action is optional and its zero value means *nobody classified this*, which is not the
// same claim as ActionHuman. That distinction is why there is no ActionUnknown constant to
// set: an unset field cannot be mistaken for a decision, and a named one invites being
// assigned deliberately. Same reason timeseries.Verdict keeps Compared separate from a zero
// baseline -- the absence of a judgement is not a judgement.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Category string   `json:"category,omitempty"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
	Action   Action   `json:"action,omitempty"`
}

// Result accumulates diagnostics from one or more checks.
type Result struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Valid reports whether a is a known action. The zero value is not one: it is the
// unclassified state, so a caller asking "is this a real action" gets false for it.
func (a Action) Valid() bool {
	switch a {
	case ActionAutomatic, ActionGuided, ActionHuman:
		return true
	default:
		return false
	}
}

// Valid reports whether s is a known severity.
func (s Severity) Valid() bool {
	switch s {
	case SeverityError, SeverityWarning:
		return true
	default:
		return false
	}
}

// Blocking reports whether s should fail a gate. Error blocks; warning does not.
func (s Severity) Blocking() bool { return s == SeverityError }

// Add appends a diagnostic to the result.
func (r *Result) Add(d *Diagnostic) { r.Diagnostics = append(r.Diagnostics, *d) }

// HasBlocking reports whether any diagnostic is blocking (Severity error).
func (r *Result) HasBlocking() bool {
	for i := range r.Diagnostics {
		if r.Diagnostics[i].Severity.Blocking() {
			return true
		}
	}
	return false
}

// Sort orders diagnostics deterministically by severity, then category, then
// path, then message — so identical inputs always render in the same order.
// SeverityError sorts before SeverityWarning ("error" < "warning").
func Sort(ds []Diagnostic) {
	slices.SortStableFunc(ds, func(a, b Diagnostic) int {
		return cmp.Or(
			cmp.Compare(a.Severity, b.Severity),
			cmp.Compare(a.Category, b.Category),
			cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Message, b.Message),
		)
	})
}
