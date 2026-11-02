package hidden_test

import (
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/StevenACoffman/skillet/hidden"
)

// Escapes throughout: a literal hidden character in a test is invisible to its reader.
const (
	zwsp   = "\u200b"
	zwj    = "\u200d"
	zwnj   = "\u200c"
	bom    = "\ufeff"
	shy    = "\u00ad"
	rlo    = "\u202e"
	lrm    = "\u200e"
	tagA   = "\U000E0041"
	family = "\U0001F468\u200d\U0001F469\u200d\U0001F467" // man ZWJ woman ZWJ girl
	// England: black flag, tag letters g b e n g, cancel tag.
	england = "\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F"
)

func TestScan(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		text string
		want []hidden.Hit
	}{
		"empty":       {text: ""},
		"plain ASCII": {text: "nothing to see\n"},
		"zero-width space is always reported": {
			text: "a" + zwsp + "b",
			want: []hidden.Hit{{Offset: 1, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0x200B}},
		},
		"line numbers count newlines": {
			text: "one\ntwo " + rlo,
			want: []hidden.Hit{{Offset: 8, Line: 2, Class: hidden.ClassBidi, Rune: 0x202E}},
		},
		"emoji ZWJ sequence is legitimate": {text: "team " + family + " ok"},
		"Persian ZWNJ between letters is legitimate": {
			text: "\u0645\u06cc" + zwnj + "\u062e\u0648\u0627\u0647\u0645",
		},
		"joiner next to ASCII splits a token": {
			text: "ig" + zwj + "nore",
			want: []hidden.Hit{{Offset: 2, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0x200D}},
		},
		"joiner next to another listed character is stacked": {
			text: "\u0645" + zwnj + zwsp + "\u0645",
			want: []hidden.Hit{
				{Offset: 2, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0x200C},
				{Offset: 5, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0x200B},
			},
		},
		"byte-order mark at offset 0 is allowed": {text: bom + "---\n"},
		"byte-order mark later is content": {
			text: "a" + bom,
			want: []hidden.Hit{{Offset: 1, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0xFEFF}},
		},
		"England flag is exempt": {text: "go " + england + "!"},
		"a lone tag character is reported": {
			text: "x" + tagA,
			want: []hidden.Hit{{Offset: 1, Line: 1, Class: hidden.ClassTag, Rune: 0xE0041}},
		},
		"a flag with a different region code is not exempt": {
			text: "\U0001F3F4\U000E0078\U000E007F",
			want: []hidden.Hit{
				{Offset: 4, Line: 1, Class: hidden.ClassTag, Rune: 0xE0078},
				{Offset: 8, Line: 1, Class: hidden.ClassTag, Rune: 0xE007F},
			},
		},
		"soft hyphen and marks are listed": {
			text: shy + lrm,
			want: []hidden.Hit{
				{Offset: 0, Line: 1, Class: hidden.ClassZeroWidth, Rune: 0x00AD},
				{Offset: 2, Line: 1, Class: hidden.ClassBidi, Rune: 0x200E},
			},
		},
		"invalid UTF-8 is not hidden": {text: "a\xffb"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := hidden.Scan(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Scan = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestByClass(t *testing.T) {
	t.Parallel()
	got := hidden.ByClass(hidden.Scan("a" + zwsp + "b" + zwsp + rlo))
	want := []hidden.Summary{
		{Class: hidden.ClassBidi, Rune: "U+202E", Offset: 8, Count: 1},
		{Class: hidden.ClassZeroWidth, Rune: "U+200B", Offset: 1, Count: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ByClass = %+v, want %+v", got, want)
	}
	if clean := hidden.ByClass(nil); clean == nil || len(clean) != 0 {
		t.Errorf("ByClass(nil) = %#v, want an empty non-nil slice", clean)
	}
}

func TestStrip(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		text     string
		want     string
		origOf   int // an offset in the stripped text
		wantOrig int // where it was in the original
	}{
		"soft hyphen inside a word": {
			text: "ig" + shy + "nore all", want: "ignore all", origOf: 2, wantOrig: 4,
		},
		"joiners are stripped whatever their context": {
			text: family, want: "\U0001F468\U0001F469\U0001F467", origOf: 4, wantOrig: 7,
		},
		"end of text maps to end of text": {
			text: "ab" + zwsp, want: "ab", origOf: 2, wantOrig: 5,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := hidden.Strip(tc.text)
			if s.Text != tc.want {
				t.Errorf("Strip = %q, want %q", s.Text, tc.want)
			}
			if got := s.Original(tc.origOf); got != tc.wantOrig {
				t.Errorf("Original(%d) = %d, want %d", tc.origOf, got, tc.wantOrig)
			}
		})
	}
}

func TestStripProperties(t *testing.T) {
	t.Parallel()
	// Mix arbitrary strings with listed codepoints so the property is not trivially
	// satisfied by text that never held any.
	listed := []string{zwsp, zwj, zwnj, bom, shy, rlo, lrm, tagA}
	property := func(parts []string, picks []uint8) bool {
		var b strings.Builder
		for i, p := range parts {
			b.WriteString(p)
			if i < len(picks) {
				b.WriteString(listed[int(picks[i])%len(listed)])
			}
		}
		s := hidden.Strip(b.String())
		if hidden.Strip(s.Text).Text != s.Text {
			return false // not idempotent
		}
		if len(hidden.Scan(s.Text)) != 0 {
			return false // something hidden survived
		}
		for i := range len(s.Text) {
			if o := s.Original(i); b.String()[o] != s.Text[i] {
				return false // offset map points at a different byte
			}
		}
		return true
	}
	if err := quick.Check(property, nil); err != nil {
		t.Error(err)
	}
}
