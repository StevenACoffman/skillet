package finding_test

import (
	"reflect"
	"testing"

	"github.com/StevenACoffman/skillet/finding"
)

func TestSeverity(t *testing.T) {
	t.Parallel()
	if !finding.SeverityError.Valid() || !finding.SeverityWarning.Valid() {
		t.Error("known severities must be Valid")
	}
	if finding.Severity("warn").Valid() {
		t.Error(`"warn" is not a known severity`)
	}
	if !finding.SeverityError.Blocking() {
		t.Error("error must be blocking")
	}
	if finding.SeverityWarning.Blocking() {
		t.Error("warning must not be blocking")
	}
}

func TestResultAddAndHasBlocking(t *testing.T) {
	t.Parallel()
	var r finding.Result
	if r.HasBlocking() {
		t.Error("empty result must not be blocking")
	}
	r.Add(&finding.Diagnostic{Severity: finding.SeverityWarning, Message: "advisory"})
	if r.HasBlocking() {
		t.Error("warning-only result must not be blocking")
	}
	r.Add(&finding.Diagnostic{Severity: finding.SeverityError, Message: "boom"})
	if !r.HasBlocking() {
		t.Error("result with an error must be blocking")
	}
}

func TestSortDeterministic(t *testing.T) {
	t.Parallel()
	ds := []finding.Diagnostic{
		{Severity: finding.SeverityWarning, Category: "b", Path: "z", Message: "m2"},
		{Severity: finding.SeverityError, Category: "b", Path: "a", Message: "m1"},
		{Severity: finding.SeverityError, Category: "a", Path: "a", Message: "m0"},
	}
	finding.Sort(ds)
	// error before warning; within error, category "a" before "b".
	want := []finding.Diagnostic{
		{Severity: finding.SeverityError, Category: "a", Path: "a", Message: "m0"},
		{Severity: finding.SeverityError, Category: "b", Path: "a", Message: "m1"},
		{Severity: finding.SeverityWarning, Category: "b", Path: "z", Message: "m2"},
	}
	if !reflect.DeepEqual(ds, want) {
		t.Fatalf("Sort = %+v\nwant %+v", ds, want)
	}
}

// TestActionIsOrthogonalToSeverity pins the property the whole field turns on. Severity
// answers whether a finding blocks; Action answers who can close it. Collapsing them is the
// mistake two consumers were about to make separately -- canonizer for its rework budget,
// skillsaw for its one-edit-per-round loop.
func TestActionIsOrthogonalToSeverity(t *testing.T) {
	t.Parallel()
	// A blocking finding a tool can fix unattended, and an advisory one needing a person:
	// both are ordinary, so neither field constrains the other.
	blockingAuto := finding.Diagnostic{
		Severity: finding.SeverityError, Action: finding.ActionAutomatic,
	}
	advisoryHuman := finding.Diagnostic{
		Severity: finding.SeverityWarning, Action: finding.ActionHuman,
	}
	if !blockingAuto.Severity.Blocking() || blockingAuto.Action != finding.ActionAutomatic {
		t.Error("a blocking finding must be allowed to be automatically fixable")
	}
	if advisoryHuman.Severity.Blocking() || advisoryHuman.Action != finding.ActionHuman {
		t.Error("an advisory finding must be allowed to need a human")
	}
}

// TestUnclassifiedActionIsNotAJudgement pins why there is no ActionUnknown constant: the
// zero value means nobody classified this, which is not the claim that a human is required.
func TestUnclassifiedActionIsNotAJudgement(t *testing.T) {
	t.Parallel()
	var d finding.Diagnostic
	if d.Action.Valid() {
		t.Error("the zero value must not be a known action")
	}
	if d.Action == finding.ActionHuman {
		t.Error("unclassified must not equal ActionHuman")
	}
}
