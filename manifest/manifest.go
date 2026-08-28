// Package manifest reads and writes the skills-manifest.json a verify pass emits: a
// machine-readable summary of a verified skill tree that downstream tools consult
// instead of rediscovering it.
//
// Build and Marshal produce one; Parse and Diff consume one. Diff answers the question
// a manifest is kept for — which skills have changed since it was written — so a caller
// can reprocess only those. Every function here is pure; the caller does the file I/O.
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
)

// Manifest is the top-level skills-manifest.json document.
type Manifest struct {
	Tool              string  `json:"tool"`               // the emitting tool, e.g. "exegesis"
	Tree              string  `json:"tree"`               // the verified tree path
	StructureVerified bool    `json:"structure_verified"` // true iff every gate passed
	Skills            []Skill `json:"skills"`

	// EdgesRecorded says whether the producer populated Skill.Edges at all.
	//
	// **The question is about the manifest, not about any skill, which is why it lives
	// here.** Per-skill nil cannot answer it: encoding/json omits a nil map and an empty
	// one alike, so a skill that declares no edges and a skill whose edges were never
	// read are the same bytes. A consumer needs to tell "compared against nothing" from
	// "compared and found nothing" -- the timeseries.Verdict.Compared distinction -- and
	// inferring it from "every skill has no edges" would be wrong for the one tree that
	// genuinely has none.
	//
	// False on a manifest written before Edges existed, which is the case it is for. A
	// producer that populates Edges and forgets this flag fails **closed**: the consumer
	// reads the graph as unavailable and declines to report, rather than reporting a
	// silent "no changes" against a baseline it never had.
	EdgesRecorded bool `json:"edges_recorded,omitempty"`
}

// Skill is one verified skill's entry.
type Skill struct {
	Slug        string `json:"slug"`
	Dir         string `json:"dir"`
	Hash        string `json:"sha256,omitempty"`       // first-16 sha256 of SKILL.md
	TestPrompts string `json:"test_prompts,omitempty"` // path if present, else ""

	// TestPromptsHash is the first-16 sha256 of the test-prompts file, or "" when
	// there is none.
	//
	// Without it a manifest records that a skill has test prompts and nothing about
	// what they say, so a SKILL.md can be rewritten while its behavioural assertions
	// still describe the previous version and every gate in the family passes. The
	// only thing comparing versions is Diff, and it had nothing to compare on.
	//
	// Empty means **absent**, not unknown. That distinction is load-bearing here in a
	// way it is not for Hash: Diff treats an empty Hash as unknown-therefore-changed,
	// and copying that rule would report every skill without test prompts as changed
	// on every run.
	TestPromptsHash string `json:"test_prompts_sha256,omitempty"`

	// Edges is the skill's related-skills graph: edge kind to the slugs it points at,
	// sorted. Empty when the skill declares none.
	//
	// It exists for the one question a manifest could not answer about a tree it no longer
	// has: what the graph looked like. Whether a skill was orphaned by a change needs the
	// baseline's edges, and Diff can say a body moved but never what it said. A consumer
	// comparing two checkouts should read both trees instead -- one parser, one version, no
	// drift -- and this is for the case where the baseline is a published artifact and the
	// checkout is gone.
	//
	// map[string][]string rather than a related.Edge slice, deliberately: this package is
	// stdlib-only, and the one other kernel package that gave that up recorded it as a cost.
	// related.Kind is a string type, so a consumer converts without a decoder. The rationale
	// on each edge is dropped -- the graph question needs kind and target, and prose would
	// bloat a file kept for hashes.
	//
	// **Recorded, not diffed.** Edges live in SKILL.md, so any edge change already moves
	// Hash and surfaces as Axes.Skill. Feeding them to axes as well would report one change
	// on two axes and make every graph edit look like two.
	Edges map[string][]string `json:"edges,omitempty"`
}

