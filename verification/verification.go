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
// **Two importers, and a third consumer's slot now exists.** gnosis (`internal/bundle`) and
// adh (`internal/contextstore`) each hand-wrote this type before it was promoted and both
// import it today. That settles the comparison this paragraph used to carry: it read that if
// the package were still unimported once those two bumped it would share
// `skillet/provenance`'s fate, which was to be carried tested with zero importers and no
// consumer waiting until v0.20.0 deleted it. They bumped, and it is not in that position.
//
// canonizer was the consumer that could not adopt it, because nothing it writes held a list
// of these. `ruleset` version 4 gives it one — the `verified` key in a ruleset's frontmatter
// block — chosen because both existing importers store events in the artifact the events are
// about rather than in a side channel. canonizer still owes the write path, which needs adh's
// actor policy: the actor comes from configured repository identity, never from a flag on the
// invocation, since a caller-supplied actor lets anyone mint a `human:` event.
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
//
// The yaml tags carry the same asymmetry for `ruleset`'s frontmatter block. They are not
// strictly required — the yaml decoder lowercases field names, so `By` and `At` would land
// on `by` and `at` without them — and they are written so the wire format is something an
// editor changes deliberately rather than a consequence of the Go field names.
type Event struct {
	By string `json:"by"           yaml:"by"`
	At string `json:"at,omitempty" yaml:"at,omitempty"`
}
