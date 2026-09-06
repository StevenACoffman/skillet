// Package ruleset is the typed model of a distilled ruleset: a set of Rules,
// each an imperative with a severity, a level, a rationale, a ✗/✓ example pair,
// and an optional ↦ source anchor. Render emits the canonical text form and Parse
// reads it back; the two round-trip. Parse handles the canonical form Render
// emits, not every hand-authored variation a distilled Markdown file may contain.
package ruleset

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/StevenACoffman/skillet/frontmatter"
	"github.com/StevenACoffman/skillet/judge"
)

// FormatVersion is the canonical-form major version this package writes and is the
// highest it can read.
//
// Bump it only when the grammar itself changes -- not to record provenance, tool identity
// or scoring metadata. identity.Hash already pins which bytes produced what, and a format
// version that accumulates those becomes a second manifest.
//
// It is 3. Version 1 was the version reader itself, which changed no grammar; version 2
// adds the ⚖ warrant marker; version 3 adds the Limitations: header and the ⊨ check marker,
// batched into one bump because each is a grammar change and shipping them apart would
// migrate every stored ruleset twice. A document is only *written* at a version when it
// uses something that version introduced -- see formatOf -- so every ruleset written before
// still renders byte-identically, which is the property the reader was shipped early to
// protect.
const FormatVersion = 3

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
	Section      string
	Severity     Severity
	Level        Level
	Statement    string
	Rationale    string
	Bad          string // the ✗ counter-example
	Good         string // the ✓ preferred form
	SourceAnchor string // the ↦ source quote or section this rule derives from

	// Warrant is the ⚖ record of a decision, for a rule no source can anchor. Its zero
	// value means the rule was never adjudicated, which is the ordinary case.
	Warrant Warrant

	// Checks are the ⊨ predicates that decide whether this rule fires, and they exist so a
	// rule can be known-answer tested against its own examples.
	//
	// A rule already ships both answers -- Bad is the case it must flag and Good the case
	// it must not -- and carried no way to run itself against them, so nothing could tell a
	// rule that discriminates from one that would fire on ordinary work. See Sound, which
	// is the check these make possible; a rule with no checks is untested rather than
	// unsound, and that is a different claim.
	//
	// Empty for every rule written before version 3, and Render emits nothing for it, so an
	// existing document is untouched.
	Checks []judge.Check
}