// Delta is the difference between two manifests, as tree-relative skill locations.
//
// It is total: every location present in either manifest appears in exactly one of the
// four slices, so a caller can report "skipped N unchanged" without a second pass and
// the four lengths sum to the size of the union. Each slice is sorted.
type Delta struct {
	Added     []string // present in cur, absent from base
	Removed   []string // present in base, absent from cur
	Changed   []string // in both, and not known to be identical
	Unchanged []string // in both, with equal known hashes

	// ChangedAxes says *what* changed, for each location in Changed. A location
	// absent from this map is in Changed for a reason the axes do not cover.
	//
	// A map keyed by a subset of Changed, rather than a fifth slice or a richer
	// Changed element. A fifth slice would break the totality promise above — a
	// location whose prompts changed is also a location that changed — and turning
	// Changed into a slice of structs would be a breaking change to a kernel type
	// with four consumers, for an ergonomic gain. The subset relation is asserted by
	// a test rather than left to this comment.
	ChangedAxes map[string]Axes `json:"changed_axes,omitempty"`
}

// Axes reports which of a skill's two files moved.
//
// The zero value says neither, which is the honest reading of a location nobody
// examined — and never appears in a Delta, because Diff records an entry only for a
// location it placed in Changed.
type Axes struct {
	// Skill is true when SKILL.md's hash differs, or is unknown on either side.
	Skill bool `json:"skill"`

	// TestPrompts is true when the test-prompts file appeared, disappeared, or
	// changed content. It is false when both sides have no test prompts at all —
	// absence on both sides is not a change, and reporting it as one would make
	// every skill without prompts permanently interesting.
	TestPrompts bool `json:"test_prompts"`
}

// hashes are one location's two content fingerprints.
//
// prompts distinguishes "no test-prompts file" from "a file whose hash is unknown",
// which skill deliberately does not: an unknown SKILL.md hash means the manifest was
// written without one and the location must read as changed, whereas a missing
// test-prompts file is an ordinary and stable state.
type hashes struct {
	skill      string
	prompts    string
	hasPrompts bool
}

// Build assembles a Manifest, sorting skills by slug so output is deterministic.
// tool names the emitting tool; verified reflects whether every gate passed
// across all skills. It is pure and does not mutate the input slice.
func Build(tool, tree string, skills []Skill, verified bool) Manifest {
	sorted := make([]Skill, len(skills))
	copy(sorted, skills)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slug < sorted[j].Slug })
	return Manifest{
		Tool:              tool,
		Tree:              tree,
		StructureVerified: verified,
		Skills:            sorted,
	}
}

// Parse decodes a skills-manifest.json document.
//
// A document with no "tool" field is rejected. Unmarshalling arbitrary JSON into a
// Manifest otherwise succeeds and yields a zero value, so aiming a --manifest flag at
// the wrong file would read as an empty tree rather than as a mistake — and a diff
// against an empty tree reports every skill as added, which looks like a real answer.
// Every manifest written by a tool in this family sets "tool", so its absence
// identifies the wrong-file case without rejecting any genuine manifest.
//
// Unknown fields are ignored, so a manifest from a newer tool still reads.
//
// Ensures: the result round-trips a Marshal of the same manifest; it is pure.
func Parse(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	if m.Tool == "" {
		return Manifest{}, errors.New(`parse manifest: no "tool" field; not a skills manifest`)
	}
	return m, nil
}

