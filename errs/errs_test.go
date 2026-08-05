package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/StevenACoffman/skillet/errs"
	"github.com/StevenACoffman/toerr/errors/errcode"
)

func TestErrorString(t *testing.T) {
	t.Parallel()
	leaf := &errs.Error{Code: errs.EINVALID, Message: "bad input"}
	wrapped := &errs.Error{Op: "pkg.Do", Err: leaf}
	nested := &errs.Error{Op: "pkg.Outer", Err: wrapped}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"leaf message", leaf, "bad input"},
		{"leaf code only", &errs.Error{Code: errs.ENOTFOUND}, "not_found"},
		{"wrapper", wrapped, "pkg.Do: bad input"},
		{"nested wrappers", nested, "pkg.Outer: pkg.Do: bad input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorCode(t *testing.T) {
	t.Parallel()
	leaf := &errs.Error{Code: errs.ECONFLICT, Message: "taken"}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"leaf", leaf, errs.ECONFLICT},
		{"wrapped keeps leaf code", &errs.Error{Op: "pkg.Do", Err: leaf}, errs.ECONFLICT},
		{"untyped becomes internal", errors.New("boom"), errs.EINTERNAL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := errs.ErrorCode(tt.err); got != tt.want {
				t.Fatalf("ErrorCode = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	t.Parallel()
	leaf := &errs.Error{Code: errs.EINVALID, Message: "bad input"}
	if got := errs.ErrorMessage(&errs.Error{Op: "pkg.Do", Err: leaf}); got != "bad input" {
		t.Fatalf("ErrorMessage = %q, want %q", got, "bad input")
	}
	if got := errs.ErrorMessage(nil); got != "" {
		t.Fatalf("ErrorMessage(nil) = %q, want empty", got)
	}
	if got := errs.ErrorMessage(errors.New("boom")); got != "an internal error occurred" {
		t.Fatalf("ErrorMessage(untyped) = %q, want generic", got)
	}
}

func TestErrorUnwrapTraverses(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("root cause")
	err := &errs.Error{Op: "pkg.Outer", Err: &errs.Error{Op: "pkg.Inner", Err: sentinel}}
	if !errors.Is(err, sentinel) {
		t.Fatal("errors.Is did not traverse the wrapper chain to the sentinel")
	}
	var target *errs.Error
	if !errors.As(err, &target) {
		t.Fatal("errors.As did not find *errs.Error in the chain")
	}
	// A wrapped fmt.Errorf still resolves its code via the chain.
	wrapped := fmt.Errorf("ctx: %w", &errs.Error{Code: errs.EUNAUTHORIZED})
	if got := errs.ErrorCode(wrapped); got != errs.EUNAUTHORIZED {
		t.Fatalf("ErrorCode through fmt.Errorf = %q, want %q", got, errs.EUNAUTHORIZED)
	}
}

// TestBridgeReadsToerrCodes covers the toerr bridge: an error coded via toerr's
// errcode reads back through ErrorCode as the same string classification an *Error
// leaf would carry, so the two representations share one vocabulary.
func TestBridgeReadsToerrCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status errcode.StatusCode
		want   string
	}{
		{"invalid", errcode.StatusInvalidArgument, errs.EINVALID},
		{"conflict", errcode.StatusFailedPrecondition, errs.ECONFLICT},
		{"already-exists is a conflict", errcode.StatusAlreadyExists, errs.ECONFLICT},
		{"not-found", errcode.StatusNotFound, errs.ENOTFOUND},
		{"permission is unauthorized", errcode.StatusPermissionDenied, errs.EUNAUTHORIZED},
		{"no analogue falls to internal", errcode.StatusUnimplemented, errs.EINTERNAL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := errcode.WithCode(tc.status, "boundary message", nil)
			if got := errs.ErrorCode(err); got != tc.want {
				t.Errorf("ErrorCode(%s) = %q, want %q", tc.status, got, tc.want)
			}
			if got := errs.ErrorMessage(err); got != "boundary message" {
				t.Errorf("ErrorMessage(%s) = %q, want %q", tc.status, got, "boundary message")
			}
		})
	}
}
