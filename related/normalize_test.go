package related_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/related"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in          string
		want        string
		wantChanged bool
	}{
		"a lowercase heading on disk is canonicalised": {
			in:          "## Related skills\n\n- depends-on: `alpha` — because\n",
			want:        "## Related Skills\n\n- depends-on: `alpha` — because\n",
			wantChanged: true,
		},
		"canonical section is already normal": {
			in:   "## Related Skills\n\n- depends-on: `alpha` — because\n",
			want: "## Related Skills\n\n- depends-on: `alpha` — because\n",
		},
		"bold kind with linked target becomes canonical": {
			in: "## Related Skills\n\n" +
				"- **composes-with** → [`alpha`](../alpha/SKILL.md): because\n",
			want:        "## Related Skills\n\n- composes-with: `alpha` — because\n",
			wantChanged: true,
		},
		"reversed form becomes canonical": {
			in:          "## Related Skills\n\n- **alpha** (contrasts-with): because\n",
			want:        "## Related Skills\n\n- contrasts-with: `alpha` — because\n",
			wantChanged: true,
		},
		"bare token becomes canonical": {
			in:          "## Related Skills\n\n- depends-on: alpha (because things)\n",
			want:        "## Related Skills\n\n- depends-on: `alpha` — (because things)\n",
			wantChanged: true,
		},
		// The 9-continuation-line risk: a wrapped rationale must survive whole.
		"wrapped rationale is folded, not truncated": {
			in: "## Related Skills\n\n" +
				"- **composes-with** [`alpha`](../alpha/SKILL.md): first part\n" +
				"  second part continues here\n",
			want:        "## Related Skills\n\n- composes-with: `alpha` — first part second part continues here\n",
			wantChanged: true,
		},
		// The 5-prose-bullet risk: a bullet naming no skill must not be deleted.
		"prose bullet is preserved verbatim": {
			in: "## Related Skills\n\n" +
				"- contrasts-with: (traditional headcount-scaling model)\n" +
				"- depends-on: `alpha` — because\n",
			want: "## Related Skills\n\n" +
				"- contrasts-with: (traditional headcount-scaling model)\n" +
				"- depends-on: `alpha` — because\n",
			// Nothing changes: the prose bullet is copied through and the other
			// bullet is already canonical.
			wantChanged: false,
		},
		"multi-target becomes one bullet per target": {
			in: "## Related Skills\n\n- composes-with: `alpha`, `beta`\n",
			want: "## Related Skills\n\n" +
				"- composes-with: `alpha` — \n" +
				"- composes-with: `beta` — \n",
			wantChanged: true,
		},
		"duplicate relationship collapses when the words are the same": {
			in: "## Related Skills\n\n" +
				"- **composes-with** [`alpha`](../alpha/SKILL.md): the one wording\n" +
				"- composes-with: `alpha` — the one wording\n",
			want:        "## Related Skills\n\n- composes-with: `alpha` — the one wording\n",
			wantChanged: true,
		},
		// The same collapse would delete an explanation, so it does not happen: the
		// second bullet stays exactly as written, legacy form and all.
		"duplicate relationship in other words is kept": {
			in: "## Related Skills\n\n" +
				"- composes-with: `alpha` — the first explanation\n" +
				"- **composes-with** [`alpha`](../alpha/SKILL.md): a different explanation\n",
			want: "## Related Skills\n\n" +
				"- composes-with: `alpha` — the first explanation\n" +
				"- **composes-with** [`alpha`](../alpha/SKILL.md): a different explanation\n",
			wantChanged: false,
		},
		// One restated target is enough to keep the whole line: splitting it would
		// write the new target canonically and leave the restated one nowhere.
		"multi-target bullet restating one of its targets is kept whole": {
			in: "## Related Skills\n\n" +
				"- depends-on: `alpha` — the first reason\n" +
				"- depends-on: `alpha`, `beta` — a different reason\n",
			want: "## Related Skills\n\n" +
				"- depends-on: `alpha` — the first reason\n" +
				"- depends-on: `alpha`, `beta` — a different reason\n",
			wantChanged: false,
		},
		// A restatement carrying no rationale has no words to lose, so it still goes.
		"duplicate relationship with no rationale collapses": {
			in: "## Related Skills\n\n" +
				"- composes-with: `alpha` — the only explanation\n" +
				"- composes-with: `alpha`\n",
			want:        "## Related Skills\n\n- composes-with: `alpha` — the only explanation\n",
			wantChanged: true,
		},
		"suffixed heading becomes canonical": {
			in:          "## Related skills (Stage 3 Filling)\n\n- depends-on: `alpha` — because\n",
			want:        "## Related Skills\n\n- depends-on: `alpha` — because\n",
			wantChanged: true,
		},
		"content outside the section is untouched": {
			in: "---\nname: s\n---\n\n# Body\n\nProse here.\n\n" +
				"## Related Skills\n\n- **alpha** (depends-on): why\n\n---\n\nTail prose.\n",
			want: "---\nname: s\n---\n\n# Body\n\nProse here.\n\n" +
				"## Related Skills\n\n- depends-on: `alpha` — why\n\n---\n\nTail prose.\n",
			wantChanged: true,
		},
		"intro sentence inside the section is kept": {
			in: "## Related Skills\n\nThese are the related skills:\n\n" +
				"- **alpha** (depends-on): why\n",
			want: "## Related Skills\n\nThese are the related skills:\n\n" +
				"- depends-on: `alpha` — why\n",
			wantChanged: true,
		},
		"no section is a no-op": {
			in:   "# Body\n\nNothing here.\n",
			want: "# Body\n\nNothing here.\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, changed := related.Normalize(tc.in)
			if got != tc.want {
				t.Errorf("Normalize mismatch\ngot:\n%s\nwant:\n%s", got, tc.want)
			}
			if changed != tc.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}
		})
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	t.Parallel()
	// Every dialect at once, so a second pass has plenty to be unstable about.
	in := "# Body\n\n## Related skills (Stage 3 Filling)\n\n" +
		"- **composes-with** → [`alpha`](../alpha/SKILL.md): first\n" +
		"  wrapped tail\n" +
		"- **beta** (depends-on): second\n" +
		"- contrasts-with: (prose, not a skill)\n" +
		"- depends-on: gamma (third)\n" +
		"- composes-with: `delta`, `epsilon`\n"

	once, changed := related.Normalize(in)
	if !changed {
		t.Fatal("first pass must change a legacy section")
	}
	twice, changedAgain := related.Normalize(once)
	if changedAgain {
		t.Errorf("second pass must be a no-op, got:\n%s", twice)
	}
	if twice != once {
		t.Errorf("not idempotent\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
}

func TestNormalizePreservesTheEdgeSet(t *testing.T) {
	t.Parallel()
	// The safety net for the migration: normalizing must not change which edges the
	// section expresses, only how they are written.
	in := "## Related skills (Stage 3 Filling)\n\n" +
		"- **composes-with** → [`alpha`](../alpha/SKILL.md): first\n" +
		"  wrapped tail\n" +
		"- **beta** (depends-on): second\n" +
		"- contrasts-with: (prose, not a skill)\n" +
		"- depends-on: gamma (third)\n" +
		"- composes-with: `delta`, `epsilon`\n"

	before := related.ParseSection(in)
	out, _ := related.Normalize(in)
	after := related.ParseSection(out)

	if len(before) != len(after) {
		t.Fatalf("edge count changed: %d -> %d\nbefore: %+v\nafter: %+v",
			len(before), len(after), before, after)
	}
	for i := range before {
		if before[i].Kind != after[i].Kind || before[i].Target != after[i].Target {
			t.Errorf("edge %d changed: %+v -> %+v", i, before[i], after[i])
		}
	}
}

func TestNormalizeMergesASecondSection(t *testing.T) {
	t.Parallel()
	// The shape of all 13 two-section skills in the real books.
	in := "# Body\n\n## Related skills (Stage 3 Filling)\n\n" +
		"- depends-on: `alpha` — first\n" +
		"- contrasts-with: (prose, not a skill)\n\n" +
		"---\n\n" +
		"## Related Skills\n\n" +
		"- composes-with: `beta` — second\n" +
		"- depends-on: `alpha` — restated in other words\n\n" +
		"---\n\n" +
		"## Audit Information\n"
	// The restatement moves with the rest: it explains the same relationship in other
	// words, and those words are on no other line.
	want := "# Body\n\n## Related Skills\n\n" +
		"- depends-on: `alpha` — first\n" +
		"- contrasts-with: (prose, not a skill)\n" +
		"- composes-with: `beta` — second\n" +
		"- depends-on: `alpha` — restated in other words\n\n" +
		"---\n\n" +
		"## Audit Information\n"

	got, changed := related.Normalize(in)
	if !changed {
		t.Fatal("expected a two-section file to change")
	}
	if got != want {
		t.Errorf("merge mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
	if again, changedAgain := related.Normalize(got); changedAgain || again != got {
		t.Errorf("merging must be idempotent, second pass:\n%s", again)
	}
}

// TestNormalizeMergeKeepsEveryEdgeAndEveryRationale pins the trade this makes: merging
// two sections collapses the *headings*, never the explanations. Both statements of one
// relationship survive; the reader still reports one edge for them.
func TestNormalizeMergeKeepsEveryEdgeAndEveryRationale(t *testing.T) {
	t.Parallel()
	in := "## Related Skills\n\n- depends-on: `alpha` — the reason that was written first\n\n" +
		"## Related Skills\n\n- depends-on: `alpha` — a later, different reason\n" +
		"- composes-with: `beta` — only in the second section\n"

	out, _ := related.Normalize(in)
	if strings.Count(out, "## Related Skills") != 1 {
		t.Errorf("expected one section after merging, got:\n%s", out)
	}
	for _, want := range []string{
		"the reason that was written first",
		"a later, different reason",
		"- composes-with: `beta` — only in the second section",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q was lost in the merge, got:\n%s", want, out)
		}
	}
	// Two bullets, one relationship: the duplication is the document's, and the reader
	// is what resolves it.
	if got := len(related.ParseSection(out)); got != 2 {
		t.Errorf("edges = %d, want 2 (alpha and beta):\n%s", got, out)
	}
}

func TestNormalizeLeavesASecondSectionHoldingProseWhereItIs(t *testing.T) {
	t.Parallel()
	// Merging would have to delete the paragraph or move it somewhere it does not
	// belong. Leaving the section in place is the smaller harm, and it is still
	// canonicalised where it can be.
	in := "## Related Skills\n\n- depends-on: `alpha` — first\n\n" +
		"## Related skills (Stage 3 Filling)\n\n" +
		"These edges were added by hand and still need review.\n\n" +
		"- **composes-with** → [`beta`](../beta/SKILL.md): second\n"

	out, _ := related.Normalize(in)
	if strings.Count(out, "## Related Skills") != 2 {
		t.Errorf("a section holding prose must not be merged away, got:\n%s", out)
	}
	if !strings.Contains(out, "These edges were added by hand and still need review.") {
		t.Errorf("prose was lost:\n%s", out)
	}
	if !strings.Contains(out, "- composes-with: `beta` — second") {
		t.Errorf("the unmerged section was not canonicalised:\n%s", out)
	}
	if again, changedAgain := related.Normalize(out); changedAgain || again != out {
		t.Errorf("the unmerged case must still be idempotent, second pass:\n%s", again)
	}
}

func TestNormalizeMergeMovesABulletItCannotParse(t *testing.T) {
	t.Parallel()
	// A bullet the reader cannot understand carries the only rationale anyone wrote, so
	// it moves verbatim rather than being dropped with its heading.
	//
	// The fixture used to be "**depends_on**", which was unparseable until canonicalKind
	// learned to read an underscore as a hyphen. It is now a kind outside the vocabulary
	// instead — the property under test is "unparsed moves", and it needs a bullet that
	// is actually unparsed to test it.
	in := "## Related Skills\n\n- depends-on: `alpha` — first\n\n" +
		"## Related Skills\n\n- **teleports-to**: alpha — a kind nobody defined\n"

	out, _ := related.Normalize(in)
	if strings.Count(out, "## Related Skills") != 1 {
		t.Errorf("expected one section after merging, got:\n%s", out)
	}
	if !strings.Contains(out, "- **teleports-to**: alpha — a kind nobody defined") {
		t.Errorf("an unparsed bullet was dropped instead of moved:\n%s", out)
	}
}

// TestNormalizeReadsTheUnderscoreDialect is the other half of the change above: the
// underscore spelling now parses, so a section repeating an edge in that dialect collapses
// to one canonical bullet rather than keeping both.
//
// Both bullets carry the same rationale, which is what makes the collapse free. Told in
// different words the second bullet would be kept — see
// TestNormalizeKeepsARestatementInOtherWords, which is the case the real corpus has.
func TestNormalizeReadsTheUnderscoreDialect(t *testing.T) {
	t.Parallel()
	in := "## Related Skills\n\n- depends-on: `alpha` — first\n\n" +
		"## Related Skills\n\n- **depends_on**: alpha — first\n"

	out, _ := related.Normalize(in)
	if strings.Contains(out, "depends_on") {
		t.Errorf("the underscore spelling survived instead of being read:\n%s", out)
	}
	if got := strings.Count(out, "- depends-on: `alpha`"); got != 1 {
		t.Errorf("canonical depends-on bullets = %d, want the duplicate collapsed:\n%s",
			got, out)
	}
}

// TestNormalizeKeepsARestatementInOtherWords is the shape 7 skills in the real market
// corpus carry: two related-skills sections, separated by the underscore thematic break
// mdformat writes, whose second section states the same five relationships as the first
// in different and longer words.
//
// Measured before this was fixed: normalizing the corpus deleted 27 such bullets across
// those 7 skills and left each holding an empty heading, while INDEX.md stayed
// byte-identical — the graph gained nothing and the documents lost paragraphs.
func TestNormalizeKeepsARestatementInOtherWords(t *testing.T) {
	t.Parallel()
	in := "# Body\n\n## Related Skills\n\n" +
		"- depends-on: `alpha` — the short reason\n\n" +
		"______________________________________________________________________\n\n" +
		"## Related Skills\n\n" +
		"- **depends_on**: alpha — the longer reason, which is the only place " +
		"anyone explained why\n\n" +
		"______________________________________________________________________\n\n" +
		"## Audit Information\n"

	out, _ := related.Normalize(in)
	if !strings.Contains(out, "the longer reason, which is the only place anyone explained why") {
		t.Errorf("a rationale nobody else wrote was deleted:\n%s", out)
	}
	if !strings.Contains(out, "the short reason") {
		t.Errorf("the first rationale was lost:\n%s", out)
	}
	// The underscore break is a thematic break, so the second section merges rather than
	// being emptied where it stands.
	if got := strings.Count(out, "## Related Skills"); got != 1 {
		t.Errorf("sections = %d, want the second merged into the first:\n%s", got, out)
	}
	if strings.Contains(out, "## Related Skills\n\n\n") {
		t.Errorf("an emptied section was left behind:\n%s", out)
	}
	// The reader still sees one edge: keeping the words does not double the graph.
	if got := len(related.ParseSection(out)); got != 1 {
		t.Errorf("edges = %d, want 1:\n%s", got, out)
	}
	if again, changed := related.Normalize(out); changed || again != out {
		t.Errorf("must be idempotent, second pass:\n%s", again)
	}
}
