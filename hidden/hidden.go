// Package hidden finds codepoints that are invisible when a document is read but present
// when an agent obeys it: zero-width characters, bidirectional controls, and the Unicode
// tag block.
//
// It is the one classifier for the family. exegesis and gnosis each carried an identical
// copy, and both were wrong in the same ways: they flagged every zero-width joiner, which
// rejects emoji sequences and Persian and Arabic text, and the whole tag block, which
// rejects the England, Scotland and Wales flags. Two copies would also have to be fixed
// twice and could then disagree, which for a security check means one tool passes what
// another blocks.
//
// # Why these constants are allowed to block
//
// The codepoints are ranges from the Unicode standard, not tuned thresholds, and the
// three context rules are exact: a joiner is suspicious only next to ASCII or another
// listed character; exactly three tag sequences are exempt; a byte-order mark is allowed
// only at offset 0. None has a number to argue down, which is what lets a gate block on
// them. The context rules are stbenjam/skillsaw's (security/invisible_unicode.py),
// re-derived here.
//
// Everything here is pure.
package hidden

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// The classes of hidden codepoint. The names are gnosis's, which already appear in its
// stored reports.
const (
	// ClassZeroWidth is text that occupies no space and survives copy-paste.
	// Instructions written in it are invisible in every reviewer's editor.
	ClassZeroWidth Class = "zero-width"

	// ClassBidi is the Trojan Source class: bidirectional controls that make rendered
	// order differ from stored order, so what a reviewer reads is not what a model
	// consumes.
	ClassBidi Class = "bidi-override"

	// ClassTag is the Unicode tag block, deprecated for language tagging and repurposed
	// as a channel for text no renderer displays at all.
	ClassTag Class = "unicode-tag"
)

// Class names a kind of hidden codepoint.
type Class string

// Hit is one hidden codepoint in a text.
type Hit struct {
	Offset int   // byte offset of the codepoint in the scanned text
	Line   int   // 1-based line number, counting "\n"
	Class  Class // which class the codepoint belongs to
	Rune   rune  // the codepoint itself; render it with Codepoint, never literally
}

// Summary is one class of hidden codepoint found in one text: the first occurrence and a
// count. One per class rather than one per occurrence, because a document with four
// hundred zero-width joiners has one problem, and four hundred findings would bury it.
type Summary struct {
	Class  Class  `json:"class"`
	Rune   string `json:"rune"`   // first codepoint of the class, as U+XXXX
	Offset int    `json:"offset"` // byte offset of that first occurrence
	Count  int    `json:"count"`  // how many codepoints of the class the text holds
}

// Stripped is a text with every listed codepoint removed, and the way back to the
// original's offsets.
type Stripped struct {
	Text    string
	offsets []int // offsets[i] is the original byte offset of Text[i]; one extra at the end
}

// Classify reports the class of r, or false when r is not a listed codepoint. It ignores
// context: a joiner or byte-order mark is listed here even where Scan would not report
// it, which is what a caller removing every invisible codepoint wants.
//
// The ranges are written as literals so a reader can check them against the standard
// without running anything.
func Classify(r rune) (Class, bool) {
	switch {
	case r == 0x00AD, // soft hyphen
		r == 0x180E,                // Mongolian vowel separator
		r >= 0x200B && r <= 0x200D, // zero-width space, non-joiner, joiner
		r >= 0x2060 && r <= 0x2064, // word joiner, invisible math operators
		r == 0xFEFF:                // zero-width no-break space / byte-order mark
		return ClassZeroWidth, true
	case r == 0x061C, // Arabic letter mark
		r == 0x200E, r == 0x200F, // left-to-right and right-to-left marks
		r >= 0x202A && r <= 0x202E, // embeddings and overrides
		r >= 0x2066 && r <= 0x2069: // isolates
		return ClassBidi, true
	case r >= 0xE0001 && r <= 0xE007F:
		return ClassTag, true
	}
	return "", false
}

// Scan reports every hidden codepoint in text that its context does not excuse.
//
// Requires: nothing; text may be empty or invalid UTF-8 (an invalid byte is not hidden).
// Ensures:  hits are in offset order; each listed codepoint is reported unless it is a
//
//	zero-width joiner or non-joiner with no ASCII or listed neighbour, a byte-order mark
//	at offset 0, or a tag character inside one of the three exempt flag sequences.
func Scan(text string) []Hit {
	exempt := flagTagOffsets(text)
	runes := []rune(text)
	var hits []Hit
	line, i := 1, 0
	for offset, r := range text {
		if class, ok := Classify(r); ok && !excused(runes, i, offset, r, exempt) {
			hits = append(hits, Hit{Offset: offset, Line: line, Class: class, Rune: r})
		}
		if r == '\n' {
			line++
		}
		i++
	}
	return hits
}

