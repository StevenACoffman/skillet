// Package errs is skillet's shared error vocabulary: one Error type with
// machine-readable codes, following Ben Johnson's leaf/wrapper convention. A leaf
// error carries a Code and Message; a wrapping error carries an Op and a nested
// Err. The two forms are never mixed on one value: leaves classify, wrappers
// build a single-line logical stack trace. Call ErrorCode/ErrorMessage at a call
// site instead of type-asserting *Error.
//
// errs is also the bridge to toerr: ErrorCode/ErrorMessage read an *Error and a
// toerr-coded error (github.com/StevenACoffman/toerr/errors/errcode) alike, so an
// error classified either way reads back as the same code. The division across
// skillet is deliberate — toerr owns wrapping and tracing; errs owns the leaf
// classification and the code vocabulary. Error stays a plain struct because a
// consumer (adh) composes it directly.
package errs

import (
	"errors"
	"strings"

	"github.com/StevenACoffman/toerr/errors/errcode"
)

// Error codes are machine-readable classifications set on leaf errors. Start
// with these five and add more only as a real need appears.
const (
	ECONFLICT     = "conflict"     // action cannot be performed in the current state
	EINTERNAL     = "internal"     // an unexpected internal error
	EINVALID      = "invalid"      // input or state failed validation
	ENOTFOUND     = "not_found"    // requested entity does not exist
	EUNAUTHORIZED = "unauthorized" // caller lacks the required authority
)

// Error is the one error type for skillet consumers. A leaf error carries Code
// and Message; a wrapping error carries Op and Err. The two forms are never
// mixed on one value.
type Error struct {
	Code    string // machine-readable; set only on leaf errors
	Message string // human-readable; set only on leaf errors
	Op      string // "package.Type.Method"; set only on wrapping errors
	Err     error  // nested cause; set only on wrapping errors
}

// Error renders the logical stack trace: each wrapper contributes its Op, and
// the leaf contributes its Message (or Code when Message is empty).
func (e *Error) Error() string {
	var b strings.Builder
	if e.Op != "" {
		b.WriteString(e.Op)
	}
	switch {
	case e.Err != nil:
		if e.Op != "" {
			b.WriteString(": ")
		}
		b.WriteString(e.Err.Error())
	case e.Message != "":
		if e.Op != "" {
			b.WriteString(": ")
		}
		b.WriteString(e.Message)
	case e.Code != "":
		if e.Op != "" {
			b.WriteString(": ")
		}
		b.WriteString(e.Code)
	}
	return b.String()
}

// Unwrap exposes the nested cause so errors.Is and errors.As traverse the chain.
func (e *Error) Unwrap() error { return e.Err }

// ErrorCode returns the machine-readable code of the first *Error in the chain
// that carries one, EINTERNAL for any other non-nil error, and "" for nil.
// Call this instead of type-asserting *Error at a call site.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		if e.Code != "" {
			return e.Code
		}
		if e.Err != nil {
			return ErrorCode(e.Err)
		}
	}
	if status := errcode.Status(err); status != errcode.StatusUnknown {
		return codeFromStatus(status)
	}
	return EINTERNAL
}

// ErrorMessage returns the human-readable message of the first *Error in the
// chain that carries one, a generic message for any other non-nil error, and ""
// for nil.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		if e.Message != "" {
			return e.Message
		}
		if e.Err != nil {
			return ErrorMessage(e.Err)
		}
	}
	if msg := errcode.Message(err); msg != "" {
		return msg
	}
	return "an internal error occurred"
}

// codeFromStatus maps a toerr errcode.StatusCode onto skillet's string code
// vocabulary, so an error coded via toerr reads back through ErrorCode as the same
// classification an *Error leaf would carry. Codes with no skillet analogue fall to
// EINTERNAL (default-signifies-exhaustive).
func codeFromStatus(code errcode.StatusCode) string {
	switch code {
	case errcode.StatusInvalidArgument:
		return EINVALID
	case errcode.StatusAlreadyExists, errcode.StatusFailedPrecondition:
		return ECONFLICT
	case errcode.StatusNotFound:
		return ENOTFOUND
	case errcode.StatusUnauthenticated, errcode.StatusPermissionDenied:
		return EUNAUTHORIZED
	default:
		return EINTERNAL
	}
}
