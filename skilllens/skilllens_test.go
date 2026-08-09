package skilllens_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/markdown"
	"github.com/StevenACoffman/skillet/skilllens"
)

// count returns how many spans of the given kind are present.
func count(spans []skilllens.Span, kind skilllens.Kind) int {
	n := 0
	for _, s := range spans {
		if s.Kind == kind {
			n++
		}
	}
	return n
}

func TestFailureMechanismsFindsInlineBranches(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		body string
		want int
	}{
		"english if": {"If the API call fails, re-query the page.", 1},
		"english when": {
			"When the record is missing, fall back to the cache.",
			1,
		},
		"chinese":                           {"如果接口超时，重试一次。", 1},
		"both languages at once":            {"If it fails, stop.\n\n如果接口超时，重试。", 2},
		"generic advice is not a mechanism": {"Handle errors carefully and be thorough.", 0},
		"no conditional":                    {"Re-query the page after each write.", 0},
		// The vocabulary is literal, not lemmatised: "timeout" is listed, "times out"
		// is not, so this reads as no mechanism. Recorded because it is a real limit of
		// the detector rather than a property of this test.
		"an unlisted inflection is missed": {"If the API times out, re-query.", 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := count(skilllens.FailureMechanisms(markdown.Parse(tc.body)), skilllens.KindProse)
			if got != tc.want {
				t.Errorf("got %d inline branches, want %d", got, tc.want)
			}
		})
	}
}

func TestFailureMechanismsFindsSections(t *testing.T) {
	t.Parallel()
	// dim 3 counts a dedicated section as evidence too, so one call must return both
	// kinds and the caller must be able to tell them apart.
	body := "## Boundary\n\n- one\n- two\n\n## Steps\n\nIf it fails, stop.\n"
	spans := skilllens.FailureMechanisms(markdown.Parse(body))
	if n := count(spans, skilllens.KindSection); n != 1 {
		t.Errorf("got %d section spans, want 1: %+v", n, spans)
	}
	if n := count(spans, skilllens.KindProse); n != 1 {
		t.Errorf("got %d prose spans, want 1: %+v", n, spans)
	}
}

func TestSofteningPhrasesAreBilingual(t *testing.T) {
	t.Parallel()
	// A China-only vocabulary scores every English skill as defect-free, which is the
	// opposite of useful — so both halves must be live.
	cases := map[string]int{
		"Use it as appropriate, and feel free to adapt.": 2,
		"根据情况灵活把握。":                                      2,
		"Do exactly this, then stop.":                    0,
		// skillsaw lowercases both sides before matching, so a capitalised phrase counts.
		// Losing that would quietly stop scoring every sentence-initial hedge.
		"As appropriate, adapt it. Feel free.": 2,
	}
	for body, want := range cases {
		t.Run(body[:min(24, len(body))], func(t *testing.T) {
			t.Parallel()
			got := skilllens.SofteningPhrases(markdown.Parse(body))
			if len(got) != want {
				t.Errorf("got %d softening phrases, want %d: %+v", len(got), want, got)
			}
		})
	}
}

func TestSectionMatchingHandlesPluralAndWordBoundary(t *testing.T) {
	t.Parallel()
	// The standard book2skill B segment is titled "Boundaries" (plural); "boundary" must
	// match it via its ies-plural, which a plain substring test misses ("boundary" is not
	// a substring of "boundaries"). This is the case that moved ~30 skills' scores when
	// skilllens used plain Contains where skillsaw's rubric used an inflecting matcher.
	plural := markdown.Parse("## B — Boundaries and Blind Spots\n\n- don't do this\n- or this\n")
	if n := count(skilllens.BlacklistSections(plural), skilllens.KindSection); n != 1 {
		t.Errorf("plural 'Boundaries' heading: BlacklistSections = %d, want 1", n)
	}
	if n := count(skilllens.FailureMechanisms(plural), skilllens.KindSection); n != 1 {
		t.Errorf("plural 'Boundaries' heading: FailureMechanisms section = %d, want 1", n)
	}
	// An ASCII term must begin at a word boundary: "red flag" must not match the "red" in
	// "Required" followed by the "flag" in "flags".
	midword := markdown.Parse("## Required Flags Reference\n\n- a\n- b\n")
	if n := count(skilllens.BlacklistSections(midword), skilllens.KindSection); n != 0 {
		t.Errorf("'Required Flags' must not match 'red flag' mid-word: got %d", n)
	}
}

