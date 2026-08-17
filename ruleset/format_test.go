package ruleset_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/ruleset"
)

const v1Doc = "Source: s\nScope:  x\n\n§1.1  [MUST][CODE]  Close it.\n      because reasons\n"

// TestUndeclaredFormatIsVersionOne is the migration story: every ruleset written before
// versioning existed reads as v1 without being touched.
func TestUndeclaredFormatIsVersionOne(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rs.Format != 1 {
		t.Errorf("Format = %d, want 1 for a file declaring none", rs.Format)
	}
}

// TestVersionOneRendersNoBlock is the property the whole change rests on. If Render started
// emitting a block, every stored ruleset would report drift the moment canonizer's
// canonical-form check lands — the new feature manufacturing the failure the next feature
// exists to detect.
func TestVersionOneRendersNoBlock(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse(v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out := ruleset.Render(rs)
	if strings.Contains(out, "---") {
		t.Errorf("v1 rendered a version block:\n%s", out)
	}
	if out != v1Doc {
		t.Errorf("v1 did not round-trip byte-identically:\n got %q\nwant %q", out, v1Doc)
	}
	// A hand-built Ruleset that never set Format must render as v1 too, or every Go caller
	// has to remember a field that has one sensible value.
	zero := rs
	zero.Format = 0
	if ruleset.Render(zero) != out {
		t.Error("Format 0 and Format 1 rendered differently")
	}
}

// TestFutureFormatIsRefused is the behaviour being bought. Without it an unknown marker in a
// newer file is folded into a rule's rationale and nothing says so.
func TestFutureFormatIsRefused(t *testing.T) {
	t.Parallel()
	future := "---\nformat: " + strconv.Itoa(ruleset.FormatVersion+1) + "\n---\n" + v1Doc
	_, err := ruleset.Parse(future)
	if err == nil {
		t.Fatal("a newer format parsed without complaint")
	}
	// The message must name both versions: "it failed" does not tell an operator whether to
	// upgrade the tool or fix the file.
	for _, want := range []string{strconv.Itoa(ruleset.FormatVersion + 1), strconv.Itoa(ruleset.FormatVersion)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name version %s", err, want)
		}
	}
}

func TestMalformedFormatIsAnError(t *testing.T) {
	t.Parallel()
	for name, block := range map[string]string{
		"negative":     "---\nformat: -1\n---\n",
		"not a number": "---\nformat: two\n---\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := ruleset.Parse(block + v1Doc); err == nil {
				t.Error("parsed without complaint")
			}
		})
	}
}

// TestBlockWithoutFormatIsVersionOne keeps the door open for other metadata: a block that
// declares no format is a v1 ruleset carrying something else, not a malformed version.
func TestBlockWithoutFormatIsVersionOne(t *testing.T) {
	t.Parallel()
	rs, err := ruleset.Parse("---\nnote: hello\n---\n" + v1Doc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if rs.Format != 1 {
		t.Errorf("Format = %d, want 1", rs.Format)
	}
}
