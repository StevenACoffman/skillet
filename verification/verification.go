// Package verification holds the record of one verification event: who confirmed
// something, and when.
//
// It is the shared half of OKF §5.2's `verified` field, which is a *list* of independent
// events rather than a single tier. A stored tier cannot say which human, or on what date,
// and cannot represent "reviewed by a person **and** re-confirmed by a nightly process" —
// the state a compounding corpus reaches most often, and the reason §5.2 keeps the events.
//
// **The fold is deliberately absent.** Deriving a trust tier from a list of these is not
// here and should not be added. Two consumers derive one today and they disagree by design:
// gnosis closes its actor vocabulary to three kinds because SPEC §10.6.4 counts distinct
// humans, while adh answers human-or-machine for a context unit. A shared fold would have
// to widen one or break the other, so the record is shared and the judgment is not. That
// was decided 2026-09-07; see skillet's TODO.md, "OKF's trust fields".
//
// **It has no importers yet, and that is a sequencing fact rather than a want of one.**
// gnosis (`internal/bundle`) and adh (`internal/contextstore`) each hand-wrote this type
// before it was promoted, and both pin a released skillet, so neither can adopt it until
// this ships. Both repositories' TODO.md record the swap as owed. That distinguishes it from
// `skillet/provenance`, which was carried tested with zero importers and no consumer waiting
// until v0.20.0 deleted it — if this package is still unimported after those two bump, it
// has the same problem and the same remedy.
package verification

// Event is one verification: an actor and a declared time.
//
// **By is the raw actor string, not a parsed actor type.** gnosis reads the raw form in its
// fold deliberately — OKF §14.1.1 makes the raw and parsed forms two different populations —
// so parsing it here would decide for both consumers, which is the fold again.
//
// **At is a string, not a time.Time, and that is a correctness choice.** The value is
// projected verbatim by its consumers and compared by neither, and parsing it here would
// make a malformed date drop a verification silently rather than carry it forward as
// written.
//
// The json tags are asymmetric on purpose: `by` is always present because an event with no
// actor records nothing, while `at` is omitted when empty because a verification whose date
// was never declared is still a verification. They come from adh's wire format, which
// already persists this shape.
type Event struct {
	By string `json:"by"`
	At string `json:"at,omitempty"`
}
