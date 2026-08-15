package testprompts

import (
	"regexp"
	"strings"

	"github.com/StevenACoffman/skillet/judge"
)

// Derivation patterns. Each is intentionally conservative: a check is emitted
// only when the cue is unambiguous, so a derived set never fails judge on a
// guess. Ambiguous Expected text yields no checks (the caller must then
// hand-write them) rather than a wrong one.
var (
	// `"Result" section` -> section_present(Result).
	reSectionQuoted = regexp.MustCompile(`"([^"]+)"\s+section`)
	// A literal markdown heading inside Expected -> section_present(<heading>).
	reHeading = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
	// `"foo" tool` / `` `foo` tool `` -> tool_called(foo).
	reToolQuoted = regexp.MustCompile("[\"`]([^\"`]+)[\"`]\\s+tool")
	// `contains/includes/mentions "phrase"` -> contains(phrase).
	reContains = regexp.MustCompile(`(?i)(?:contains|includes|mentions|outputs)\s+"([^"]+)"`)
	// `<= N chars`, `under N characters`, `at most N characters` -> max_chars(N).
	reMaxChars = regexp.MustCompile(`(?i)(?:<=|≤|under|at most|no more than)\s+(\d+)\s+char`)
	// `>= N chars`, `at least N characters` -> min_chars(N).
	reMinChars = regexp.MustCompile(`(?i)(?:>=|≥|at least|no fewer than)\s+(\d+)\s+char`)
)

// DeriveChecks converts an Expected description into deterministic judge checks.
// Every returned Check is backed by an unambiguous cue in expected; it returns
// nil (not a wrong guess) when nothing is inferable, and the result is
// de-duplicated and stably ordered for identical input.
//
// Do not expect this to cover a corpus, and do not widen it to make it. Measured
// 2026-08-15 over 1284 real behavioral Expected strings: **none** carried any cue
// below, 1% held a backticked span, and 15% were the single word "invoke". Those
// strings are prose written for a human judge -- "a single generic decodeValid[T
// Validator] helper" appears with no markup at all -- so there is nothing to read.
//
// The tempting widening is a contains check over identifier-shaped tokens, which
// would reach about a fifth of that corpus. It is wrong for the reason stated above:
// a CamelCase word inside prose is not an assertion that the output must contain that
// literal string, so the check fails a correct answer. That lowers the base of the
// dimension it feeds, which is the heaviest in skillsaw's rubric. Returning nothing
// is the honest answer; a manufactured false negative is not.
func DeriveChecks(expected string) []judge.Check {
	var checks []judge.Check
	seen := map[judge.Check]bool{}
	add := func(op judge.Op, arg string) {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return
		}
		c := judge.Check{Op: op, Arg: arg}
		if !seen[c] {
			seen[c] = true
			checks = append(checks, c)
		}
	}
	// Fixed operator order (not text order) so output is stable.
	for _, m := range reSectionQuoted.FindAllStringSubmatch(expected, -1) {
		add(judge.OpSectionPresent, m[1])
	}
	for _, m := range reHeading.FindAllStringSubmatch(expected, -1) {
		add(judge.OpSectionPresent, m[1])
	}
	for _, m := range reToolQuoted.FindAllStringSubmatch(expected, -1) {
		add(judge.OpToolCalled, m[1])
	}
	for _, m := range reContains.FindAllStringSubmatch(expected, -1) {
		add(judge.OpContains, m[1])
	}
	for _, m := range reMaxChars.FindAllStringSubmatch(expected, -1) {
		add(judge.OpMaxChars, m[1])
	}
	for _, m := range reMinChars.FindAllStringSubmatch(expected, -1) {
		add(judge.OpMinChars, m[1])
	}
	return checks
}
