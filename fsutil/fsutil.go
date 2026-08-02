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

// SubdirsContaining returns the immediate subdirectories of dir (within fsys)
// that directly contain a file named marker, as slash paths joined onto dir.
// The result is sorted by name, because fs.ReadDir returns entries in order.
//
// It returns an error if dir cannot be read. Pass "." for the root of fsys.
// This is the single-tree discovery shape (a marker file in each candidate
// child of one parent directory).
func SubdirsContaining(fsys fs.FS, dir, marker string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("fsutil.SubdirsContaining %q: %w", dir, err)
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := path.Join(dir, e.Name())
		if hasMarker(fsys, sub, marker) {
			dirs = append(dirs, sub)
		}
	}
	return dirs, nil
}

// SubdirsContainingAny unions SubdirsContaining across dirs, deduplicating
// repeated subdirectory paths and preserving order — dirs in the given order,
// subdirectories sorted within each. A dir that does not exist is skipped,
// modelling an optional root absent in the current environment; any other read
// error is returned, so a permission fault is surfaced rather than hidden. This
// is the multi-root discovery shape (scan several candidate parents at once).
func SubdirsContainingAny(fsys fs.FS, dirs []string, marker string) ([]string, error) {
	var out []string
	seen := make(map[string]bool)
	for _, dir := range dirs {
		subs, err := SubdirsContaining(fsys, dir, marker)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, sub := range subs {
			if !seen[sub] {
				seen[sub] = true
				out = append(out, sub)
			}
		}
	}
	return out, nil
}

// hasMarker reports whether a file named marker exists directly inside dir.
func hasMarker(fsys fs.FS, dir, marker string) bool {
	_, err := fs.Stat(fsys, path.Join(dir, marker))
	return err == nil
}
