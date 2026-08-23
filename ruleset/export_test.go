package ruleset

// This file is compiled only under `go test` and adds no exported surface as
// consumers see the package.
//
// The marker table is unexported because it is the parser's own vocabulary and no
// caller has a use for it. A test does: TestEveryMarkerIsNonASCII asserts the property
// the rejection rule depends on, and asserting it against a copy of the list would
// check the copy.

// MarkerPrefixesForTest returns the body-line prefixes the parser recognises.
func MarkerPrefixesForTest() []string {
	out := make([]string, 0, len(markers()))
	for _, m := range markers() {
		out = append(out, m.prefix)
	}
	return out
}
