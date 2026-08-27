package related_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/related"
)

// TestParseSectionDialects covers every bullet dialect found in real skill trees,
// plus the three ways a first draft of the tolerant reader got it wrong.
func TestParseSectionDialects(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		bullet string
		want   []related.Edge
	}{
		"canonical": {
			bullet: "- depends-on: `alpha` — because",
			want: []related.Edge{
				{Kind: related.DependsOn, Target: "alpha", Rationale: "because"},
			},
		},
		"bold kind with arrow and linked backticked target": {
			bullet: "- **composes-with** → [`alpha`](../alpha/SKILL.md): because",
			want: []related.Edge{
				{Kind: related.ComposesWith, Target: "alpha", Rationale: "because"},
			},
		},
		"bold kind with linked backticked target": {
			bullet: "- **composes-with** [`alpha`](../alpha/SKILL.md): because",
			want: []related.Edge{
				{Kind: related.ComposesWith, Target: "alpha", Rationale: "because"},
			},
		},
		"plain kind with linked bare target": {
			bullet: "- depends-on: [alpha](../alpha/SKILL.md) — because",
			want: []related.Edge{
				{Kind: related.DependsOn, Target: "alpha", Rationale: "because"},
			},
		},
		"reversed: bold slug with kind in parens": {
			bullet: "- **alpha** (contrasts-with): because",
			want: []related.Edge{
				{Kind: related.ContrastsWith, Target: "alpha", Rationale: "because"},
			},
		},
		"bare token followed by prose": {
			bullet: "- depends-on: alpha (because of things)",
			want: []related.Edge{
				{Kind: related.DependsOn, Target: "alpha", Rationale: "(because of things)"},
			},
		},
		"multi-target expands to one edge per target": {
			bullet: "- composes-with: `alpha`, `beta`",
			want: []related.Edge{
				{Kind: related.ComposesWith, Target: "alpha"},
				{Kind: related.ComposesWith, Target: "beta"},
			},
		},
		"no rationale": {
			bullet: "- depends-on: `alpha`",
			want:   []related.Edge{{Kind: related.DependsOn, Target: "alpha"}},
		},
		"asterisk list marker": {
			bullet: "* depends-on: `alpha` — because",
			want: []related.Edge{
				{Kind: related.DependsOn, Target: "alpha", Rationale: "because"},
			},
		},

		// Regression: a first draft scanned the whole line for backticks and
		// manufactured edges to skills named "--force" and "--yes" out of a
		// rationale, which the verify graph gate then reported as dangling.
		"backticked code in the rationale is not a target": {
			bullet: "- composes-with: `alpha` — tier flags (`--force`, `--yes`) are stable",
			want: []related.Edge{
				{
					Kind:      related.ComposesWith,
					Target:    "alpha",
					Rationale: "tier flags (`--force`, `--yes`) are stable",
				},
			},
		},
		// Regression: unknown kinds must stay skipped. Both of these appear in real
		// trees and a first draft accepted the reversed one.
		"unknown kind in the reversed form yields nothing": {
			bullet: "- **alpha** (broader): because",
		},
		"unknown kind in the plain form yields nothing": {
			bullet: "- precedes: `alpha` — because",
		},
		// A bullet whose target is prose names no skill; inventing one would create
		// an edge the graph gate would report as dangling.
		"parenthesised prose yields nothing": {
			bullet: "- contrasts-with: (traditional headcount-scaling model)",
		},
		"capitalised words are not a slug": {
			bullet: "- depends-on: Four Golden Signals",
		},
		"not a bullet at all": {
			bullet: "some prose about depends-on: `alpha`",
		},
		"thematic break is not a bullet": {
			bullet: "---",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			md := "## Related skills\n\n" + tc.bullet + "\n"
			got := related.ParseSection(md)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseSection(%q) = %+v, want %+v", tc.bullet, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("edge %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseSectionHeadingVariants(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		heading string
		want    int
	}{
		"canonical heading": {heading: "## Related skills", want: 1},
		"suffixed heading is the section": {
			heading: "## Related skills (Stage 3 Filling)",
			want:    1,
		},
		"capitalised heading is the section": {heading: "## Related Skills", want: 1},
		"capitalised and suffixed heading is the section": {
			heading: "## Related Skills (Stage 3 Filling)",
			want:    1,
		},
		"deeper level is not the section": {heading: "### Related skills", want: 0},
		"different word is not the section": {
			heading: "## Related skillset",
			want:    0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			md := tc.heading + "\n\n- depends-on: `alpha` — because\n"
			if got := len(related.ParseSection(md)); got != tc.want {
				t.Errorf("%s: parsed %d edges, want %d", tc.heading, got, tc.want)
			}
		})
	}
}

