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
	r.Add(finding.Diagnostic{Severity: finding.SeverityWarning, Message: "advisory"})
	if r.HasBlocking() {
		t.Error("warning-only result must not be blocking")
	}
	r.Add(finding.Diagnostic{Severity: finding.SeverityError, Message: "boom"})
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
