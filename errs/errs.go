// Package errs is skillet's error-code vocabulary and the bridge that reads it off a
// toerr-coded error.
//
// It is five constants and a translation. `ErrorCode` and `ErrorMessage` take any error
// and answer in skillet's vocabulary, so a consumer classifies without knowing which
// library produced the error or type-asserting anything.
//
// # The division across the family
//
// **toerr owns wrapping and tracing; errs owns the code vocabulary.** A leaf is
// `errcode.WithCode(status, message, cause)` and a wrapper is
// `toerr.WrapWithMessage(err, op)`; neither lives here. What lives here is the mapping
// from toerr's eleven-value status enum onto the five classifications skillet's consumers
// branch on — which is a translation rather than a forwarding, because the arms are
// many-to-one and the vocabularies are not the same size.
//
// # The Error type is gone, deliberately
//
// This package used to carry an `Error` struct on Ben Johnson's leaf/wrapper convention,
// retained after skillet's own code stopped using it because a consumer composed it
// directly. None does now: every consumer classifies through the two functions below.
// A type kept for an importer that no longer imports it is the case this repository's
// backlog records under `provenance` — carried, tested, and deleted with zero importers —
// and removing it here is that lesson applied before the carrying cost accrues.
package errs

import "github.com/StevenACoffman/toerr/errors/errcode"

// Error codes are machine-readable classifications. Start with these five and add more
// only as a real need appears.
//
// They are strings rather than an enum because consumers put them on the wire: gnosis's
// machine-output envelope carries one as its `reason` token, where an integer whose
// meaning depended on declaration order would be unreadable in a captured log.
const (
	ECONFLICT     = "conflict"     // action cannot be performed in the current state
	EINTERNAL     = "internal"     // an unexpected internal error
	EINVALID      = "invalid"      // input or state failed validation
	ENOTFOUND     = "not_found"    // requested entity does not exist
	EUNAUTHORIZED = "unauthorized" // caller lacks the required authority
)

// ErrorCode returns the machine-readable classification of an error.
//
// Requires: nothing; err may be nil, coded, or from anywhere.
// Ensures: "" for nil, the mapped code for a toerr-coded error anywhere in the chain, and
// EINTERNAL for everything else. Pure.
//
// **EINTERNAL for an unclassified error rather than "" or an error of its own.** A caller
// asking for a classification has to get one, and the honest answer for an error nobody
// classified is that something went wrong internally — which is also the safe direction:
// an unclassified failure reads as a fault rather than as a caller's mistake.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if status := errcode.Status(err); status != errcode.StatusUnknown {
		return codeFromStatus(status)
	}
	return EINTERNAL
}

// ErrorMessage returns the human-readable message an error carries.
//
// Requires: nothing.
// Ensures: "" for nil, the coded error's message when it has one, and a generic sentence
// otherwise. Pure.
//
// **A generic sentence rather than `err.Error()` for an uncoded error.** This answers
// "what may I show a user", and an arbitrary error's text is written for whoever reads
// the logs — it names internal operations, paths and identifiers. A caller wanting the
// full text has it already.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if msg := errcode.Message(err); msg != "" {
		return msg
	}
	return "an internal error occurred"
}

// codeFromStatus maps a toerr errcode.StatusCode onto skillet's string vocabulary.
//
// Requires: nothing.
// Ensures: one of the five codes for every status, EINTERNAL for those with no analogue.
// Pure.
//
// **Two arms are many-to-one, and that is the translation's content.** errcode draws on
// HTTP and gRPC and separates `AlreadyExists` from `FailedPrecondition`, and
// `Unauthenticated` from `PermissionDenied`; skillet's vocabulary does not, because its
// consumers branch on what a caller can do about a failure rather than on which protocol
// named it. Collapsing them here keeps that decision in one place instead of at every
// call site.
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
