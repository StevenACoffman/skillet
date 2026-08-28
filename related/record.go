package related

import "sort"

// The encoding a manifest records edges in, and its inverse.
//
// **It lives here rather than in `manifest` because `manifest` cannot import this
// package**: it is stdlib-only, which is the stated reason `manifest.Skill.Edges` is a
// `map[string][]string` rather than a `[]related.Edge` in the first place. So `manifest`
// owns how edges are stored and this package owns what an edge is, and the translation
// between them belongs on the side that knows the second.
//
// **The round trip is not identity, by design.** Kind and target survive; the rationale
// does not, because a manifest is kept for hashes and prose would bloat it. A test
// asserting that a node survives a manifest unchanged is asserting something the storage
// decision rejected.

// EdgeMap renders edges the way manifest.Skill.Edges records them: each kind to the slugs
// it points at, sorted, with the rationale dropped.
//
// Two producers write that field -- exegesis's `verify` and skillsaw's inventory -- and a
// field encoded two ways is a field that can disagree with itself across the tools that
// share it, which is the drift this package exists to prevent.
//
// Requires: nothing.
// Ensures:  pure. nil for a skill declaring none, so the omitempty field stays omitted
// rather than being written as an empty object; a kind and target pair appearing twice is
// recorded once; does not mutate edges.
func EdgeMap(edges []Edge) map[string][]string {
	if len(edges) == 0 {
		return nil
	}
	out := make(map[string][]string, len(edges))
	seen := make(map[edgeKey]bool, len(edges))
	for _, e := range edges {
		k := edgeKey{kind: e.Kind, target: e.Target}
		if seen[k] {
			continue
		}
		seen[k] = true
		out[string(e.Kind)] = append(out[string(e.Kind)], e.Target)
	}
	for kind := range out {
		sort.Strings(out[kind])
	}
	return out
}

// EdgesFrom reads back what EdgeMap wrote.
//
// The kinds are visited in sorted order so a caller comparing two reconstituted graphs
// gets the same answer on every run: Go randomises map iteration, and an ordering that
// varies for no reason in the data turns a stable comparison into a flaky one.
//
// **An unrecognised kind is preserved, not dropped.** ParseSection cannot produce one, so
// a manifest written by a producer in this family holds only known kinds -- but a
// hand-written or newer-tool manifest may, and silently discarding it would make the
// recorded graph look smaller than the document says it is. A consumer that ranks kinds
// already has to decide what an unknown one is worth; that decision is not this
// function's to pre-empt.
//
// Requires: m came from a manifest, whether or not this package wrote it.
// Ensures:  pure. One Edge per recorded kind and target, kinds ascending and targets in
// recorded order; Rationale is always empty, because the manifest never carried one; does
// not mutate m.
func EdgesFrom(m map[string][]string) []Edge {
	if len(m) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(m))
	for kind := range m {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	var out []Edge
	for _, kind := range kinds {
		for _, target := range m[kind] {
			out = append(out, Edge{Kind: Kind(kind), Target: target})
		}
	}
	return out
}
