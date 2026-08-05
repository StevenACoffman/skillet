// Package provenance reads and writes a comment header that marks a file as a
// vendored copy of an artifact whose home is another repository, and computes
// the content digest that header records. The digest is defined over the file
// with its provenance header stripped, so stamping a header does not change the
// digest it records. Adapted, generalized, from modelith's provenance package.
//
// Status: provisional. No package across the family imports it yet — a speculative
// extraction awaiting its first use. Kept as a ready, tested unit; delete it if it
// stays unused rather than let unused surface accrete. See skillet's TODO.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// LinePrefix begins every provenance line. A line is header only when it starts
// at column zero and sits in the leading comment block.
const LinePrefix = "# skillet-vendored-"

// Banner is the value Stamp writes for the vendored key. Nothing enforces it — a
// digest over the file cannot cover the header that records it — but an agent
// editing the file reads it and stops, which is the point.
const Banner = "DO NOT EDIT — this file is a copy. Change it at its origin."

// lineRE matches one provenance line and splits it into key and value.
var lineRE = regexp.MustCompile(`^# skillet-vendored-([a-z][a-z0-9-]*):(.*)$`)

// Header is the provenance block of a vendored file.
type Header struct {
	Vendored string // the DO-NOT-EDIT banner
	Origin   string // the home repository
	Ref      string // branch or tag
	Commit   string // commit SHA
	Imported string // ISO timestamp of the copy
	Digest   string // "sha256:<64 hex>" over the header-stripped content
}

// Present reports whether src carries a provenance line anywhere, which is what
// makes a file vendored.
func Present(src []byte) bool {
	for _, line := range strings.Split(string(src), "\n") {
		if lineRE.MatchString(line) {
			return true
		}
	}
	return false
}

// Parse reads the provenance header from src. It returns nil when src carries no
// provenance line, the ordinary case for a file this repo owns. Only lines in
// the leading comment block are read as header.
func Parse(src []byte) *Header {
	if !Present(src) {
		return nil
	}
	var h Header
	lines := strings.Split(string(src), "\n")
	lead := leadingCommentBlock(lines)
	for i, line := range lines {
		if i >= lead {
			break
		}
		m := lineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if field := h.field(m[1]); field != nil {
			*field = strings.TrimSpace(m[2])
		}
	}
	return &h
}

// Format renders h as the header block, every line terminated. Keys are written
// in a fixed order so the output is deterministic.
func (h *Header) Format() string {
	var b strings.Builder
	for _, key := range []string{"vendored", "origin", "ref", "commit", "imported", "digest"} {
		if v := *h.field(key); v != "" {
			fmt.Fprintf(&b, "%s%s: %s\n", LinePrefix, key, v)
		}
	}
	return b.String()
}

// Strip returns src with its leading-block header lines removed — the content
// the digest covers. A provenance-looking line below the leading comment block
// is not header and is kept.
func Strip(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	lead := leadingCommentBlock(lines)
	kept := lines[:0]
	for i, line := range lines {
		if i < lead && lineRE.MatchString(line) {
			continue
		}
		kept = append(kept, line)
	}
	return []byte(strings.Join(kept, "\n"))
}

// Digest returns the digest of src as a header records it: SHA-256 over src with
// its provenance lines removed, so stamping a header does not change the digest
// of the file it is stamped into.
func Digest(src []byte) string {
	sum := sha256.Sum256(Strip(src))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Verify reports whether src still hashes to the digest its own header records,
// and returns the digest src actually has.
func (h *Header) Verify(src []byte) (ok bool, got string) {
	got = Digest(src)
	return got == h.Digest, got
}

// Stamp returns src with h's header inserted at the top, after a leading
// "# yaml-language-server:" line when src opens with one so the editor directive
// stays first.
func Stamp(src []byte, h *Header) []byte {
	lines := strings.Split(string(src), "\n")
	at, lead := 0, leadingCommentBlock(lines)
	for i := range lead {
		if strings.HasPrefix(lines[i], "# yaml-language-server:") {
			at = i + 1
		}
	}
	head := strings.Join(lines[:at], "\n")
	if at > 0 {
		head += "\n"
	}
	return []byte(head + h.Format() + strings.Join(lines[at:], "\n"))
}

func (h *Header) field(key string) *string {
	switch key {
	case "vendored":
		return &h.Vendored
	case "origin":
		return &h.Origin
	case "ref":
		return &h.Ref
	case "commit":
		return &h.Commit
	case "imported":
		return &h.Imported
	case "digest":
		return &h.Digest
	}
	return nil
}

// leadingCommentBlock returns the number of lines in the run at the top of the
// file that are blank or comments — where the header belongs.
func leadingCommentBlock(lines []string) int {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return i
	}
	return len(lines)
}
