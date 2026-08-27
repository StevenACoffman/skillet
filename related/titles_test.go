package related_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/related"
)

func tree() []related.Node {
	return []related.Node{
		{Slug: "alpha", Heading: "Alpha Skill"},
		{Slug: "beta", Heading: "Beta Skill"},
		// Two skills sharing a heading. Measured on a real corpus: 232 distinct
		// headings across 233 skills, so this is not hypothetical.
		{Slug: "gamma-one", Heading: "Shared Heading"},
		{Slug: "gamma-two", Heading: "Shared Heading"},
		{Slug: "no-heading"},
	}
}

// wantResolve asserts a title's resolution and slug together, because the pair is the
// contract: a slug without TitleResolved would be written into an edge by a caller that
// only checked one of them.
func wantResolve(t *testing.T, title, wantSlug string, wantRes related.Resolution) {
	t.Helper()
	slug, res := related.NewTitles(tree()).Resolve(title)
	if res != wantRes {
		t.Errorf("Resolve(%q) resolution = %v, want %v", title, res, wantRes)
	}
	if slug != wantSlug {
		t.Errorf("Resolve(%q) slug = %q, want %q", title, slug, wantSlug)
	}
}

func TestResolveTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		title string
		slug  string
		res   related.Resolution
	}{
		{"an exact heading resolves", "Alpha Skill", "alpha", related.TitleResolved},
		{"surrounding space is ignored", "  Beta Skill ", "beta", related.TitleResolved},
		// Refusing is the point: picking either would be a coin flip recorded as fact.
		{
			"a shared heading is ambiguous and yields no slug",
			"Shared Heading",
			"",
			related.TitleAmbiguous,
		},
		{"an unknown title yields no slug", "Nothing Like This", "", related.TitleUnknown},
		// Exact only. Slugifying would return "alpha" here and be wrong as often as right.
		{"a near miss does not resolve", "Alpha Skil", "", related.TitleUnknown},
		{"case is not folded", "alpha skill", "", related.TitleUnknown},
		{"a skill with no heading is not indexed", "", "", related.TitleUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wantResolve(t, tt.title, tt.slug, tt.res)
		})
	}
}

// TestTitleRefsSkipsWhatIsNotATitle. A bold token that is already a slug, or a kind in
// the bold position, is a dialect problem rather than a lookup one and must not be
// reported here — reporting it would inflate the count the dialect work is measured by.
func TestTitleRefsSkipsWhatIsNotATitle(t *testing.T) {
	t.Parallel()
	nodes := append(tree(), related.Node{
		Slug:    "caller",
		Heading: "Caller",
		Body: "## Related Skills\n\n" +
			"- **Alpha Skill** — *depends-on* → a title\n" +
			"- **alpha** — *depends-on* → already a slug\n" +
			"- **composes_with**: beta — a kind in the bold slot\n" +
			"- composes-with: `beta` — canonical, not bold\n",
	})
	refs := related.TitleRefs(nodes)
	if len(refs) != 1 {
		t.Fatalf("refs = %+v, want only the display-title bullet", refs)
	}
	if refs[0].Slug != "alpha" || refs[0].Resolution != related.TitleResolved {
		t.Errorf("ref = %+v, want alpha resolved", refs[0])
	}
}

// TestInlineSlugNeedsTheTargetInTree. A title carrying its own slug resolves only when
// that slug is a skill here; a cross-tree reference has no answer, and inventing one
// would be the wrong-match failure by another route.
func TestInlineSlugNeedsTheTargetInTree(t *testing.T) {
	t.Parallel()
	body := func(title string) string {
		return "## Related Skills\n\n- **" + title + "** — *depends-on* → why\n"
	}
	nodes := append(tree(), related.Node{
		Slug: "caller", Heading: "Caller",
		Body: body("Some Title (`alpha`)") + body("Other Title (`elsewhere`)"),
	})
	refs := related.TitleRefs(nodes)
	got := map[string]related.Resolution{}
	for _, r := range refs {
		got[r.Slug] = r.Resolution
	}
	if got["alpha"] != related.TitleResolved {
		t.Errorf("an in-tree inline slug did not resolve: %+v", refs)
	}
	for _, r := range refs {
		if r.Title == "Other Title (`elsewhere`)" && r.Resolution != related.TitleUnknown {
			t.Errorf("a cross-tree inline slug resolved to %q; it has no answer here", r.Slug)
		}
	}
}

