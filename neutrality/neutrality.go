// Package neutrality implements the runtime-neutrality red-light scan shared by
// skillet tools. It flags wording or paths that bind a skill to a single agent
// runtime, which makes other runtimes refuse to install it. The pattern is
// byte-identical to the darwin source (skillsaw scan / exegesis lint) so the
// tools never disagree about what "runtime-bound" means.
package neutrality

import (
	"regexp"
	"strings"
)

// redLightPattern is the exact red-light regex from the darwin source (SKILL.md
// §"Runtime 适配性审查"). It is applied per line, so "^" anchors to each line's
// start. It is a const rather than a compiled package var so the package holds
// no global state; Scan compiles it once per call.
const redLightPattern = `(在 Claude Code|Claude Code skill|Claude Code 用户|Cursor only|Codex 中|^\[!\[Claude Code|~/\.claude/skills/[a-z]|/plugin install\b)`

// dynamicInlinePattern matches Claude Code's inline dynamic-context syntax: an
// exclamation mark, then a code span holding a command. The mark must not itself sit
// inside a code span, which is how prose writes about a literal "!" (for example
// "prefix `!` to negate"). It is applied per line.
const dynamicInlinePattern = "(^|[^`])!`[^`]+`"

// Hit is one red-light match.
type Hit struct {
	File string `json:"file"`
	Line int    `json:"line"` // 1-indexed
	Text string `json:"text"` // the matching line, trimmed
}

// NamedFile pairs a display name with file contents for Scan.
type NamedFile struct {
	Name    string
	Content string
}

// Scan runs the red-light regex line-by-line over each file and returns every
// match. Hits follow the given file order, then line number. It is pure: values
// in, values out, no filesystem access. CRLF line endings are normalized so a
// match and its 1-indexed line number are independent of the source's newlines.
func Scan(files []NamedFile) []Hit {
	redLight := regexp.MustCompile(redLightPattern)
	var hits []Hit
	for _, f := range files {
		lines := strings.Split(strings.ReplaceAll(f.Content, "\r\n", "\n"), "\n")
		for i, line := range lines {
			if redLight.MatchString(line) {
				hits = append(hits, Hit{File: f.Name, Line: i + 1, Text: strings.TrimSpace(line)})
			}
		}
	}
	return hits
}

// DynamicContext reports Claude Code's dynamic-context syntax, which binds a skill to
// that runtime in a way Scan's wording check cannot see: Claude Code runs the command
// before the model reads the skill, and every other client shows the raw text. Two
// forms are reported, each at the line it starts on:
//
//   - an inline "!" immediately followed by a code span, outside any fenced block;
//   - a fenced block whose info string starts with "!".
//
// An ordinary fenced block is example code, and an inline form written inside one is a
// skill documenting the syntax rather than using it, so it is skipped.
//
// Requires: nothing.
// Ensures:  hits follow file order, then line order; it is pure. CRLF is normalized.
func DynamicContext(files []NamedFile) []Hit {
	inline := regexp.MustCompile(dynamicInlinePattern)
	var hits []Hit
	for _, f := range files {
		inFence := false
		lines := strings.Split(strings.ReplaceAll(f.Content, "\r\n", "\n"), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if fence, info := fenceLine(trimmed); fence {
				if !inFence && strings.HasPrefix(info, "!") {
					hits = append(hits, Hit{File: f.Name, Line: i + 1, Text: trimmed})
				}
				inFence = !inFence
				continue
			}
			if !inFence && inline.MatchString(line) {
				hits = append(hits, Hit{File: f.Name, Line: i + 1, Text: trimmed})
			}
		}
	}
	return hits
}

// fenceLine reports whether trimmed opens or closes a fenced block, and its info string.
func fenceLine(trimmed string) (bool, string) {
	for _, marker := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, marker) {
			return true, strings.TrimSpace(strings.TrimLeft(trimmed, marker[:1]))
		}
	}
	return false, ""
}
