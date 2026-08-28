package related

import "strings"

// rationales is the rewrite's memory: for each relationship already written, the words
// the surviving bullet carries for it.
//
// The reader keeps a set (see newEdges) because "have I seen this relationship" is all
// it needs. A rewrite needs more — it decides whether a later bullet can be deleted, and
// that turns on whether the words it carries are already on the page.
type rationales map[edgeKey]string

// Normalize rewrites md's related-skills content into the canonical form: one
// `## Related Skills` heading, and every bullet that names a skill replaced by one
// canonical Bullet per target it named. It reports whether md changed.
//
// A file holding several related-skills sections is merged into the first, which is
// why this can move lines at all: 13 skills in the real books carry a second section,
// and leaving them apart means one skill declares its relationships in two places that
// disagree. A later section is merged only when every line of it is a bullet, a blank,
// or a thematic break — the shapes that can be moved or dropped without losing
// anything. A later section holding prose is left exactly where it is and normalized
// in place, because deleting a paragraph to tidy a heading is not a trade worth making.
//
// Apart from that merge it substitutes only the lines it understands. A bullet whose
// "target" is prose, an introductory sentence, a blank line, a thematic break, fenced
// code, and everything outside the sections are all left exactly as they were. That is
// deliberate: a rewrite that only ever replaces matched lines cannot lose unmatched
// content, so there is no separate preservation rule that could be got wrong.
// Regenerating a section from its parsed edges would have deleted the five prose
// bullets in the real books.
//
// Edges are deduplicated by (Kind, Target) across every section, first occurrence
// winning, matching ParseSection — so a relationship stated in both sections is
// written once rather than twice. Merged bullets keep document order; the canonical
// order is the reader's job (see ParseSection), and imposing it here would reorder
// bullets relative to the prose lines this refuses to move.
//
// **Deduplication never costs words.** A bullet is deleted only when what it says is
// already on the bullet that survives: the same rationale, or none of its own. A bullet
// that restates a relationship in *different* words is kept exactly as written, the way
// one the reader cannot parse is, because both carry prose that exists nowhere else and
// this is a rewrite of form, not an editor. Which of two explanations is better is a
// judgement, and the file is where a person makes it.
//
// Ensures: Normalize(Normalize(md)) == Normalize(md); ParseSection(out) == the same
//
//	edges as ParseSection(md); every line outside the sections is byte-identical;
//	no rationale present in md is absent from out; it is pure.
func Normalize(md string) (string, bool) {
	lines := strings.Split(md, "\n")
	secs := findSections(lines)
	if len(secs) == 0 {
		return md, false
	}
	// The first section is rewritten first so that it, not a later one, decides the
	// rationale a relationship stated twice keeps.
	written := make(rationales)
	first := normalizeBullets(lines, secs[0], written)
	var hoisted []string
	merged := make(map[int]bool, len(secs))
	for i := 1; i < len(secs); i++ {
		if !mergeable(lines, secs[i]) {
			continue
		}
		merged[i] = true
		for _, b := range sectionBullets(lines, secs[i].head, secs[i].end) {
			hoisted = append(hoisted, canonicalize(b, lines, written)...)
		}
	}
	result := strings.Join(
		assemble(lines, secs, withBullets(first, hoisted), merged, written), "\n")
	return result, result != md
}

// assemble rebuilds the file: every line outside a section copied through, each
// surviving section under the canonical heading, and each merged section's span
// dropped entirely — its bullets are already in first.
func assemble(
	lines []string,
	secs []section,
	first []string,
	merged map[int]bool,
	written rationales,
) []string {
	out := make([]string, 0, len(lines))
	at := 0
	for i, sec := range secs {
		out = append(out, lines[at:sec.head]...)
		at = sec.end
		if merged[i] {
			continue
		}
		out = append(out, sectionHeading)
		if i == 0 {
			out = append(out, first...)
			continue
		}
		out = append(out, normalizeBullets(lines, sec, written)...)
	}
	return append(out, lines[at:]...)
}

// normalizeBullets returns the section body with each parseable bullet replaced by
// its canonical form. Lines that are not a parseable bullet are copied through
// untouched.
func normalizeBullets(lines []string, sec section, written rationales) []string {
	var body []string
	at := sec.head + 1
	for _, b := range sectionBullets(lines, sec.head, sec.end) {
		body = append(body, lines[at:b.start]...) // untouched lines before this bullet
		body = append(body, canonicalize(b, lines, written)...)
		at = b.end
	}
	return append(body, lines[at:sec.end]...)
}

