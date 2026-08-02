package provenance_test

import (
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/provenance"
)

func TestPresent(t *testing.T) {
	t.Parallel()
	if provenance.Present([]byte("# just a comment\nbody\n")) {
		t.Error("owned file should not be Present")
	}
	if !provenance.Present([]byte("# skillet-vendored-origin: https://x\nbody\n")) {
		t.Error("stamped file should be Present")
	}
}

func TestParseNilForOwnedFile(t *testing.T) {
	t.Parallel()
	if provenance.Parse([]byte("kind: Thing\nbody\n")) != nil {
		t.Error("Parse should return nil for a file with no provenance header")
	}
}

func TestStampParseRoundTrip(t *testing.T) {
	t.Parallel()
	content := []byte("# yaml-language-server: $schema=x\nkind: Thing\nbody\n")
	h := &provenance.Header{
		Vendored: provenance.Banner,
		Origin:   "https://github.com/o/r",
		Ref:      "main",
		Commit:   "abc123",
		Imported: "2026-08-01T00:00:00Z",
	}
	h.Digest = provenance.Digest(content)
	stamped := provenance.Stamp(content, h)

	if !strings.HasPrefix(string(stamped), "# yaml-language-server:") {
		t.Errorf("editor directive must stay first:\n%s", stamped)
	}
	got := provenance.Parse(stamped)
	if got == nil {
		t.Fatal("Parse returned nil for a stamped file")
	}
	if *got != *h {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, h)
	}
}

func TestDigestStableUnderStamp(t *testing.T) {
	t.Parallel()
	content := []byte("kind: Thing\nbody line\n")
	h := &provenance.Header{Origin: "o", Digest: provenance.Digest(content)}
	stamped := provenance.Stamp(content, h)
	if provenance.Digest(stamped) != provenance.Digest(content) {
		t.Error("stamping a header must not change the content digest")
	}
	ok, got := provenance.Parse(stamped).Verify(stamped)
	if !ok {
		t.Errorf("Verify should pass on a freshly stamped file; got digest %s", got)
	}
}