// Diff reports how the skills in cur differ from those recorded in base.
//
// Skills are matched on location, not slug. A slug is not unique: skill.DiscoverRoots
// scans several runtime roots, so .claude/skills/foo and .cursor/skills/foo are two
// distinct skills sharing the slug "foo", and matching on slug would silently collapse
// them into one. The location is taken relative to each manifest's own Tree, so a tree
// recorded as "." matches the same tree recorded by absolute path — otherwise two runs
// that spelled --tree differently would report every skill as both added and removed.
//
// An entry whose hash is unknown on either side is reported as Changed, never as
// Unchanged. Hash is omitempty and a writer leaves it empty when the skill could not be
// loaded, so treating it as unchanged would silently skip a skill that was never
// successfully hashed. The same applies when a location appears twice with disagreeing
// hashes: neither can be trusted, so it counts as changed.
//
// Only Tree and Skills are read. Tool and StructureVerified are ignored, so a caller
// can assemble cur as a plain struct literal from a fresh scan rather than inventing a
// value for a field that has no meaning before verification has run.
//
// Ensures: the four Delta slices partition the union of both manifests' locations,
//
//	and ChangedAxes is keyed by exactly the members of Changed; it is pure and does
//	not mutate either argument.
//
// A skill whose test prompts changed is Changed even when its prose did not, which is
// a behaviour change: before this, a manifest recorded that a test-prompts file
// existed and nothing about its content, so the pair could drift apart with every
// gate passing.
func Diff(base, cur Manifest) Delta {
	baseHashes, curHashes := index(base), index(cur)
	var d Delta
	for k, bh := range baseHashes {
		ch, inCur := curHashes[k]
		a := axes(bh, ch)
		switch {
		case !inCur:
			d.Removed = append(d.Removed, k)
		case a.Skill || a.TestPrompts:
			d.Changed = append(d.Changed, k)
			if d.ChangedAxes == nil {
				d.ChangedAxes = make(map[string]Axes)
			}
			d.ChangedAxes[k] = a
		default:
			d.Unchanged = append(d.Unchanged, k)
		}
	}
	for k := range curHashes {
		if _, inBase := baseHashes[k]; !inBase {
			d.Added = append(d.Added, k)
		}
	}
	// Map iteration is randomized; sort so the output is reproducible.
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	sort.Strings(d.Unchanged)
	return d
}

// Marshal renders the manifest as indented JSON with a trailing newline.
func (m Manifest) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	return append(b, '\n'), nil
}

// Stale returns the locations that need reprocessing: those added since base plus those
// whose content changed, sorted.
//
// This is the question a caller asks when a manifest is used as a skip list, so the
// answer lives here rather than in each caller. Two callers unioning the same two fields
// themselves would be the same design decision written down twice.
//
// The receiver is a pointer only to avoid copying the four slice headers; Stale does not
// mutate the Delta, and Delta has no other methods.
func (d *Delta) Stale() []string {
	out := make([]string, 0, len(d.Added)+len(d.Changed))
	out = append(out, d.Added...)
	out = append(out, d.Changed...)
	sort.Strings(out)
	return out
}

// index maps each skill's tree-relative location to its recorded hash.
//
// A location recorded twice with disagreeing hashes collapses to the unknown hash "",
// which Diff already classifies as changed. Last-wins would let a duplicate hide a real
// change; falling back to "unknown" reuses the rule that already governs a missing hash
// rather than inventing a second one.
func index(m Manifest) map[string]hashes {
	out := make(map[string]hashes, len(m.Skills))
	for _, s := range m.Skills {
		k := location(m.Tree, s.Dir)
		h := hashes{
			skill:      s.Hash,
			prompts:    s.TestPromptsHash,
			hasPrompts: s.TestPrompts != "" || s.TestPromptsHash != "",
		}
		if prev, dup := out[k]; dup && prev.skill != h.skill {
			// Two skills at one location disagreeing about the hash: the location is
			// not describable, so it reads as unknown and therefore changed.
			out[k] = hashes{}
			continue
		}
		out[k] = h
	}
	return out
}

// axes reports what moved between two readings of one location.
//
// Requires: both sides describe the same location.
// Ensures: pure. Skill is true when the SKILL.md hash differs or is unknown on
// either side, matching Diff's existing rule. TestPrompts is true when the file
// appeared, disappeared, or changed content, and false when neither side has one.
func axes(base, cur hashes) Axes {
	return Axes{
		Skill: base.skill == "" || cur.skill == "" || base.skill != cur.skill,
		TestPrompts: base.hasPrompts != cur.hasPrompts ||
			(base.hasPrompts && base.prompts != cur.prompts),
	}
}

// location identifies a skill by its directory relative to the tree it was recorded
// under, so the same skill matches across manifests that spelled the tree differently.
// A dir that cannot be made relative to tree is used as-is, which still matches itself.
func location(tree, dir string) string {
	if rel, err := filepath.Rel(tree, dir); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(dir)
}