func TestParseSectionDedupesRepeatedEdges(t *testing.T) {
	t.Parallel()
	// The state after `relate` runs over a legacy section: a legacy bullet and the
	// canonical bullet it was rewritten as both name the same relationship.
	md := "## Related skills\n\n" +
		"- **composes-with** [`alpha`](../alpha/SKILL.md): legacy wording\n" +
		"- composes-with: `alpha` — canonical wording\n"
	got := related.ParseSection(md)
	if len(got) != 1 {
		t.Fatalf("expected the relationship once, got %+v", got)
	}
	if got[0].Rationale != "legacy wording" {
		t.Errorf("first occurrence should win, got %q", got[0].Rationale)
	}
}

func TestUpsertOverLegacySectionStaysIdempotent(t *testing.T) {
	t.Parallel()
	// The write path must not have regressed: Upsert matches only canonical bullets,
	// so it appends beside a legacy one, and a second identical Upsert is a no-op.
	md := "# Skill\n\n## Related skills (Stage 3 Filling)\n\n" +
		"- **composes-with** [`alpha`](../alpha/SKILL.md): legacy wording\n"
	edge := related.Edge{Kind: related.DependsOn, Target: "beta", Rationale: "why"}

	first, changed := related.Upsert(md, edge)
	if !changed {
		t.Fatal("first Upsert must change the section")
	}
	second, changedAgain := related.Upsert(first, edge)
	if changedAgain {
		t.Errorf("second Upsert must be a no-op, got:\n%s", second)
	}
	// It must write into the suffixed section, not append a second one.
	if n := countHeadings(first); n != 1 {
		t.Errorf("expected exactly one related-skills heading, got %d:\n%s", n, first)
	}
	// The legacy edge must survive the write untouched.
	edges := related.ParseSection(first)
	if len(edges) != 2 {
		t.Fatalf("expected the legacy and new edge, got %+v", edges)
	}
}

// countHeadings counts the `## Related skills...` headings in md.
func countHeadings(md string) int {
	n := 0
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## Related skills") {
			n++
		}
	}
	return n
}

func TestParseSectionReadsQualifiedTargets(t *testing.T) {
	t.Parallel()
	// The form the real books already use for cross-tree links: 26 superseded-by
	// bullets and 9 on the other kinds, none of which parsed before.
	cases := map[string]struct {
		bullet string
		want   related.Edge
	}{
		"superseded-by, bold kind, bare qualified token": {
			bullet: "- **superseded-by**: merged/all-books-v1/merged-skill — replaced by the merge",
			want: related.Edge{
				Kind:      related.SupersededBy,
				Target:    "merged/all-books-v1/merged-skill",
				Rationale: "replaced by the merge",
			},
		},
		"depends-on, canonical, backticked qualified target": {
			bullet: "- depends-on: `other-book/some-skill` — needs it first",
			want: related.Edge{
				Kind:      related.DependsOn,
				Target:    "other-book/some-skill",
				Rationale: "needs it first",
			},
		},
		"composes-with, linked qualified target": {
			bullet: "- **composes-with** → [`other-book/some-skill`](../../other-book/some-skill/SKILL.md): together",
			want: related.Edge{
				Kind:      related.ComposesWith,
				Target:    "other-book/some-skill",
				Rationale: "together",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := related.ParseSection("## Related Skills\n\n" + tc.bullet + "\n")
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("ParseSection = %#v, want one %#v", got, tc.want)
			}
		})
	}
}

