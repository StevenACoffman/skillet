package quotecheck

// Status values.
//
// Unchecked is the zero value, following the same rule as finding.Action's absent
// classification and timeseries.Verdict.Compared: nobody looked is not the claim that this
// is fine, and a value that defaults to a verdict would launder one into the other.
//
// They are declared above the type they belong to because the linter orders declarations by
// kind at file scope. That is the same pressure that split this file.
const (
	Unchecked Status = iota // no source was searched for this passage
	Found                   // located in at least one source
	Missing                 // searched for and located in none
)

// Status is whether a passage was located.
//
// Three values rather than two, because "checked and not found" and "not checked" call for
// opposite responses and a bool cannot hold both. The first is the fabrication guard firing;
// the second says nothing about the quotation at all.
//
// Unchecked is the zero value, following the same rule as finding.Action's absent
// classification and timeseries.Verdict.Compared: nobody looked is not the claim that this
// is fine, and a value that defaults to a verdict would launder one into the other.
type Status int

// String renders a status for a report.
func (s Status) String() string {
	switch s {
	case Found:
		return "found"
	case Missing:
		return "missing"
	case Unchecked:
		return "unchecked"
	default:
		return "unknown"
	}
}
