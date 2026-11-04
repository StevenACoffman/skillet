package ruleset

import (
	"fmt"
	"regexp"
	"strings"
)

// githubLayout is the first format version whose rules render for a Markdown viewer:
// one element per line, an anchor on every rule header, and rule references linked.
//
// **Why a version and not a change to Render.** Every stored ruleset renders
// byte-identically today, and canonizer reports a file whose bytes differ from Render's as
// non-canonical at error severity. Changing the layout for everyone would turn every
// ruleset in every corpus blocking at once, for a reason belonging to none of them. A
// document opts in by declaring `format: 5`, which is the same opt-in the version reader
// already offers for every other feature.
//
// **What a viewer did with the old layout.** GitHub renders a file in CommonMark, where a
// rule's indented lines are paragraph continuations, so the statement, rationale, ✗, ✓ and
// ↦ ran together as one paragraph. And a reference such as "§4.13" was plain text a
// reader had to find by scrolling.
const githubLayout = 5

// refPattern matches a rule reference: `§` and a dotted number such as 4.13 or 2.10.1.
//
// A function rather than a package variable, the shape this module uses for patterns and
// the one gochecknoglobals leaves alone. Greedy, so §4.13 is never read as §4.1 followed by
// "3".
func refPattern() *regexp.Regexp { return regexp.MustCompile(`§(\d+(?:\.\d+)+)`) }

// linkPattern matches a reference as linkRefs writes it: `[§4.13](#r4.13)`.
func linkPattern() *regexp.Regexp {
	return regexp.MustCompile(`\[§(\d+(?:\.\d+)+)\]\(#r(\d+(?:\.\d+)+)\)`)
}

// ruleAnchor is the HTML anchor a version-5 rule header ends with.
//
// An `<a id>` rather than a Markdown heading, because GitHub keeps the element (prefixing
// the id with `user-content-`, which its page script maps a plain `#r4.13` link to), and a
// heading would derive its id from the statement's text, changing it whenever the rule is
// reworded. A text-fragment link was the other candidate and is not usable here: GitHub's
// sanitizer drops a link whose target is a bare `#:~:text=` directive.
func ruleAnchor(section string) string { return `<a id="r` + section + `"></a>` }

// linkRefs rewrites each reference to a rule of this ruleset as a link to its anchor.
//
// Only ids in sections are linked: a reference to a rule that does not exist, such as one a
// rework deleted, stays plain rather than linking nowhere. Text inside backtick code spans
// is left alone, because a link there renders as its literal syntax.
//
// Requires: sections holds the ids of every rule in the ruleset.
// Ensures:  unlinkRefs(linkRefs(s, ids)) == s for any s; it is pure.
func linkRefs(s string, sections map[string]bool) string {
	parts := strings.Split(s, "`")
	pattern := refPattern()
	for i := 0; i < len(parts); i += 2 {
		parts[i] = pattern.ReplaceAllStringFunc(parts[i], func(ref string) string {
			id := strings.TrimPrefix(ref, "§")
			if !sections[id] {
				return ref
			}
			return "[" + ref + "](#r" + id + ")"
		})
	}
	return strings.Join(parts, "`")
}

// unlinkRefs reverses linkRefs, turning `[§4.13](#r4.13)` back into `§4.13`.
//
// A link whose text and target disagree is left as written: linkRefs never produces one, so
// it is the author's own Markdown and not this layout's to remove. Applied once, which is
// what keeps text that already held a link round-tripping: linkRefs wraps the inner
// reference and this unwraps exactly that.
//
// Code spans are skipped exactly as linkRefs skips them, or a link written inside one --
// which linkRefs left alone -- would be unwrapped on the way back.
//
// Ensures: it is pure.
func unlinkRefs(s string) string {
	parts := strings.Split(s, "`")
	pattern := linkPattern()
	for i := 0; i < len(parts); i += 2 {
		parts[i] = pattern.ReplaceAllStringFunc(parts[i], func(link string) string {
			m := pattern.FindStringSubmatch(link)
			if m[1] != m[2] {
				return link
			}
			return "§" + m[1]
		})
	}
	return strings.Join(parts, "`")
}