// excused reports whether the listed codepoint r, at rune index i and byte offset, is
// legitimate in its context.
func excused(runes []rune, i, offset int, r rune, exemptTags map[int]bool) bool {
	switch {
	case r == 0xFEFF:
		return offset == 0
	case r == 0x200C, r == 0x200D:
		return !joinerSuspicious(runes, i)
	case r >= 0xE0001 && r <= 0xE007F:
		return exemptTags[offset]
	}
	return false
}

// joinerSuspicious reports whether the joiner at runes[i] sits next to ASCII or another
// listed codepoint. Legitimate joiners sit between emoji or cursive-script letters, all
// non-ASCII; one beside ASCII was put into plain text to split tokens invisibly, and one
// beside another listed character is part of a stacked payload.
func joinerSuspicious(runes []rune, i int) bool {
	for _, j := range []int{i - 1, i + 1} {
		if j < 0 || j >= len(runes) {
			continue
		}
		if runes[j] < 0x80 {
			return true
		}
		if _, listed := Classify(runes[j]); listed {
			return true
		}
	}
	return false
}

// flagTagOffsets returns the byte offsets of tag characters that belong to the England,
// Scotland or Wales flag. These are the only recommended uses of tag characters, and each
// is an exact sequence (black flag, a fixed region code in tag letters, cancel tag), so
// the exemption cannot carry anything but its own region code.
func flagTagOffsets(text string) map[int]bool {
	exempt := map[int]bool{}
	for _, seq := range flagSequences() {
		base := len("\U0001F3F4")
		for start := 0; ; {
			k := strings.Index(text[start:], seq)
			if k < 0 {
				break
			}
			at := start + k
			for off := range seq[base:] {
				exempt[at+base+off] = true
			}
			start = at + len(seq)
		}
	}
	return exempt
}

// flagSequences builds the three exempt flag emoji. Built per call rather than kept in a
// package variable, matching textnorm: it is three short strings, and package state is
// shared state.
func flagSequences() []string {
	seqs := make([]string, 0, 3)
	for _, region := range []string{"gbeng", "gbsct", "gbwls"} {
		var b strings.Builder
		b.WriteString("\U0001F3F4") // waving black flag
		for _, c := range region {
			b.WriteRune(0xE0000 + c) // the tag letter mirroring c
		}
		b.WriteRune(0xE007F) // cancel tag
		seqs = append(seqs, b.String())
	}
	return seqs
}

// ByClass aggregates hits to one Summary per class, ordered by class name so two runs
// over one text produce comparable output. It returns an empty, non-nil slice for no hits.
func ByClass(hits []Hit) []Summary {
	byClass := map[Class]*Summary{}
	for _, h := range hits {
		if s, seen := byClass[h.Class]; seen {
			s.Count++
			continue
		}
		byClass[h.Class] = &Summary{
			Class:  h.Class,
			Rune:   Codepoint(h.Rune),
			Offset: h.Offset,
			Count:  1,
		}
	}
	out := make([]Summary, 0, len(byClass))
	for _, s := range byClass {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Class < out[j].Class })
	return out
}

// Codepoint renders r the way the Unicode standard writes one: U+ and at least four hex
// digits. A hidden character must never be echoed literally, because a report read by an
// agent would carry the payload onward.
func Codepoint(r rune) string { return fmt.Sprintf("U+%04X", r) }

// Strip removes every codepoint Classify lists, regardless of context, so a pattern can
// be matched against what a reader sees: "ig­nore" matches "ignore" once stripped.
//
// Ensures: Strip(s.Text).Text == s.Text; Scan(s.Text) is empty; Original maps every
//
//	offset in s.Text (and len(s.Text)) back into the original text; it is pure.
func Strip(text string) Stripped {
	var b strings.Builder
	offsets := make([]int, 0, len(text)+1)
	for offset, r := range text {
		if _, listed := Classify(r); listed {
			continue
		}
		n := b.Len()
		_, size := utf8.DecodeRuneInString(text[offset:])
		b.WriteString(text[offset : offset+size])
		for k := n; k < b.Len(); k++ {
			offsets = append(offsets, offset+(k-n))
		}
	}
	offsets = append(offsets, len(text))
	return Stripped{Text: b.String(), offsets: offsets}
}

// Original maps a byte offset in s.Text to the corresponding offset in the text Strip was
// given. An offset outside [0, len(s.Text)] is clamped, since a caller asking about one
// has a bug of its own and a panic here would hide where.
func (s Stripped) Original(i int) int {
	switch {
	case len(s.offsets) == 0:
		return 0
	case i < 0:
		return s.offsets[0]
	case i >= len(s.offsets):
		return s.offsets[len(s.offsets)-1]
	}
	return s.offsets[i]
}