func TestParseSectionStillRejectsProseWithSlashes(t *testing.T) {
	t.Parallel()
	// Tolerating a path must not turn prose into edges: every segment has to be a
	// strict slug, so a sentence containing a slash names no skill.
	for name, bullet := range map[string]string{
		"prose in parentheses": "- superseded-by: (the read/write split we abandoned)",
		"capitalised path":     "- superseded-by: Merged/All-Books/Skill",
		"trailing separator":   "- superseded-by: merged/all-books-v1/",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := related.ParseSection("## Related Skills\n\n" + bullet + "\n"); got != nil {
				t.Errorf("ParseSection = %#v, want no edges", got)
			}
		})
	}
}

// TestSixthDialect covers "- **slug** — kind: why", the orientation the books write most
// and the reader understood least: measured against the 233-skill corpus, not one of its
// 110 bullets parsed before this.
func TestSixthDialect(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		bullet string
		want   related.Kind
	}{
		"informs maps to itself": {
			"- **alpha** — informs: shapes how it is applied.",
			related.Informs,
		},
		"combines is composes-with": {
			"- **alpha** — combines: use them together.",
			related.ComposesWith,
		},
		"compares is contrasts-with": {
			"- **alpha** — compares: alternatives.",
			related.ContrastsWith,
		},
		"depends on is depends-on": {
			"- **alpha** — depends on: needs it first.",
			related.DependsOn,
		},
		"canonical spelling still ok": {
			"- **alpha** — composes-with: together.",
			related.ComposesWith,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := related.ParseSection("## Related Skills\n\n" + tc.bullet + "\n")
			if len(got) != 1 {
				t.Fatalf("got %d edges, want 1: %+v", len(got), got)
			}
			if got[0].Kind != tc.want {
				t.Errorf("Kind = %q, want %q", got[0].Kind, tc.want)
			}
			if got[0].Target != "alpha" {
				t.Errorf("Target = %q, want alpha", got[0].Target)
			}
			if got[0].Rationale == "" {
				t.Error("rationale was dropped")
			}
		})
	}
}

// TestPrerequisiteForIsNotAbsorbed pins the one dialect kind that must stay unread. It is
// the inverse of depends-on -- 13 of its 18 corpus edges have their exact flip already
// present as a depends-on bullet in the target's file -- so absorbing it here would either
// reverse real edges or attribute an edge to a skill whose text never declared it.
func TestPrerequisiteForIsNotAbsorbed(t *testing.T) {
	t.Parallel()
	got := related.ParseSection(
		"## Related Skills\n\n- **alpha** — prerequisite for: run this one first.\n")
	if len(got) != 0 {
		t.Errorf("'prerequisite for' was absorbed as %+v; it needs a rewrite, not a read", got)
	}
}

// TestReversedFormsDoNotInventTargets pins both reversed orientations against a rationale
// of bare words. takeTargets keeps taking targets while the head parses as one, so without
// a separator it stops at, "use them together" becomes two extra edges.
func TestReversedFormsDoNotInventTargets(t *testing.T) {
	t.Parallel()
	for name, bullet := range map[string]string{
		"paren":            "- **alpha** (composes-with) use them together.",
		"paren with colon": "- **alpha** (composes-with): use them together.",
		"dash":             "- **alpha** — combines: use them together.",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := related.ParseSection("## Related Skills\n\n" + bullet + "\n")
			if len(got) != 1 {
				t.Fatalf("got %d edges, want 1: %+v", len(got), got)
			}
			if got[0].Target != "alpha" {
				t.Errorf("Target = %q, want alpha", got[0].Target)
			}
		})
	}
}

// TestReversedParenFormKeepsItsRationale pins the second half of that fix. Adding the em
// dash alone left the rationale as "— : because", because this dialect writes "(kind): why"
// and the colon was reaching trimRationale as a leading character it happened to strip.
func TestReversedParenFormKeepsItsRationale(t *testing.T) {
	t.Parallel()
	got := related.ParseSection(
		"## Related Skills\n\n- **alpha** (contrasts-with): because it differs.\n")
	if len(got) != 1 {
		t.Fatalf("got %d edges, want 1: %+v", len(got), got)
	}
	if got[0].Rationale != "because it differs." {
		t.Errorf("Rationale = %q, want %q", got[0].Rationale, "because it differs.")
	}
}