// renderGitHubRule writes one rule in the version-5 layout.
//
// Every line but the rule's last ends with a CommonMark hard break, a trailing backslash,
// so a viewer shows one element per line. The last line has none, because a backslash
// ending a paragraph renders as a literal backslash.
//
// Requires: sections holds the ids of every rule in the ruleset.
// Ensures:  parseGitHubLines applied to the result restores the lines renderRule writes for
// r, with references unlinked by unlinkRules; it is pure.
func renderGitHubRule(b *strings.Builder, r *Rule, sections map[string]bool) {
	lines := []string{fmt.Sprintf("§%s  [%s][%s]  %s%s",
		r.Section, r.Severity, r.Level, linkRefs(r.Statement, sections), ruleAnchor(r.Section))}
	if r.Rationale != "" {
		lines = append(lines, indent+linkRefs(r.Rationale, sections))
	}
	if r.Bad != "" {
		lines = append(lines, indent+"✗  "+linkRefs(r.Bad, sections))
	}
	if r.Good != "" {
		lines = append(lines, indent+"✓  "+linkRefs(r.Good, sections))
	}
	// The anchor, warrant and checks are not linked: an anchor's § names a section of the
	// *source*, often numbered (`§11.1.2 Pattern 2`), and linking it into this file would
	// send the reader to the wrong document.
	if r.SourceAnchor != "" {
		lines = append(lines, indent+"↦  "+r.SourceAnchor)
	}
	if r.Warrant.Present() {
		lines = append(lines, fmt.Sprintf("%s⚖  %s %s  %s",
			indent, r.Warrant.By, r.Warrant.At, r.Warrant.Rationale))
	}
	for _, c := range r.Checks {
		lines = append(lines, fmt.Sprintf("%s⊨  %s  %s", indent, c.Op, c.Arg))
	}
	b.WriteString(strings.Join(lines, "\\\n"))
	b.WriteString("\n")
}

// parseGitHubLines undoes the version-5 line layout before the shared parser reads lines:
// it removes the hard break from every line followed by another line of the same rule, and
// the anchor from every rule header.
//
// A hard break is removed only where renderGitHubRule wrote one -- a line whose next line
// continues the rule -- so a rule's last line keeps a trailing backslash of its own, and a
// continued line whose content ends in one keeps that one too, since Render adds a second.
//
// Ensures: len(result) == len(lines); it is pure.
func parseGitHubLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		continued := i+1 < len(lines) &&
			strings.TrimSpace(lines[i+1]) != "" && !strings.HasPrefix(lines[i+1], "§")
		if continued {
			line = strings.TrimSuffix(line, `\`)
		}
		if strings.HasPrefix(line, "§") {
			if m := ruleHeaderRE.FindStringSubmatch(line); m != nil {
				line = strings.TrimSuffix(line, ruleAnchor(m[1]))
			}
		}
		out[i] = line
	}
	return out
}

// ruleIDs returns the set of rule sections in rs, which linkRefs links to.
//
// Ensures: it is pure.
func ruleIDs(rs *Ruleset) map[string]bool {
	ids := make(map[string]bool, len(rs.Rules))
	for i := range rs.Rules {
		ids[rs.Rules[i].Section] = true
	}
	return ids
}

// layoutLines returns the lines Parse's shared loop should read: lines unchanged below
// version 5, and with the version-5 layout undone at or above it.
//
// Ensures: len(result) == len(lines); it is pure.
func layoutLines(format int, lines []string) []string {
	if format < githubLayout {
		return lines
	}
	return parseGitHubLines(lines)
}

// unlinkRules reverses linkRefs on the fields renderGitHubRule links, in a version-5
// document, and does nothing below version 5.
func unlinkRules(rs *Ruleset) {
	if rs.Format < githubLayout {
		return
	}
	for i := range rs.Rules {
		r := &rs.Rules[i]
		r.Statement = unlinkRefs(r.Statement)
		r.Rationale = unlinkRefs(r.Rationale)
		r.Bad = unlinkRefs(r.Bad)
		r.Good = unlinkRefs(r.Good)
	}
}
