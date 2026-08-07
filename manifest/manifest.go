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
}

// Skill is one verified skill's entry.
type Skill struct {
	Slug        string `json:"slug"`
	Dir         string `json:"dir"`
	Hash        string `json:"sha256,omitempty"`       // first-16 sha256 of SKILL.md
	TestPrompts string `json:"test_prompts,omitempty"` // path if present, else ""
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
// Ensures: the four Delta slices partition the union of both manifests' locations; it
//
//	is pure and does not mutate either argument.
func Diff(base, cur Manifest) Delta {
	baseHashes, curHashes := index(base), index(cur)
	var d Delta
	for k, bh := range baseHashes {
		ch, inCur := curHashes[k]
		switch {
		case !inCur:
			d.Removed = append(d.Removed, k)
		case bh == "" || ch == "" || bh != ch:
			d.Changed = append(d.Changed, k)
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
func index(m Manifest) map[string]string {
	out := make(map[string]string, len(m.Skills))
	for _, s := range m.Skills {
		k := location(m.Tree, s.Dir)
		if prev, dup := out[k]; dup && prev != s.Hash {
			out[k] = ""
			continue
		}
		out[k] = s.Hash
	}
	return out
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
