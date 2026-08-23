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

func TestUnexaminedRequiresBothFields(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		u    finding.Unexamined
		want bool
	}{
		"both present":      {finding.Unexamined{Aspect: "temporal drift", Reason: "no clock"}, true},
		"no reason":         {finding.Unexamined{Aspect: "temporal drift"}, false},
		"no aspect":         {finding.Unexamined{Reason: "no clock"}, false},
		"zero value":        {finding.Unexamined{}, false},
		"whitespace reason": {finding.Unexamined{Aspect: "a", Reason: "  \t "}, false},
		"whitespace aspect": {finding.Unexamined{Aspect: "\n", Reason: "b"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tc.u.Valid(); got != tc.want {
				t.Errorf("Valid() = %v, want %v for %+v", got, tc.want, tc.u)
			}
		})
	}
}

// TestUnexaminedCannotBlock pins the structural guarantee the field's placement exists
// for: a declared gap is advisory, and no amount of it can make a result blocking. It is
// worth a test rather than a comment because the property comes from Unexamined sitting
// beside Diagnostics rather than inside one, and a later refactor that folded it in --
// giving it a Severity so it could be rendered by the same code path -- would look tidy
// and would silently arm every gate in the family against a critic's own honesty.
func TestUnexaminedCannotBlock(t *testing.T) {
	t.Parallel()
	r := finding.Result{
		Unexamined: []finding.Unexamined{
			{Aspect: "security surface", Reason: "no threat model in the corpus"},
			{Aspect: "temporal degradation", Reason: "single snapshot only"},
		},
	}
	if r.HasBlocking() {
		t.Error("a result carrying only declared gaps must not block")
	}
	r.Add(&finding.Diagnostic{Severity: finding.SeverityWarning, Message: "advisory"})
	if r.HasBlocking() {
		t.Error("gaps plus a warning must not block")
	}
	r.Add(&finding.Diagnostic{Severity: finding.SeverityError, Message: "real defect"})
	if !r.HasBlocking() {
		t.Error("an error diagnostic must still block; the test would pass vacuously otherwise")
	}
}
