// Package fsutil provides small, filesystem-shaped helpers shared by skillet
// tools. The functions take an fs.FS rather than touching the OS directly, so
// callers inject os.DirFS in production and testing/fstest.MapFS in tests, and
// the discovery logic stays pure and independently testable.
package fsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// Found is the result of a discovery walk: the directories that hold the marker, and
// every entry that could not be judged.
type Found struct {
	Dirs    []string  // slash paths joined onto the walked dir
	Skipped []Skipped // entries left out because they could not be read
}

// Skipped is a directory entry a walk could not judge, and why. It is reported rather
// than dropped: an entry nobody checked must not read as one that was checked and found
// to hold nothing.
type Skipped struct {
	Path   string `json:"path"`   // slash path within the walked filesystem
	Reason string `json:"reason"` // the error that stopped the walk reading it
}

// SubdirsContaining returns the immediate subdirectories of dir (within fsys) that
// directly contain a file named marker, as slash paths joined onto dir, sorted by name.
//
// Directories are decided with fs.Stat rather than DirEntry.IsDir, so a symbolic link
// to a directory counts as one. Given an fs.FS from os.Root.FS, a link that leaves the
// root cannot be followed; it, and any other entry or marker that cannot be read, is
// returned in Skipped with the error that stopped it. A marker that simply does not
// exist is not a skip: that directory is not a match.
//
// It returns an error if dir itself cannot be read. Pass "." for the root of fsys.
func SubdirsContaining(fsys fs.FS, dir, marker string) (Found, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return Found{}, fmt.Errorf("fsutil.SubdirsContaining %q: %w", dir, err)
	}
	found := Found{Dirs: make([]string, 0, len(entries))}
	for _, e := range entries {
		sub := path.Join(dir, e.Name())
		info, err := fs.Stat(fsys, sub)
		if err != nil {
			found.Skipped = append(found.Skipped, Skipped{Path: sub, Reason: err.Error()})
			continue
		}
		if !info.IsDir() {
			continue
		}
		switch _, err := fs.Stat(fsys, path.Join(sub, marker)); {
		case err == nil:
			found.Dirs = append(found.Dirs, sub)
		case !errors.Is(err, fs.ErrNotExist):
			found.Skipped = append(
				found.Skipped,
				Skipped{Path: path.Join(sub, marker), Reason: err.Error()},
			)
		}
	}
	return found, nil
}

// SubdirsContainingAny unions SubdirsContaining across dirs, deduplicating repeated
// subdirectory paths and preserving order — dirs in the given order, subdirectories
// sorted within each. A dir that does not exist is skipped, modelling an optional root
// absent in the current environment; any other read error is returned, so a permission
// fault is surfaced rather than hidden. This is the multi-root discovery shape.
func SubdirsContainingAny(fsys fs.FS, dirs []string, marker string) (Found, error) {
	var out Found
	seen := make(map[string]bool)
	for _, dir := range dirs {
		found, err := SubdirsContaining(fsys, dir, marker)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return Found{}, err
		}
		for _, sub := range found.Dirs {
			if !seen[sub] {
				seen[sub] = true
				out.Dirs = append(out.Dirs, sub)
			}
		}
		out.Skipped = append(out.Skipped, found.Skipped...)
	}
	return out, nil
}
