package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/StevenACoffman/skillet/errs"
	toerr "github.com/StevenACoffman/toerr/errors"
	"github.com/StevenACoffman/toerr/errors/errcode"
)

// TestErrorCodeTranslatesEveryStatus is the package's whole content: eleven statuses
// mapped onto five classifications.
//
// It is a table over the mapping rather than a spot check, because the arms are
// many-to-one and a collapse is silent — `PermissionDenied` reading back as EINTERNAL
// instead of EUNAUTHORIZED would turn "you may not" into "something broke" at every
// consumer at once, and nothing downstream could tell.
func TestErrorCodeTranslatesEveryStatus(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		status errcode.StatusCode
		want   string
	}{
		"invalid argument":              {errcode.StatusInvalidArgument, errs.EINVALID},
		"not found":                     {errcode.StatusNotFound, errs.ENOTFOUND},
		"already exists is a conflict":  {errcode.StatusAlreadyExists, errs.ECONFLICT},
		"failed precondition likewise":  {errcode.StatusFailedPrecondition, errs.ECONFLICT},
		"unauthenticated":               {errcode.StatusUnauthenticated, errs.EUNAUTHORIZED},
		"permission denied likewise":    {errcode.StatusPermissionDenied, errs.EUNAUTHORIZED},
		"internal":                      {errcode.StatusInternal, errs.EINTERNAL},
		"no analogue falls to internal": {errcode.StatusUnimplemented, errs.EINTERNAL},
		"canceled falls to internal":    {errcode.StatusCanceled, errs.EINTERNAL},
		"deadline falls to internal":    {errcode.StatusDeadlineExceeded, errs.EINTERNAL},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := errcode.WithCode(tc.status, "boundary message", nil)
			if got := errs.ErrorCode(err); got != tc.want {
				t.Errorf("ErrorCode(%s) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

// TestErrorCodeAnswersForEveryError. A caller asking for a classification has to get one,
// and the two ends of the range are the ones worth pinning: nil is not a failure, and an
// error nobody classified is an internal fault rather than a caller's mistake.
func TestErrorCodeAnswersForEveryError(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err  error
		want string
	}{
		"nil is not a failure":     {nil, ""},
		"unclassified is internal": {errors.New("boom"), errs.EINTERNAL},
		"a bare wrap is internal":  {toerr.WrapWithMessage(errors.New("boom"), "pkg.Do"), errs.EINTERNAL},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := errs.ErrorCode(tc.err); got != tc.want {
				t.Errorf("ErrorCode = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTheCodeSurvivesWrapping is what makes the vocabulary usable at a distance: a
// classification set at the boundary has to still be readable after the layers above have
// added their operations, or every caller would have to classify at the point of failure.
func TestTheCodeSurvivesWrapping(t *testing.T) {
	t.Parallel()

	leaf := errcode.WithCode(errcode.StatusNotFound, "no such skill", nil)
	for name, err := range map[string]error{
		"toerr wrap":  toerr.WrapWithMessage(leaf, "skill.Load"),
		"two wraps":   toerr.WrapWithMessage(toerr.WrapWithMessage(leaf, "skill.Load"), "cmd.Run"),
		"fmt.Errorf":  fmt.Errorf("ctx: %w", leaf),
		"errors.Join": errors.Join(leaf),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := errs.ErrorCode(err); got != errs.ENOTFOUND {
				t.Errorf("ErrorCode through %s = %q, want not_found", name, got)
			}
			if !errors.Is(err, leaf) {
				t.Errorf("%s broke the Unwrap chain", name)
			}
		})
	}
}

// TestErrorMessageShowsOnlyWhatWasWrittenForAReader. This answers "what may I show a
// user", so an uncoded error's text must not leak: it is written for whoever reads the
// logs and names internal operations, paths and identifiers.
func TestErrorMessageShowsOnlyWhatWasWrittenForAReader(t *testing.T) {
	t.Parallel()

	coded := errcode.WithCode(errcode.StatusInvalidArgument, "the name is required", nil)
	if got := errs.ErrorMessage(coded); got != "the name is required" {
		t.Errorf("ErrorMessage = %q, want the coded message", got)
	}
	if got := errs.ErrorMessage(nil); got != "" {
		t.Errorf("ErrorMessage(nil) = %q, want empty", got)
	}

	leaky := errors.New("open /home/steve/.config/token: permission denied")
	got := errs.ErrorMessage(leaky)
	if got != "an internal error occurred" {
		t.Errorf("ErrorMessage(unclassified) = %q, want the generic sentence", got)
	}
	if got == leaky.Error() {
		t.Error("an unclassified error's own text reached a user-facing message")
	}
}

// TestTheFiveCodesAreDistinct. Two constants sharing a value would collapse two kinds of
// failure into one, and every consumer branching on them would take the same arm for both
// — a defect that looks like correct behaviour until the day the two need to differ.
func TestTheFiveCodesAreDistinct(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, code := range []string{
		errs.ECONFLICT, errs.EINTERNAL, errs.EINVALID, errs.ENOTFOUND, errs.EUNAUTHORIZED,
	} {
		if code == "" {
			t.Error("a code is empty, which ErrorCode uses to mean nil")
		}
		if seen[code] {
			t.Errorf("%q is used by two constants", code)
		}
		seen[code] = true
	}
	if len(seen) != 5 {
		t.Errorf("got %d distinct codes, want five", len(seen))
	}
}
