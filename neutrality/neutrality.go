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
