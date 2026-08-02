package manifest_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/StevenACoffman/skillet/manifest"
)

func TestBuildSortsAndDoesNotMutate(t *testing.T) {
	t.Parallel()
	in := []manifest.Skill{
		{Slug: "zebra", Dir: "d/zebra"},
		{Slug: "alpha", Dir: "d/alpha"},
	}
	m := manifest.Build("exegesis", "tree", in, true)
	if m.Tool != "exegesis" || m.Tree != "tree" || !m.StructureVerified {
		t.Fatalf("header wrong: %+v", m)
	}
	if m.Skills[0].Slug != "alpha" || m.Skills[1].Slug != "zebra" {
		t.Errorf("skills not sorted by slug: %+v", m.Skills)
	}
	if in[0].Slug != "zebra" {
		t.Error("Build must not mutate the input slice order")
	}
}

func TestMarshal(t *testing.T) {
	t.Parallel()
	m := manifest.Build("skillsaw", "tree", []manifest.Skill{
		{Slug: "a", Dir: "d/a", Hash: "abc123"},
		{Slug: "b", Dir: "d/b"}, // no hash, no test-prompts -> omitempty
	}, false)
	b, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		t.Error("Marshal output must end with a newline")
	}
	var round manifest.Manifest
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if round.Tool != "skillsaw" || len(round.Skills) != 2 {
		t.Errorf("round-trip mismatch: %+v", round)
	}
	if bytes.Contains(b, []byte(`"sha256"`)) && bytes.Contains(b, []byte(`"test_prompts"`)) {
		// b's second skill has neither; ensure omitempty dropped them somewhere.
		if bytes.Count(b, []byte(`"sha256"`)) != 1 {
			t.Errorf("omitempty: expected exactly one sha256 field, got:\n%s", b)
		}
	}
}
