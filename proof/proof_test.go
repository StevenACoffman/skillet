package proof_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/StevenACoffman/skillet/errs"
	"github.com/StevenACoffman/skillet/proof"
)

// writeFile creates root/rel with content.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "a.txt", "alpha")
	writeFile(t, root, "sub/b.txt", "beta")

	pkt, err := proof.Create(root, "arc-1", "deadbeef", []string{"a.txt", "sub/b.txt"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pkt.Provenance == nil || pkt.Provenance.GitSHA != "deadbeef" {
		t.Errorf("provenance not recorded: %+v", pkt.Provenance)
	}
	if err := proof.Verify(root, &pkt); err != nil {
		t.Fatalf("Verify should pass on unchanged bytes: %v", err)
	}
}

func TestCreateEmptyIsInvalid(t *testing.T) {
	t.Parallel()
	_, err := proof.Create(t.TempDir(), "arc", "", nil)
	if errs.ErrorCode(err) != errs.EINVALID {
		t.Fatalf("empty path set: code = %q, want EINVALID", errs.ErrorCode(err))
	}
}

func TestVerifyMissingArtifact(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "a.txt", "alpha")
	pkt, err := proof.Create(root, "arc", "", []string{"a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if code := errs.ErrorCode(proof.Verify(root, &pkt)); code != errs.ECONFLICT {
		t.Fatalf("missing artifact: code = %q, want ECONFLICT", code)
	}
}

func TestVerifyDigestMismatch(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "a.txt", "alpha")
	pkt, err := proof.Create(root, "arc", "", []string{"a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "a.txt", "TAMPERED") // bytes changed after Create
	if code := errs.ErrorCode(proof.Verify(root, &pkt)); code != errs.ECONFLICT {
		t.Fatalf("digest mismatch: code = %q, want ECONFLICT", code)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "a.txt", "alpha")
	pkt, err := proof.Create(root, "arc", "sha", []string{"a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".adh", "proof.json")
	if err := proof.Save(path, &pkt); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := proof.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Arc != "arc" || len(got.Artifacts) != 1 ||
		got.Artifacts[0].Digest != pkt.Artifacts[0].Digest {
		t.Fatalf("Save/Load mismatch: %+v", got)
	}
}
