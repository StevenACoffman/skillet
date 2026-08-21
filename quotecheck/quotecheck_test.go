package quotecheck_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/quotecheck"
)

// Fixtures are deliberately wordy: a passage shorter than MinPassageWords is dropped before
// matching, so a terse fixture would exercise nothing at all.

const (
	// faithful appears verbatim in the source below.
	faithful = "The cache is enabled by default and holds entries for one hour."
	// fabricated does not, and is the failure this package exists to catch.
	fabricated = "The cache is replicated across every region without configuration."
)

func book() quotecheck.Source {
	return quotecheck.Source{
		Name: "manual.txt",
		Text: "Chapter 3. " + faithful + " Operators may tune it per deployment.",
	}
}

func TestFindsAndMisses(t *testing.T) {
	t.Parallel()

	got := quotecheck.Check([]string{faithful, fabricated}, []quotecheck.Source{book()})
	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2: %+v", len(got), got)
	}
	if got[0].Status != quotecheck.Found {
		t.Errorf("faithful passage status = %v, want found", got[0].Status)
	}
	if got[0].FoundIn != "manual.txt" {
		t.Errorf("FoundIn = %q, want the source name", got[0].FoundIn)
	}
	if got[1].Status != quotecheck.Missing {
		t.Errorf("fabricated passage status = %v, want missing", got[1].Status)
	}
	if got[1].FoundIn != "" {
		t.Errorf("a missing passage names a source: %q", got[1].FoundIn)
	}
}

// TestFoldingBridgesTypographyAndWrapping is the mechanical latitude the guard allows, and
// the whole of it. A quotation is line-wrapped in Markdown and its source is not, so a
// literal comparison would report every faithful quotation missing.
func TestFoldingBridgesTypographyAndWrapping(t *testing.T) {
	t.Parallel()

	wrapped := "The cache is enabled\n   by default and holds\nentries for one hour."
	curly := strings.ReplaceAll(faithful, "'", "’")

	for name, quote := range map[string]string{
		"line wrapping": wrapped,
		"typography":    curly,
	} {
		got := quotecheck.Check([]string{quote}, []quotecheck.Source{book()})
		if len(got) != 1 || got[0].Status != quotecheck.Found {
			t.Errorf("%s: %+v, want one found passage", name, got)
		}
	}
}

// TestNoSourcesIsUncheckedNotMissing is half of why Status has three values.
//
// Reporting "missing" here would be a fabrication verdict reached without looking at
// anything, which is the worst possible output: it is indistinguishable from a real finding
// and it is always wrong.
func TestNoSourcesIsUncheckedNotMissing(t *testing.T) {
	t.Parallel()

	got := quotecheck.Check([]string{faithful}, nil)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Status != quotecheck.Unchecked {
		t.Errorf("status = %v, want unchecked", got[0].Status)
	}
	if got[0].Missing() {
		t.Error("Missing() is true for a passage nobody searched for")
	}
}

// TestShortQuotationIsReportedNotDropped is the other half, and the more dangerous one.
//
// Every passage of this quotation falls below MinPassageWords, so there is nothing to match.
// Emitting no finding would make it vanish: a caller counting findings would see a clean
// pass over a quotation that was never checked. The silence is the bug, not the verdict.
func TestShortQuotationIsReportedNotDropped(t *testing.T) {
	t.Parallel()

	short := "Too brief. Also brief."
	if len(quotecheck.Passages(short)) != 0 {
		t.Fatal("fixture is not short enough to exercise the case")
	}

	got := quotecheck.Check([]string{short}, []quotecheck.Source{book()})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1 — a short quotation must not vanish", len(got))
	}
	if got[0].Status != quotecheck.Unchecked {
		t.Errorf("status = %v, want unchecked", got[0].Status)
	}
	if got[0].Passage != short {
		t.Errorf("Passage = %q, want the whole quotation", got[0].Passage)
	}
}

// TestUncheckedIsTheZeroValue pins the choice a future edit is most likely to undo. A Status
// that defaulted to Found or Missing would launder "nobody looked" into a verdict.
func TestUncheckedIsTheZeroValue(t *testing.T) {
	t.Parallel()

	var zero quotecheck.Status
	if zero != quotecheck.Unchecked {
		t.Errorf("zero Status = %v, want unchecked", zero)
	}
	var f quotecheck.Finding
	if f.Missing() {
		t.Error("a zero Finding reports Missing; the zero value must assert nothing")
	}
}

// TestPassagesFiltersAndSplits covers the split rule, including the over-splitting that is
// deliberate: "e.g." becomes fragments too short to survive the filter, which is the
// intended outcome rather than a defect worked around.
func TestPassagesFiltersAndSplits(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		in   string
		want int
	}{
		"two long sentences": {faithful + " " + fabricated, 2},
		"short dropped":      {"Too brief.", 0},
		"abbreviation over-splits into fragments below the filter": {"See e.g. this.", 0},
		"empty": {"", 0},
	}
	for name, tc := range cases {
		if got := len(quotecheck.Passages(tc.in)); got != tc.want {
			t.Errorf("%s: got %d passages, want %d", name, got, tc.want)
		}
	}
}

// TestSupportCountsOnlyLocated: an unchecked passage is not support. Counting it would let a
// document with no sources at all report the same support as one whose quotes all validated.
func TestSupportCountsOnlyLocated(t *testing.T) {
	t.Parallel()

	findings := []quotecheck.Finding{
		{Status: quotecheck.Found},
		{Status: quotecheck.Missing},
		{Status: quotecheck.Unchecked},
	}
	if got := quotecheck.Support(findings); got != 1 {
		t.Errorf("Support = %d, want 1", got)
	}
	if got := quotecheck.Support(nil); got != 0 {
		t.Errorf("Support(nil) = %d, want 0", got)
	}
}

// TestCheckIsDeterministic: the guard sits under a gate, so two runs over one input must
// agree. Map iteration order is the usual way this breaks.
func TestCheckIsDeterministic(t *testing.T) {
	t.Parallel()

	quotes := []string{faithful, fabricated, "Too brief."}
	sources := []quotecheck.Source{book(), {Name: "other.txt", Text: "unrelated prose here"}}

	first := quotecheck.Check(quotes, sources)
	for range 20 {
		again := quotecheck.Check(quotes, sources)
		if len(again) != len(first) {
			t.Fatalf("length varies: %d then %d", len(first), len(again))
		}
		for i := range first {
			if again[i] != first[i] {
				t.Fatalf("finding %d varies:\n%+v\n%+v", i, first[i], again[i])
			}
		}
	}
}

// TestResultIsNeverNil: callers range over the result and count it, and a nil result that
// is sometimes a slice is the kind of inconsistency that gets papered over at each call
// site rather than fixed once here.
func TestResultIsNeverNil(t *testing.T) {
	t.Parallel()

	for name, quotes := range map[string][]string{
		"nil quotes":   nil,
		"empty quotes": {},
	} {
		if got := quotecheck.Check(quotes, []quotecheck.Source{book()}); got == nil {
			t.Errorf("%s: Check returned nil", name)
		}
	}
}
