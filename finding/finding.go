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

// Severity classifies a Diagnostic.
type Severity string

// Diagnostic is one finding about an artifact.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Category string   `json:"category,omitempty"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// Result accumulates diagnostics from one or more checks.
type Result struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
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
func (r *Result) Add(d Diagnostic) { r.Diagnostics = append(r.Diagnostics, d) }

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
