package fsutil_test

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/StevenACoffman/skillet/fsutil"
)

// errFS forces a non-ErrNotExist error when a named directory is read, to prove
// SubdirsContainingAny surfaces real faults instead of silently skipping them.
type errFS struct {
	fs.FS
	failDir string
}

func (e errFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == e.failDir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrPermission}
	}
	return fs.ReadDir(e.FS, name)
}

func TestSubdirsContaining(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a/SKILL.md":     {Data: []byte("x")},
		"b/SKILL.md":     {Data: []byte("x")},
		"c/notes.txt":    {Data: []byte("x")}, // dir without the marker
		"top.txt":        {Data: []byte("x")}, // top-level file, not a dir
		"d/sub/SKILL.md": {Data: []byte("x")}, // marker is nested, not direct
	}
	got, err := fsutil.SubdirsContaining(fsys, ".", "SKILL.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SubdirsContaining = %v, want %v", got, want)
	}
}

func TestSubdirsContainingNoMatches(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"only/notes.txt": {Data: []byte("x")}}
	got, err := fsutil.SubdirsContaining(fsys, ".", "SKILL.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no matches, got %v", got)
	}
}

func TestSubdirsContainingUnreadableDir(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a/SKILL.md": {Data: []byte("x")}}
	_, err := fsutil.SubdirsContaining(fsys, "nope", "SKILL.md")
	if err == nil {
		t.Fatal("want error for a missing dir, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want fs.ErrNotExist, got %v", err)
	}
}

func TestSubdirsContainingAnyDedupAndSkip(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		".claude/skills/alpha/SKILL.md": {Data: []byte("x")},
		".cursor/skills/beta/SKILL.md":  {Data: []byte("x")},
	}
	// The first root is repeated (dedup) and a nonexistent root is included (skip).
	roots := []string{".claude/skills", ".cursor/skills", ".claude/skills", "missing/root"}
	got, err := fsutil.SubdirsContainingAny(fsys, roots, "SKILL.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{".claude/skills/alpha", ".cursor/skills/beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SubdirsContainingAny = %v, want %v", got, want)
	}
}

func TestSubdirsContainingAnySurfacesRealErrors(t *testing.T) {
	t.Parallel()
	base := fstest.MapFS{"open/alpha/SKILL.md": {Data: []byte("x")}}
	fsys := errFS{FS: base, failDir: "locked"}
	_, err := fsutil.SubdirsContainingAny(fsys, []string{"open", "locked"}, "SKILL.md")
	if err == nil {
		t.Fatal("want error from the locked dir, got nil")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("want fs.ErrPermission, got %v", err)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatal("a permission fault must not be treated as a missing root")
	}
}