// wantResolveTitles rewrites one body and asserts the result and the count together: a
// count without the text would pass on a rewrite that changed the wrong thing.
func wantResolveTitles(t *testing.T, body, want string, wantN int) {
	t.Helper()
	nodes := tree()
	known := map[string]bool{}
	for i := range nodes {
		known[nodes[i].Slug] = true
	}
	got, n := related.ResolveTitles(body, related.NewTitles(nodes), known)
	if n != wantN {
		t.Errorf("changed = %d, want %d", n, wantN)
	}
	if got != want {
		t.Errorf("body =\n%s\nwant\n%s", got, want)
	}
}

func TestResolveTitlesRewritesOnlyWhatItIsSureOf(t *testing.T) {
	t.Parallel()
	head := "## Related Skills\n\n"
	tests := []struct {
		name       string
		body, want string
		n          int
	}{{
		name: "an exact title becomes its slug, the rest of the bullet untouched",
		body: head + "- **Alpha Skill** — *depends-on* → why\n",
		want: head + "- **alpha** — *depends-on* → why\n", n: 1,
	}, {
		// Both are refusals for the same reason: a wrong rewrite becomes an edge
		// nobody can distinguish from an authored one.
		name: "an ambiguous title is left byte-identical",
		body: head + "- **Shared Heading** — *depends-on* → why\n",
		want: head + "- **Shared Heading** — *depends-on* → why\n", n: 0,
	}, {
		name: "an unknown title is left byte-identical",
		body: head + "- **Nothing Like This** — *depends-on* → why\n",
		want: head + "- **Nothing Like This** — *depends-on* → why\n", n: 0,
	}, {
		name: "a bullet already naming a slug is not touched",
		body: head + "- **alpha** — *depends-on* → why\n",
		want: head + "- **alpha** — *depends-on* → why\n", n: 0,
	}, {
		name: "a kind in the bold position is not a title",
		body: head + "- **composes_with**: alpha — why\n",
		want: head + "- **composes_with**: alpha — why\n", n: 0,
	}, {
		// Prose outside the section must never be rewritten, however it reads.
		name: "text outside a related section is untouched",
		body: "# Doc\n\n- **Alpha Skill** — a sentence about it\n",
		want: "# Doc\n\n- **Alpha Skill** — a sentence about it\n", n: 0,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wantResolveTitles(t, tt.body, tt.want, tt.n)
		})
	}
}

// TestResolveTitlesThenNormalizeYieldsACanonicalBullet is the pair that matters: the
// substitution is only useful because the reader can then understand the bullet, which
// it could not before the dialect tolerances landed.
func TestResolveTitlesThenNormalizeYieldsACanonicalBullet(t *testing.T) {
	t.Parallel()
	nodes := tree()
	known := map[string]bool{}
	for i := range nodes {
		known[nodes[i].Slug] = true
	}
	body := "## Related Skills\n\n- **Alpha Skill** — *depends-on* → because.\n"
	resolved, n := related.ResolveTitles(body, related.NewTitles(nodes), known)
	if n != 1 {
		t.Fatalf("changed = %d, want 1", n)
	}
	out, _ := related.Normalize(resolved)
	if !strings.Contains(out, "- depends-on: `alpha` — because.") {
		t.Errorf("normalized =\n%s\nwant a canonical depends-on bullet", out)
	}
}
