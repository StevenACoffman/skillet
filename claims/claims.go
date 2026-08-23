// Package claims extracts the checkable assertions a document makes about its own
// repository.
//
// A skill body naming `make verify` or `bw sync` claims those commands exist. Nothing
// checks it: `lint` covers links, and reachability is a separate question, so commands
// are the uncovered half.
//
// # The environment boundary, which is the whole reason this package is small
//
// Finding a command-shaped claim in a body is a pure text function and lives here.
// **Resolving one to an executable is environment-dependent and does not.** Link
// resolution can be answered from the tree; command resolution depends on `$PATH`, on
// what is installed, and on the shell — so a kernel that answered it would be
// answering differently on every machine, and a caller could not tell a missing
// command from a missing toolchain.
//
// Evidence out, policy to the caller: the same boundary `skilllens` draws. This
// package imports nothing outside the standard library and `TestImportsNothing` says
// so, because a boundary with no check is a comment.
package claims

import (
	"regexp"
	"sort"
	"strings"
)

// executable matches a plausible command name: lowercase, digits, and the punctuation
// that appears in real program names, optionally prefixed by an explicit relative path.
//
// Anchored at both ends, so a token carrying anything else — parentheses, an uppercase
// letter, an operator — is not a command name. That is what excludes `foo.Bar()` and
// prose, which are the two large classes of code span that are not invocations.
var executable = regexp.MustCompile(`^(\./)?[a-z][a-z0-9._/-]*$`)

// isOperator reports a token that makes a span an expression rather than an
// invocation. A switch rather than a package-level map: the set is fixed, and a
// mutable global is a thing any caller in the process can rewrite.
func isOperator(token string) bool {
	switch token {
	case "=", ":=", "==", "+=", "<-":
		return true
	default:
		return false
	}
}

// hasUpper reports an uppercase letter anywhere in the span.
//
// This is the discriminator that separates an invocation from a *declaration*, and it
// was added because a test found `var Doc struct` reported as a command — first token
// lowercase, two more tokens, no operator. `type Foo interface` and `func Bar() error`
// are the same class, and in a corpus of skills about Go they are common enough to
// matter.
//
// A keyword list was the obvious alternative and is worse: it would be a maintained
// list, it would be Go's keywords in a kernel whose consumers document many languages,
// and it is the second-place-to-remember smell this repo names elsewhere. Case needs no
// list.
//
// What it costs is real and is the direction already chosen: `git push origin HEAD` and
// `make DEBUG=1` are missed. A missed command is a check that does not run; a false one
// is a warning about something that never existed. Shell commands and their flags are
// conventionally lowercase, so the loss is small and the class it removes is large.
func hasUpper(span string) bool {
	return strings.ToLower(span) != span
}

// Commands reports the shell commands a document claims its repository supports.
//
// Requires: spans are inline code-span contents, as markdown.Doc.CodeSpans collects
// them. Passing arbitrary strings is not an error and will simply find fewer.
// Ensures: deduplicated and sorted, so two runs over one document agree and a caller
// can diff two documents' claims. Empty rather than nil when nothing qualifies. Pure —
// no filesystem, no `$PATH`, no execution.
//
// A span qualifies when it contains whitespace and its first token looks like an
// executable name. The whitespace requirement is what excludes the large majority of
// code spans, which are identifiers, filenames, and flags quoted inline.
//
// **Both error directions are real and the choice between them is deliberate.**
// `make verify` and `./scripts/build.sh --fast` are found. Not found: `SKILL.md` and
// `go` (no arguments), `foo.Bar()` (not an executable name), `x = 1` (an expression),
// `var Doc struct` (a declaration — see hasUpper), and `git push origin HEAD` (a real
// command, lost to the same rule). What still slips through is a code span holding
// lowercase prose — `the make verify step` — which reports a command named "the".
//
// A missed command is a check that never runs, silently; a false one is a warning
// naming something obviously wrong, which a reader discards in a second. The heuristic
// errs toward finding fewer, and the consumer's finding is advisory for that reason.
func Commands(spans []string) []string {
	seen := make(map[string]bool, len(spans))
	out := make([]string, 0, len(spans))
	for _, span := range spans {
		cmd := strings.TrimSpace(span)
		if !isInvocation(cmd) || seen[cmd] {
			continue
		}
		seen[cmd] = true
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out
}

// isInvocation reports whether a code span reads as a command being run.
//
// The parenthesis rule came from measuring rather than from thinking: over 233 real
// skill bodies the head of the false-positive list was `defer cancel()`, `go func()`,
// and `getenv func(string) string` — Go statements whose *first* token is a lowercase
// word, which is all the executable pattern examines. A command's arguments are paths,
// flags, and words; they are not call expressions.
func isInvocation(span string) bool {
	fields := strings.Fields(span)
	if len(fields) < 2 || hasUpper(span) || !executable.MatchString(fields[0]) {
		return false
	}
	for _, f := range fields[1:] {
		if isOperator(f) || strings.ContainsAny(f, "()") {
			return false
		}
	}
	return true
}
