package claims_test

import (
	"go/build"
	"reflect"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/claims"
	"github.com/StevenACoffman/skillet/markdown"
)

func TestCommands(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		span string
		want bool
	}{
		"a make target":        {"make verify", true},
		"a tool subcommand":    {"bw sync --force", true},
		"a go invocation":      {"go test ./...", true},
		"an explicit path":     {"./scripts/build.sh --fast", true},
		"a hyphenated program": {"golangci-lint run", true},
		// Excluded, and each is a class rather than an example.
		"a filename":         {"SKILL.md", false},
		"a bare program":     {"go", false},
		"a function call":    {"foo.Bar()", false},
		"an assignment":      {"x = 1", false},
		"a comparison":       {"n == 3", false},
		"a Go declaration":   {"var Doc struct", false},
		"a type declaration": {"type Foo interface", false},
		// Lost to the same rule, and recorded so the cost is visible rather than
		// discovered. Shell commands are conventionally lowercase; this is the price.
		"a command with a capitalised argument": {"git push origin HEAD", false},
		"a flag alone":                          {"--force", false},
		"empty":                                 {"", false},
		"whitespace only":                       {"   ", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := claims.Commands([]string{tc.span})
			if found := len(got) == 1; found != tc.want {
				t.Errorf("Commands(%q) = %v, want found: %v", tc.span, got, tc.want)
			}
		})
	}
}

// TestCommandsIsDeduplicatedAndSorted, so two runs over one document agree and a caller
// can diff two documents' claims.
func TestCommandsIsDeduplicatedAndSorted(t *testing.T) {
	t.Parallel()
	got := claims.Commands([]string{
		"make verify", "go test ./...", "make verify", "bw sync now",
	})
	want := []string{"bw sync now", "go test ./...", "make verify"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestNoCommandsIsEmptyNotNil, so a caller need not distinguish "found none" from
// "did not look".
func TestNoCommandsIsEmptyNotNil(t *testing.T) {
	t.Parallel()
	if got := claims.Commands(nil); got == nil {
		t.Error("Commands(nil) returned nil")
	}
}

// TestReadsWhatMarkdownProduces. The unit cases above are literals, so nothing yet
// checks that the shape markdown actually emits is the shape this reads — the join
// that, if wrong, would make every document claim nothing.
func TestReadsWhatMarkdownProduces(t *testing.T) {
	t.Parallel()
	doc := markdown.Parse("# S\n\nRun `make verify` before `git push`, see `SKILL.md`.\n" +
		"Then `go test ./...` passes.\n")

	got := claims.Commands(doc.CodeSpans)
	want := []string{"git push", "go test ./...", "make verify"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v (spans %v)", got, want, doc.CodeSpans)
	}
}

// TestCodeInsideAFenceIsNotAClaim. A fenced block is an example, not an assertion that
// the repository supports what it shows, and CodeSpans is inline spans only.
func TestCodeInsideAFenceIsNotAClaim(t *testing.T) {
	t.Parallel()
	doc := markdown.Parse("# S\n\n```sh\nmake nonexistent-target\n```\n")

	if got := claims.Commands(doc.CodeSpans); len(got) != 0 {
		t.Errorf("a fenced example was read as a claim: %v", got)
	}
}

// TestImportsNothing is the environment boundary, checked rather than promised. The
// entire reason the promotion splits here is that resolving a command is
// environment-dependent; a package that grew an os or exec import would have moved the
// boundary without anybody deciding to.
func TestImportsNothing(t *testing.T) {
	t.Parallel()
	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatalf("read the package: %v", err)
	}
	for _, imp := range pkg.Imports {
		if strings.Contains(imp, ".") {
			t.Errorf("claims imports %q; it must depend on the standard library only", imp)
		}
		switch imp {
		case "os", "os/exec", "path/filepath", "net", "net/http":
			t.Errorf("claims imports %q, which is the environment this package must not touch", imp)
		}
	}
}

// TestAGoStatementIsNotACommand. Each of these came from the head of the false-positive
// list over 233 real skill bodies: a lowercase first token, which is all the executable
// pattern examines, followed by a call expression.
func TestAGoStatementIsNotACommand(t *testing.T) {
	t.Parallel()
	for _, span := range []string{
		"defer cancel()", "go func()", "getenv func(string) string", "close(ch)",
	} {
		if got := claims.Commands([]string{span}); len(got) != 0 {
			t.Errorf("Commands(%q) = %v, want none", span, got)
		}
	}
}
