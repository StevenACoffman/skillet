package verification_test

import (
	"encoding/json"
	"testing"

	"github.com/StevenACoffman/skillet/verification"
)

// TestEventJSONRoundTripsAndOmitsAnUndeclaredTime pins the only behaviour a two-string
// struct has: its wire format.
//
// There is no test that the fields exist — that is the compiler's job. What is worth
// asserting is the **asymmetry**: `by` is always written and `at` disappears when empty.
// That came from adh's persisted format, so flattening it (dropping `omitempty`, or adding
// it to `by`) would silently change bytes already on disk in a consumer. A tidy-up is
// exactly how that would happen, which is why the asymmetry is a test and not a comment.
func TestEventJSONRoundTripsAndOmitsAnUndeclaredTime(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		event verification.Event
		want  string
	}{
		"both fields": {
			verification.Event{By: "human:steve", At: "2026-09-07"},
			`{"by":"human:steve","at":"2026-09-07"}`,
		},
		"undeclared time":    {verification.Event{By: "check:lint"}, `{"by":"check:lint"}`},
		"actorless is empty": {verification.Event{}, `{"by":""}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if got := string(b); got != tc.want {
				t.Errorf("Marshal = %s, want %s", got, tc.want)
			}
			var back verification.Event
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if back != tc.event {
				t.Errorf("round trip = %+v, want %+v", back, tc.event)
			}
		})
	}
}
