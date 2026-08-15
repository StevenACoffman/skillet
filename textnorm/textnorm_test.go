package textnorm_test

import (
	"testing"

	"github.com/StevenACoffman/skillet/textnorm"
)

func TestFold(t *testing.T) {
	t.Parallel()
	// The invisible characters are written as escapes for the reason the package doc gives:
	// a literal zero-width space is invisible, so nobody could tell what the case covers or
	// notice it being edited away. staticcheck proposed exactly that edit here -- one of its
	// two conflicting fixes rewrote the case to {"ab", "ab"}, which passes and tests nothing.
	cases := map[string]struct{ in, want string }{
		"whitespace runs collapse":  {"a   b\n\tc", "a b c"},
		"leading and trailing go":   {"  a b  ", "a b"},
		"curly single quote":        {"don\u2019t", "don't"},
		"curly double quotes":       {"\u201cq\u201d", `"q"`},
		"en and em dash":            {"a\u2013b\u2014c", "a-b-c"},
		"ellipsis expands":          {"a\u2026", "a..."},
		"non-breaking space folds":  {"a\u00a0b", "a b"},
		"zero-width space vanishes": {"a\u200bb", "ab"},
		"already folded is a no-op": {"a b c", "a b c"},
		"empty":                     {"", ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := textnorm.Fold(tc.in); got != tc.want {
				t.Errorf("Fold(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFoldPreservesCase pins the decision that separates this from a general "normalize":
// a quotation differing only in case is a different quotation, and folding case would widen
// every guard that compares one against its source.
func TestFoldPreservesCase(t *testing.T) {
	t.Parallel()
	if got := textnorm.Fold("The Thing"); got != "The Thing" {
		t.Errorf("Fold lowered case: %q", got)
	}
	if textnorm.Fold("abc") == textnorm.Fold("ABC") {
		t.Error("Fold treated two cases as equal")
	}
}

// TestFoldMakesTheCanonizerCaseAgree is the defect that moved this package into skillet:
// canonizer folded whitespace only, so an anchor copied from a source with a curly
// apostrophe failed there while the same passage passed exegesis quotecheck.
func TestFoldMakesTheCanonizerCaseAgree(t *testing.T) {
	t.Parallel()
	fromSource := "the agent’s   own words"
	fromRuleset := "the agent's own words"
	if textnorm.Fold(fromSource) != textnorm.Fold(fromRuleset) {
		t.Errorf("the two spellings still disagree: %q vs %q",
			textnorm.Fold(fromSource), textnorm.Fold(fromRuleset))
	}
}
