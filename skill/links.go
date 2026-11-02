package skill

import (
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/StevenACoffman/skillet/markdown"
)

// BrokenLinks returns the relative link and image destinations in body that name no
// file or directory in fsys, which is the skill's own directory. Each is returned as
// written, once, in document order.
//
// It is the one definition of "a link that does not resolve" for the family: exegesis
// gates on it and skillsaw scores it, and two definitions would disagree about which
// links count.
//
// Only links that should resolve inside the skill are judged. A destination is skipped
// when it is empty, a fragment ("#heading"), has a URL scheme ("https:", "mailto:"), is
// absolute or home-relative ("/x", "~/x"), or escapes the skill ("../x"). exegesis
// reports absolute and escaping links as defects of their own, and a broken-link
// finding for them would repeat it under a worse name. Links inside code are code, not
// links, and are never judged.
//
// Requires: fsys is rooted at the skill directory.
// Ensures:  every result is a destination from body whose path, after dropping any
//
//	fragment or query and percent-decoding, does not exist in fsys.
func BrokenLinks(fsys fs.FS, body string) []string {
	var broken []string
	seen := map[string]bool{}
	for _, dest := range markdown.Parse(body).LinkDestinations {
		target, judged := localTarget(dest)
		if !judged || seen[dest] {
			continue
		}
		seen[dest] = true
		if _, err := fs.Stat(fsys, target); err != nil {
			broken = append(broken, dest)
		}
	}
	return broken
}

// localTarget turns a link destination into a path inside the skill, or reports false
// when the destination is not one BrokenLinks judges.
func localTarget(dest string) (string, bool) {
	dest = strings.TrimSpace(dest)
	if i := strings.IndexAny(dest, "#?"); i >= 0 {
		dest = dest[:i]
	}
	if external(dest) {
		return "", false
	}
	if unescaped, err := url.PathUnescape(dest); err == nil {
		dest = unescaped
	}
	clean := path.Clean(dest)
	if clean == ".." || strings.HasPrefix(clean, "../") || !fs.ValidPath(clean) {
		return "", false
	}
	return clean, true
}

// external reports whether dest points outside the skill by its form alone: empty (a
// bare fragment), absolute, home-relative, or carrying a URL scheme.
func external(dest string) bool {
	if dest == "" || strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "~") {
		return true
	}
	u, err := url.Parse(dest)
	return err == nil && u.Scheme != ""
}