// Ruleset is a distilled set of Rules derived from one source.
type Ruleset struct {
	Source string
	Scope  string

	// Limitations is what this ruleset does not cover, and it is the counterpart to Scope
	// rather than a second phrasing of it.
	//
	// A capability statement that will not say what it does not cover is an advertisement:
	// rules distilled from one book and presented without that book's bounds read as rules
	// for the whole subject. Scope alone cannot carry this, because a scope naming only what
	// is included is exactly the shape being objected to.
	//
	// Empty is the ordinary case for every ruleset written before version 3, and Render
	// omits the header entirely when it is empty -- so an existing document is untouched
	// and does not suddenly declare a version it does not need.
	Limitations string
	// Format is the canonical-form major version this ruleset is written in. A file that
	// declares none is 1, so the zero value reads correctly for every ruleset written before
	// versioning existed -- unlike finding.Action, whose zero value had to mean "nobody
	// judged", a missing format genuinely *is* version 1.
	Format int
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

// readFormat takes the optional leading YAML block off md and returns the canonical-form
// major version it declares, defaulting to 1 when there is none.
//
// A version newer than this parser understands is an error rather than a best effort. The
// whole reason the block exists is that an unknown marker line is otherwise folded into a
// rule's rationale, silently: refusing loudly is the behaviour being bought.
//
// Requires: nothing.
// Ensures:  the returned format is in [1, FormatVersion]; body is md with any leading YAML
//
//	block removed; it is pure.
func readFormat(md string) (format int, body string, err error) {
	block, body := frontmatter.Split(md)
	if strings.TrimSpace(block) == "" {
		return 1, body, nil
	}
	var header struct {
		Format int `yaml:"format"`
	}
	if uerr := yaml.Unmarshal([]byte(block), &header); uerr != nil {
		return 0, "", fmt.Errorf("ruleset: unreadable format header: %w", uerr)
	}
	switch {
	case header.Format == 0:
		// A block that declares no format is v1 with metadata, not a malformed version.
		return 1, body, nil
	case header.Format < 1:
		return 0, "", fmt.Errorf("ruleset: format %d is not a version", header.Format)
	case header.Format > FormatVersion:
		return 0, "", fmt.Errorf(
			"ruleset: format %d is newer than this parser understands (%d)",
			header.Format, FormatVersion)
	}
	return header.Format, body, nil
}

// renderFormat emits the version block, and only above version 1.
//
// Silence at 1 is what keeps this change inert: every ruleset written before versioning
// existed renders byte-identically, so the canonical-form round-trip check canonizer is
// adding does not report drift on files nobody touched.
//
// Written by hand rather than marshalled. The canonical form's promise is byte-stability,
// and a marshaller's key order, quoting and line endings are its choice rather than ours.
// formatOf returns the lowest canonical-form version that can express rs.
//
// The version is derived from what the document uses rather than read from the field a
// caller set, because those can disagree and only one of them is true. A ruleset carrying a
// warrant but declaring version 1 would render a ⚖ line under no version block: a v1 reader
// rejects it, so nothing is silently mis-parsed, but the file would describe itself
// wrongly and the next tool to round-trip it would report drift that is not there.
//
// Ensures: pure. Never below 1, so a Ruleset built in Go without setting Format is a valid
// v1 document rather than a malformed one, and never above what its content requires, which
// is what keeps a corpus of warrant-free rulesets rendering byte-identically.
func formatOf(rs *Ruleset) int {
	if rs.Limitations != "" {
		return 3
	}
	for i := range rs.Rules {
		if len(rs.Rules[i].Checks) > 0 {
			return 3
		}
	}
	for i := range rs.Rules {
		if rs.Rules[i].Warrant.Present() {
			return 2
		}
	}
	if rs.Format > 1 {
		return rs.Format
	}
	return 1
}

func renderFormat(format int) string {
	if format <= 1 {
		return ""
	}
	return fmt.Sprintf("---\nformat: %d\n---\n", format)
}

// Render emits rs in the canonical text form. It is deterministic: the same
// Ruleset always renders byte-identically.
//
// A Format of 0 renders as version 1, so a Ruleset built in Go without setting it is a
// valid v1 ruleset rather than a malformed one. Parse returns 1 for an undeclared file, so
// the two agree on what a version-less ruleset is.
func Render(rs *Ruleset) string {
	var b strings.Builder
	b.WriteString(renderFormat(formatOf(rs)))
	fmt.Fprintf(&b, "Source: %s\n", rs.Source)
	fmt.Fprintf(&b, "Scope:  %s\n", rs.Scope)
	// Appended after the two headers every document already has, so a ruleset gaining
	// limitations shows a one-line diff rather than a reordered head.
	if rs.Limitations != "" {
		fmt.Fprintf(&b, "Limitations: %s\n", rs.Limitations)
	}
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
		if r.SourceAnchor != "" {
			fmt.Fprintf(&b, "%s↦  %s\n", indent, r.SourceAnchor)
		}
		if r.Warrant.Present() {
			fmt.Fprintf(&b, "%s⚖  %s %s  %s\n",
				indent, r.Warrant.By, r.Warrant.At, r.Warrant.Rationale)
		}
		// One line per check, in the order given: a rule's checks are a conjunction, and
		// re-ordering them on render would change the bytes without changing the meaning,
		// which is what the inert-render property forbids.
		for _, c := range r.Checks {
			fmt.Fprintf(&b, "%s⊨  %s  %s\n", indent, c.Op, c.Arg)
		}
	}
	return b.String()
}

// Parse reads the canonical form Render emits. A malformed rule header or an
// unknown severity/level is an error, not a silent skip.
func Parse(md string) (Ruleset, error) {
	format, body, err := readFormat(md)
	if err != nil {
		return Ruleset{}, err
	}
	md = body
	var (
		rs  Ruleset
		cur *Rule
	)
	rs.Format = format
	flush := func() {
		if cur != nil {
			rs.Rules = append(rs.Rules, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case applyMeta(&rs, line):
		case strings.HasPrefix(line, "§"):
			flush()
			r, err := parseHeader(line)
			if err != nil {
				return Ruleset{}, err
			}
			cur = &r
		case cur != nil && trimmed != "":
			if err := applyBody(cur, trimmed); err != nil {
				return Ruleset{}, err
			}
		}
	}
	flush()
	return rs, nil
}

// applyMeta consumes a ruleset-level metadata line, reporting whether it did.
//
// It returns a bool rather than setting a field and falling through, so Parse's
// switch has one arm for "this line is metadata" instead of one per key — which is
// what keeps adding a key from making the dispatch harder to read.
func applyMeta(rs *Ruleset, line string) bool {
	switch {
	case strings.HasPrefix(line, "Source:"):
		rs.Source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
	case strings.HasPrefix(line, "Scope:"):
		rs.Scope = strings.TrimSpace(strings.TrimPrefix(line, "Scope:"))
	case strings.HasPrefix(line, "Limitations:"):
		rs.Limitations = strings.TrimSpace(strings.TrimPrefix(line, "Limitations:"))
	default:
		return false
	}
	return true
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
