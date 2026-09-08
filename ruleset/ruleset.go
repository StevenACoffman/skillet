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
	"github.com/StevenACoffman/skillet/verification"
)

// FormatVersion is the canonical-form major version this package writes and is the
// highest it can read.
//
// Bump it only when the grammar itself changes -- not to record metadata identity.Hash
// already establishes, such as tool identity or scoring. A hash pins which bytes produced
// what, and a format version that accumulates the same facts becomes a second manifest.
//
// **That prohibition read "not to record provenance" until version 4, and the wording was
// wider than its own reason.** The ground it gives is that identity.Hash covers the fact
// already, which is true of tool identity and untrue of a verification event: a hash
// establishes which bytes exist and cannot say who read them and agreed. Version 4 records
// exactly that, so the sentence was narrowed to what it argues for rather than bent. The
// "second manifest" warning stands and is still the right fear.
//
// It is 4. Version 1 was the version reader itself, which changed no grammar; version 2
// adds the ⚖ warrant marker; version 3 adds the Limitations: header and the ⊨ check marker,
// batched into one bump because each is a grammar change and shipping them apart would
// migrate every stored ruleset twice; version 4 adds the frontmatter block's verified key.
//
// **Version 4 is a bump rather than a tolerated unknown key, and the alternative is worse.**
// An unmodelled verified: already parses without error and is then dropped by Render, so
// leaving the version at 3 would let an older tool round-trip a verified ruleset and lose a
// human's judgement with no error at all. That is the silent loss the block was introduced
// to convert into a loud refusal -- see readFrontmatter -- so a key a reader must model to
// preserve is a grammar change by this constant's own test.
//
// A document is only *written* at a version when it uses something that version introduced
// -- see formatOf -- so every ruleset written before still renders byte-identically, which
// is the property the reader was shipped early to protect.
const FormatVersion = 4

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
	// Format and Verified are the frontmatter block; the three fields above are body
	// headers. They are grouped by where they live in the file because that is the question
	// a reader of this struct is usually answering.

	// Format is the canonical-form major version this ruleset is written in. A file that
	// declares none is 1, so the zero value reads correctly for every ruleset written before
	// versioning existed -- unlike finding.Action, whose zero value had to mean "nobody
	// judged", a missing format genuinely *is* version 1.
	Format int

	// Verified is the independent verification events attesting to this ruleset: who
	// confirmed it, and when. It is OKF §5.2's list rather than a single trust tier, and
	// deriving a tier from it belongs to a consumer -- see the verification package, which
	// leaves the fold out for the same reason.
	//
	// **The events attest to the rules, not to the bytes.** identity.Hash pins bytes and a
	// proof packet binds a ruleset to its source; neither can say a person read the rules
	// and agreed, which is what this carries and why it is not derivable from content.
	// The corollary is that an event outlives the text it attested to: nothing here
	// re-checks it when a rule changes, so a consumer comparing an event against a later
	// revision is reading a claim about an earlier one.
	//
	// Empty is the ordinary case for every ruleset written before version 4, and Render
	// emits nothing for it, so an existing document is untouched.
	Verified []verification.Event

	Rules []Rule
}

// frontmatterBlock is the ruleset's leading YAML block, and it is a type rather than a
// pair of return values because readFrontmatter would otherwise return four things.
//
// The yaml tags are not strictly needed -- yaml.v3 lowercases field names, so By and At
// would land on by and at regardless -- and they are written because without them the wire
// format is an implicit consequence of the Go names, and a rename would change the format
// silently.
type frontmatterBlock struct {
	Format   int                  `yaml:"format"`
	Verified []verification.Event `yaml:"verified"`
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

// readFrontmatter takes the optional leading YAML block off md and returns what it
// declares, defaulting to version 1 when there is none.
//
// A version newer than this parser understands is an error rather than a best effort. The
// whole reason the block exists is that an unknown marker line is otherwise folded into a
// rule's rationale, silently: refusing loudly is the behaviour being bought.
//
// It was readFormat until version 4 gave the block a second key. A function named for one
// field that returns two is the name telling a reader less than the code does, so it reads
// the block and is named for it.
//
// **Nothing here tolerates a bare actor or a bare mapping under verified.** gnosis accepts
// both, because OKF §11 forbids rejecting a conformant document over an optional family's
// shape and gnosis reads documents that predate its own reader. This key has no such
// population: version 4 invents it, so there is no legacy to be lenient towards, and
// writing the leniency anyway would be special-case handling for a case that cannot exist.
//
// Requires: nothing.
// Ensures:  the returned Format is in [1, FormatVersion]; body is md with any leading YAML
//
//	block removed; it is pure.
func readFrontmatter(md string) (frontmatterBlock, string, error) {
	block, body := frontmatter.Split(md)
	if strings.TrimSpace(block) == "" {
		return frontmatterBlock{Format: 1}, body, nil
	}
	var header frontmatterBlock
	if uerr := yaml.Unmarshal([]byte(block), &header); uerr != nil {
		return frontmatterBlock{}, "", fmt.Errorf("ruleset: unreadable frontmatter: %w", uerr)
	}
	switch {
	case header.Format == 0:
		// A block that declares no format is v1 with metadata, not a malformed version.
		header.Format = 1
	case header.Format < 1:
		return frontmatterBlock{}, "", fmt.Errorf(
			"ruleset: format %d is not a version", header.Format)
	case header.Format > FormatVersion:
		return frontmatterBlock{}, "", fmt.Errorf(
			"ruleset: format %d is newer than this parser understands (%d)",
			header.Format, FormatVersion)
	}
	return header, body, nil
}

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
	// First, because the clauses below return early and run highest-version-first: a
	// verified ruleset that declares no Limitations: must not fall through to 3.
	if len(rs.Verified) > 0 {
		return 4
	}
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

// renderFrontmatter emits the leading YAML block, and only above version 1.
//
// Silence at 1 is what keeps this change inert: every ruleset written before versioning
// existed renders byte-identically, so the canonical-form round-trip check canonizer is
// adding does not report drift on files nobody touched.
//
// Written by hand rather than marshalled. The canonical form's promise is byte-stability,
// and a marshaller's key order, quoting and line endings are its choice rather than ours.
func renderFrontmatter(format int, verified []verification.Event) string {
	if format <= 1 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\nformat: %d\n", format)
	if len(verified) > 0 {
		b.WriteString("verified:\n")
		for i := range verified {
			fmt.Fprintf(&b, "  - by: %q\n", verified[i].By)
			if verified[i].At != "" {
				fmt.Fprintf(&b, "    at: %q\n", verified[i].At)
			}
		}
	}
	b.WriteString("---\n")
	return b.String()
}

// Render emits rs in the canonical text form. It is deterministic: the same
// Ruleset always renders byte-identically.
//
// A Format of 0 renders as version 1, so a Ruleset built in Go without setting it is a
// valid v1 ruleset rather than a malformed one. Parse returns 1 for an undeclared file, so
// the two agree on what a version-less ruleset is.
func Render(rs *Ruleset) string {
	var b strings.Builder
	b.WriteString(renderFrontmatter(formatOf(rs), rs.Verified))
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
	header, body, err := readFrontmatter(md)
	if err != nil {
		return Ruleset{}, err
	}
	md = body
	var (
		rs  Ruleset
		cur *Rule
	)
	rs.Format = header.Format
	rs.Verified = header.Verified
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