// canonicalize returns the replacement lines for one logical bullet: a canonical bullet
// per target it states for the first time, or the bullet's own lines when rewriting it
// would delete words.
//
// Two cases keep a bullet exactly as written, and they are the same case: it carries
// prose that exists nowhere else. One is a bullet the reader cannot parse. The other is a
// bullet restating a relationship in words that differ from the ones already written —
// which is not hypothetical, it is 7 skills in the real corpus, each stating its
// relationships twice with the longer explanation second.
//
// A target is dropped only when nothing goes with it, and then the bullet disappears only
// if every target it named was such a one.
func canonicalize(b logicalBullet, lines []string, written rationales) []string {
	edges, understood := readBullet(b.text)
	if !understood {
		return lines[b.start:b.end]
	}
	fresh, lossy := written.enter(edges)
	switch {
	case lossy:
		return lines[b.start:b.end]
	case len(fresh) == 0:
		return nil
	}
	out := make([]string, 0, len(fresh))
	for _, e := range fresh {
		out = append(out, Bullet(e))
	}
	return out
}

// enter takes one bullet's edges into the record, returning those no earlier bullet had
// stated and whether writing the bullet canonically would delete a rationale.
//
// One pass answers both, deliberately: asked as two functions they would have to run in
// one order — the deletion question is about what *earlier* bullets said, so it cannot be
// asked after this bullet's own targets are recorded — and an ordering a caller has to
// know is one it can get wrong.
//
// **Rationales are compared exactly, not approximately.** A looser test would delete a
// line on the strength of a similarity score, and the asymmetry that governs everything
// here is that a deleted line is invisible afterwards while a kept duplicate is plain to
// any reader.
//
// An edge carrying no rationale of its own cannot lose one, so it never makes a bullet
// lossy — which is what lets a bare "- depends-on: `a`" collapse into an explained bullet
// for the same relationship instead of pinning both in place forever.
//
// Every new target is recorded even when the caller goes on to keep the bullet verbatim:
// that line still declares those relationships, so a third bullet naming them is a
// restatement of it.
func (r rationales) enter(edges []Edge) (fresh []Edge, lossy bool) {
	fresh = make([]Edge, 0, len(edges))
	for _, e := range edges {
		key := edgeKey{kind: e.Kind, target: e.Target}
		prior, already := r[key]
		switch {
		case !already:
			r[key] = e.Rationale
			fresh = append(fresh, e)
		case e.Rationale != "" && e.Rationale != prior:
			lossy = true
		}
	}
	return fresh, lossy
}

// withBullets returns body with extra inserted after its last bullet, so hoisted
// bullets join the list rather than landing after the blank line and thematic break
// that close the section. A section with no bullets yet takes them after the blank
// line that follows its heading, which keeps the list separated from the heading.
func withBullets(body, extra []string) []string {
	if len(extra) == 0 {
		return body
	}
	at := 0
	for at < len(body) && strings.TrimSpace(body[at]) == "" {
		at++
	}
	for i, line := range body {
		if isBulletLine(strings.TrimSpace(line)) {
			at = i + 1
		}
	}
	out := make([]string, 0, len(body)+len(extra))
	out = append(out, body[:at]...)
	out = append(out, extra...)
	return append(out, body[at:]...)
}

// mergeable reports whether sec can be folded into an earlier section without losing
// content: every line of it other than the heading must be part of a bullet, blank, or
// a thematic break. The bullets move; the blanks and the break are dropped along with
// the heading, the break having separated a section that no longer exists.
//
// All 13 two-section skills in the real books qualify — heading, blank, bullets, blank,
// break — so the guard costs nothing there and refuses the case it exists for: a later
// section holding prose, a sub-heading, or fenced code.
func mergeable(lines []string, sec section) bool {
	inBullet := make(map[int]bool, sec.end-sec.head)
	for _, b := range sectionBullets(lines, sec.head, sec.end) {
		for i := b.start; i < b.end; i++ {
			inBullet[i] = true
		}
	}
	for i := sec.head + 1; i < sec.end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if inBullet[i] || trimmed == "" || isThematicBreak(trimmed) {
			continue
		}
		return false
	}
	return true
}

// isThematicBreak reports whether a trimmed line is a horizontal rule: three or more of
// `-`, `_` or `*`, and nothing else.
//
// All three characters, because the corpus writes the underscore form. `mdformat` renders
// every rule as `______…`, so recognizing only the dash form made mergeable refuse the
// exact files the merge was written for — 7 skills whose second section is closed by an
// underscore rule were normalized in place and left holding an emptied heading.
//
// The spaced forms CommonMark also allows ("* * *") are deliberately not recognized:
// isBulletLine reads that one as a bullet, since it opens with the "* " marker, and a
// line that is both a bullet and a break would make mergeable's answer depend on which
// predicate ran first. The unspaced forms cannot collide — "---" is excluded from
// isBulletLine by name, and "___" / "***" carry no bullet marker.
func isThematicBreak(trimmed string) bool {
	if len(trimmed) < 3 {
		return false
	}
	switch trimmed[0] {
	case '-', '_', '*':
		return strings.Trim(trimmed, trimmed[:1]) == ""
	default:
		return false
	}
}
