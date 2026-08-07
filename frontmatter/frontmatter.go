// Package frontmatter separates a leading "---"-delimited YAML block from the markdown
// body that follows it.
//
// It exists so that every file in the family carrying a YAML header — a SKILL.md, a
// merge run's source-verification record, anything later — agrees on where the header
// ends. A second implementation would differ on the awkward inputs (an empty block, a
// missing closing delimiter, CRLF line endings), and two tools would then disagree about
// what a document's header is.
//
// Unmarshalling is deliberately left to the caller: one reads the block into a map to
// enumerate its keys, another into a typed struct. Wrapping the YAML library here would
// serve neither and would only add a layer to see through.
//
// Split is pure; nothing in this package touches the filesystem.
package frontmatter

import "strings"

const delimiter = "---"

// Split separates a leading "---"-delimited YAML block from the markdown body.
//
// The opening delimiter must be the document's first line; a document that does not
// begin with one has no frontmatter, and body is the whole document. The block ends at
// the next line beginning with the delimiter, and body is everything after that line.
//
// CRLF line endings are normalized to LF before anything else, so a header written on
// Windows is found. That normalization lives here rather than in each caller precisely
// because a caller that forgets it does not get an error — it gets an empty block, and
// then reports the fields as missing from a file where they are plainly present.
//
// An unterminated block is not frontmatter: the opening delimiter is left in body rather
// than consumed, so the one rule a caller needs is that an empty block means body holds
// the entire document.
//
// Ensures: body never contains the block or its delimiters; it is pure.
func Split(text string) (block, body string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	open := delimiter + "\n"
	if !strings.HasPrefix(text, open) {
		return "", text
	}
	// Scan by line rather than searching for "\n---": the closing delimiter may be the
	// very first line after the opening one, where it has no newline before it, and a
	// substring search would miss it and call a well-formed empty header malformed.
	lines := strings.Split(text[len(open):], "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, delimiter) {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	// An opening delimiter with no close. Returning the remainder here would silently
	// eat the first line of a malformed document; hand back what we were given.
	return "", text
}