func TestBlacklistSectionsCarryTheirUnits(t *testing.T) {
	t.Parallel()
	// The substance threshold is the caller's policy, so the count travels with the span
	// rather than being applied here.
	body := "## Anti-patterns\n\n- do not do this\n- or this\n- or this\n\n## Steps\n\n- go\n"
	spans := skilllens.BlacklistSections(markdown.Parse(body))
	if len(spans) != 1 {
		t.Fatalf("want one blacklist section, got %+v", spans)
	}
	if spans[0].Kind != skilllens.KindSection {
		t.Errorf("kind = %q, want a section", spans[0].Kind)
	}
	if spans[0].Units < 3 {
		t.Errorf("Units = %d, want the section's 3 items so a caller can threshold on it",
			spans[0].Units)
	}
}

func TestDetectorsIgnoreCodeBlocks(t *testing.T) {
	t.Parallel()
	// They read Doc.Prose, where markdown has blanked code. A conditional inside a shell
	// transcript is not the skill's own instruction, and a "# Boundary" comment in a
	// snippet is not a section.
	body := "## Steps\n\n```sh\nif curl fails; then retry; fi\n# it depends\n```\n\nJust do it.\n"
	d := markdown.Parse(body)
	if n := count(skilllens.FailureMechanisms(d), skilllens.KindProse); n != 0 {
		t.Errorf("matched %d branches inside a code block", n)
	}
	if got := skilllens.SofteningPhrases(d); len(got) != 0 {
		t.Errorf("matched softening inside a code block: %+v", got)
	}
}

func TestVocabulariesAreFreshPerCall(t *testing.T) {
	t.Parallel()
	// An exported slice would be shared mutable state one caller could edit under all
	// the others, which is how a shared rubric silently diverges.
	for name, get := range map[string]func() []string{
		"FailureSectionTitles": skilllens.FailureSectionTitles,
		"SofteningTerms":       skilllens.SofteningTerms,
		"BlacklistTitles":      skilllens.BlacklistTitles,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			a := get()
			if len(a) == 0 {
				t.Fatal("vocabulary is empty")
			}
			a[0] = "clobbered"
			if get()[0] == "clobbered" {
				t.Error("callers share one slice: an edit here changes it for everyone")
			}
		})
	}
}

func TestVocabulariesKeepBothLanguages(t *testing.T) {
	t.Parallel()
	// Dropping half on the way up would silently widen what passes, which is the
	// failure the bilingual note in the source warns about.
	for name, terms := range map[string][]string{
		"FailureSectionTitles": skilllens.FailureSectionTitles(),
		"SofteningTerms":       skilllens.SofteningTerms(),
		"BlacklistTitles":      skilllens.BlacklistTitles(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var ascii, han int
			for _, term := range terms {
				if isASCII(term) {
					ascii++
				} else {
					han++
				}
			}
			if ascii == 0 || han == 0 {
				t.Errorf("%s has %d ASCII and %d non-ASCII terms; both halves must survive",
					name, ascii, han)
			}
		})
	}
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func TestEmptyDocument(t *testing.T) {
	t.Parallel()
	d := markdown.Parse("")
	for name, got := range map[string][]skilllens.Span{
		"FailureMechanisms": skilllens.FailureMechanisms(d),
		"SofteningPhrases":  skilllens.SofteningPhrases(d),
		"BlacklistSections": skilllens.BlacklistSections(d),
	} {
		if len(got) != 0 {
			t.Errorf("%s on an empty document returned %+v", name, got)
		}
	}
	_ = strings.TrimSpace("")
}