// wantEdges parses one bullet and reports the kind and target it yielded, or ok=false.
// Extracted before the table so each row reads as a claim about a corpus dialect.
func wantEdges(t *testing.T, bullet string) (related.Edge, bool) {
	t.Helper()
	edges := related.ParseSection("## Related Skills\n\n" + bullet + "\n")
	if len(edges) == 0 {
		return related.Edge{}, false
	}
	return edges[0], true
}

// TestDialectsAddedFromCorpusMeasurement. Measured over a 233-skill tree: the reader
// resolved 222 edges from 520 bullets. These are the three forms that accounted for the
// mechanically-readable part of the gap, each traced to where it died in readBullet.
func TestDialectsAddedFromCorpusMeasurement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		bullet string
		kind   related.Kind
		target string
		ok     bool
	}{{
		// Died in canonicalKind: "composes_with" is not a known spelling.
		name:   "an underscore kind reads as the hyphenated one",
		bullet: "- **composes_with**: alpha — why",
		kind:   related.ComposesWith, target: "alpha", ok: true,
	}, {
		// Died in canonicalKind: the asterisks were part of the name it compared.
		name:   "an italic kind in the reversed orientation reads",
		bullet: "- **alpha** — *combines*: why",
		kind:   related.ComposesWith, target: "alpha", ok: true,
	}, {
		// Died in splitReversedDash: Cut(after, ":") found no colon.
		name:   "an arrow may separate the kind from the rationale",
		bullet: "- **alpha** — *depends-on* → why",
		kind:   related.DependsOn, target: "alpha", ok: true,
	}, {
		name:   "an underscore kind also reads in the reversed orientation",
		bullet: "- **alpha** — depends_on: why",
		kind:   related.DependsOn, target: "alpha", ok: true,
	}, {
		// The refusals dialects.go records must survive the new tolerance.
		//
		// "prerequisite for" is the inverse of depends-on: 13 of its 18 corpus edges
		// already have their flip in the other skill's file. Reading it without flipping
		// reverses real edges; flipping it attributes an edge to a file that never
		// declared it. It is a rewrite, not a read.
		name:   "prerequisite-for is still refused",
		bullet: "- **alpha** — *prerequisite for*: why", ok: false,
	}, {
		name:   "a kind outside the vocabulary is still refused",
		bullet: "- **teleports-to**: alpha — why", ok: false,
	}, {
		// isSlug in the reversed orientation is what keeps a display title out. Resolving
		// a title is the tree-walker's job; admitting it here would resolve titles by
		// accident and silently.
		name:   "a bold display title is still refused",
		bullet: "- **Alpha Skill** — *depends-on* → why", ok: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			edge, ok := wantEdges(t, tt.bullet)
			if ok != tt.ok {
				t.Fatalf("parsed = %v, want %v (edge %+v)", ok, tt.ok, edge)
			}
			if !tt.ok {
				return
			}
			if edge.Kind != tt.kind || edge.Target != tt.target {
				t.Errorf("edge = %+v, want kind %q target %q", edge, tt.kind, tt.target)
			}
		})
	}
}

// TestEmphasisSurvivesInARationale. The emphasis stripping is scoped to the kind token on
// purpose: a rationale legitimately contains *emphasis*, and stripping it wherever it
// appeared would edit prose the author wrote.
func TestEmphasisSurvivesInARationale(t *testing.T) {
	t.Parallel()
	edge, ok := wantEdges(t, "- depends-on: `alpha` — it is *strictly* required")
	if !ok {
		t.Fatal("a canonical bullet stopped parsing")
	}
	if !strings.Contains(edge.Rationale, "*strictly*") {
		t.Errorf("rationale = %q, want the emphasis untouched", edge.Rationale)
	}
}
