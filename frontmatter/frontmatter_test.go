package frontmatter_test

import (
	"testing"

	"github.com/StevenACoffman/skillet/frontmatter"
)

func TestSplit(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in        string
		wantBlock string
		wantBody  string
		why       string
	}{
		"a well-formed header": {
			in:        "---\ntitle: x\n---\nbody\n",
			wantBlock: "title: x",
			wantBody:  "body\n",
		},
		"no header at all leaves the document whole": {
			in:       "just body\n",
			wantBody: "just body\n",
		},
		// The closing delimiter is the first line after the opening one, so it has no
		// newline before it; a "\n---" substring search misses it and leaks the
		// delimiter into the body.
		"an empty header is still a header": {
			in:       "---\n---\nbody\n",
			wantBody: "body\n",
			why:      "the closing delimiter must not leak into the body",
		},
		// Returning the remainder would drop the opening line from a malformed file.
		"an unterminated header is not a header": {
			in:       "---\ntitle: x\nno close\n",
			wantBody: "---\ntitle: x\nno close\n",
			why:      "no header means the body is the entire document",
		},
		"CRLF endings are found and normalized": {
			in:        "---\r\ntitle: x\r\n---\r\nbody\r\n",
			wantBlock: "title: x",
			wantBody:  "body\n",
			why:       "a caller that forgets to normalize gets no error, just empty fields",
		},
		"no trailing newline after the close": {
			in:        "---\ntitle: x\n---",
			wantBlock: "title: x",
		},
		"a longer rule still closes the block": {
			in:        "---\ntitle: x\n----\nbody\n",
			wantBlock: "title: x",
			wantBody:  "body\n",
		},
		"trailing text on the closing line is dropped with it": {
			in:        "---\ntitle: x\n--- ignored\nbody\n",
			wantBlock: "title: x",
			wantBody:  "body\n",
		},
		"a leading blank line means no header": {
			in:       "\n---\ntitle: x\n---\nbody\n",
			wantBody: "\n---\ntitle: x\n---\nbody\n",
		},
		"a multi-line header keeps its shape": {
			in:        "---\ncheck: r-quote-accuracy\nsources:\n  - book: a\n---\nbody\n",
			wantBlock: "check: r-quote-accuracy\nsources:\n  - book: a",
			wantBody:  "body\n",
		},
		"dashes inside the body are not a delimiter": {
			in:        "---\ntitle: x\n---\nbody\n--- not a header\n",
			wantBlock: "title: x",
			wantBody:  "body\n--- not a header\n",
		},
		"an empty document": {},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			block, body := frontmatter.Split(tc.in)
			if block != tc.wantBlock {
				t.Errorf("block = %q, want %q (%s)", block, tc.wantBlock, tc.why)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q (%s)", body, tc.wantBody, tc.why)
			}
		})
	}
}

func TestSplitNeverLeavesADelimiterInTheBody(t *testing.T) {
	t.Parallel()
	// The contract a caller relies on: whatever it hands back as the body is markdown,
	// never a stray fragment of the header.
	for _, in := range []string{
		"---\n---\n",
		"---\n---\nbody\n",
		"---\na: 1\n---\n",
		"---\n\n---\nbody\n",
	} {
		_, body := frontmatter.Split(in)
		if len(body) >= 3 && body[:3] == "---" {
			t.Errorf("Split(%q) left a delimiter at the head of the body: %q", in, body)
		}
	}
}
