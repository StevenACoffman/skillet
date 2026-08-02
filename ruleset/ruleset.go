// Package ruleset is the typed model of a distilled ruleset: a set of Rules,
// each an imperative with a severity, a level, a rationale, and a ✗/✓ example
// pair. Render emits the canonical text form and Parse reads it back; the two
// round-trip. Parse handles the canonical form Render emits, not every hand-
// authored variation a distilled Markdown file may contain.
package ruleset

import (
	"fmt"
	"regexp"
	"strings"
)

// Severity is how strictly a Rule is enforced.
const (
	MUST     Severity = "MUST"
	SHOULD   Severity = "SHOULD"
	CONSIDER Severity = "CONSIDER"
)

// Level is where a Rule applies.
const (
	CODE   Level = "CODE"
	ARCH   Level = "ARCH"
	METHOD Level = "METHOD"
)

// indent leads a rule's rationale and example lines in the canonical form.
const indent = "      "

// ruleHeaderRE matches "§<section>  [<SEVERITY>][<LEVEL>]  <statement>".
var ruleHeaderRE = regexp.MustCompile(`^§(\S+)\s+\[([A-Z]+)\]\[([A-Z]+)\]\s+(.*)$`)

// Severity is how strictly a Rule is enforced.
type Severity string

// Level is where a Rule applies.
type Level string

// Rule is one atomic, mechanically applicable constraint.
type Rule struct {
	Section   string
	Severity  Severity
	Level     Level
	Statement string
	Rationale string
	Bad       string // the ✗ counter-example
	Good      string // the ✓ preferred form
}

// Ruleset is a distilled set of Rules derived from one source.
type Ruleset struct {
	Source string
	Scope  string
	Rules  []Rule
}

// Valid reports whether s is a known severity.
func (s Severity) Valid() bool {
	switch s {
	case MUST, SHOULD, CONSIDER:
		return true
	default:
		return false
	}
}

// Valid reports whether l is a known level.
func (l Level) Valid() bool {
	switch l {
	case CODE, ARCH, METHOD:
		return true
	default:
		return false
	}
}

// Render emits rs in the canonical text form. It is deterministic: the same
// Ruleset always renders byte-identically.
func Render(rs Ruleset) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Source: %s\n", rs.Source)
	fmt.Fprintf(&b, "Scope:  %s\n", rs.Scope)
	for i := range rs.Rules {
		r := &rs.Rules[i]
		b.WriteString("\n")
		fmt.Fprintf(&b, "§%s  [%s][%s]  %s\n", r.Section, r.Severity, r.Level, r.Statement)
		if r.Rationale != "" {
			fmt.Fprintf(&b, "%s%s\n", indent, r.Rationale)
		}
		if r.Bad != "" {
			fmt.Fprintf(&b, "%s✗  %s\n", indent, r.Bad)
		}
		if r.Good != "" {
			fmt.Fprintf(&b, "%s✓  %s\n", indent, r.Good)
		}
	}
	return b.String()
}

// Parse reads the canonical form Render emits. A malformed rule header or an
// unknown severity/level is an error, not a silent skip.
func Parse(md string) (Ruleset, error) {
	var (
		rs  Ruleset
		cur *Rule
	)
	flush := func() {
		if cur != nil {
			rs.Rules = append(rs.Rules, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Source:"):
			rs.Source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
		case strings.HasPrefix(line, "Scope:"):
			rs.Scope = strings.TrimSpace(strings.TrimPrefix(line, "Scope:"))
		case strings.HasPrefix(line, "§"):
			flush()
			r, err := parseHeader(line)
			if err != nil {
				return Ruleset{}, err
			}
			cur = &r
		case cur != nil && trimmed != "":
			applyBody(cur, trimmed)
		}
	}
	flush()
	return rs, nil
}

func parseHeader(line string) (Rule, error) {
	m := ruleHeaderRE.FindStringSubmatch(line)
	if m == nil {
		return Rule{}, fmt.Errorf("ruleset: malformed rule header: %q", line)
	}
	sev, lvl := Severity(m[2]), Level(m[3])
	if !sev.Valid() {
		return Rule{}, fmt.Errorf("ruleset: unknown severity %q in %q", sev, line)
	}
	if !lvl.Valid() {
		return Rule{}, fmt.Errorf("ruleset: unknown level %q in %q", lvl, line)
	}
	return Rule{Section: m[1], Severity: sev, Level: lvl, Statement: strings.TrimSpace(m[4])}, nil
}

func applyBody(r *Rule, trimmed string) {
	switch {
	case strings.HasPrefix(trimmed, "✗"):
		r.Bad = strings.TrimSpace(strings.TrimPrefix(trimmed, "✗"))
	case strings.HasPrefix(trimmed, "✓"):
		r.Good = strings.TrimSpace(strings.TrimPrefix(trimmed, "✓"))
	case r.Rationale == "":
		r.Rationale = trimmed
	default:
		r.Rationale += " " + trimmed
	}
}
