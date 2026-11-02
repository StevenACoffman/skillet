package speclint_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/skill"
	"github.com/StevenACoffman/skillet/speclint"
)

// want is one expected diagnostic: its severity and a substring of its message.
type want struct {
	sev finding.Severity
	sub string
}

func TestAllowedFrontmatterKey(t *testing.T) {
	// The spec's own set, plus the documented `tags` deviation.
	for _, k := range []string{
		"name", "description", "license", "compatibility", "metadata", "allowed-tools", "tags",
	} {
		if !speclint.AllowedFrontmatterKey(k) {
			t.Errorf("AllowedFrontmatterKey(%q) = false, want true", k)
		}
	}
	// `author` and `version` are the two most commonly assumed keys and are not in
	// the spec at any level — the spec's own example carries them inside `metadata`.
	for _, k := range []string{"", "Name", "author", "version", "tools", "id", "type"} {
		if speclint.AllowedFrontmatterKey(k) {
			t.Errorf("AllowedFrontmatterKey(%q) = true, want false", k)
		}
	}
}

func errorWith(sub string) want { return want{finding.SeverityError, sub} }
func warnWith(sub string) want  { return want{finding.SeverityWarning, sub} }

// load writes frontmatter and a one-line body as <tmp>/<dir>/SKILL.md and loads it, so
// each case exercises the same parse a real skill goes through.
func load(t *testing.T, dir, frontmatter string) *skill.Skill {
	t.Helper()
	path := filepath.Join(t.TempDir(), dir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\n" + frontmatter + "---\nbody\n"
	if err := os.WriteFile(
		filepath.Join(path, skill.FileName),
		[]byte(content),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	s, err := skill.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestFrontmatter(t *testing.T) {
	t.Parallel()
	overLong := strings.Repeat("x", speclint.DescriptionMaxRunes+1)
	atLimit := strings.Repeat("y", speclint.DescriptionMaxRunes)
	atTarget := strings.Repeat("z", speclint.DescriptionTargetRunes)
	overTarget := strings.Repeat("z", speclint.DescriptionTargetRunes+1)
	longName := strings.Repeat("a", speclint.NameMaxRunes+1)

	cases := map[string]struct {
		dir         string
		frontmatter string
		want        []want // in emission order; nil = no diagnostics
	}{
		"clean frontmatter": {
			dir:         "x",
			frontmatter: "name: x\ndescription: does x, use when y\ntags: [a]\n",
		},
		"unknown key is a warning that points at metadata": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\nauthor: me\n",
			want:        []want{warnWith(`"author" is not in the Agent Skills spec`)},
		},
		"Claude Code key is a warning that names Claude Code": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\ndisable-model-invocation: true\n",
			want:        []want{warnWith("Claude Code extension")},
		},
		"empty description": {
			dir:         "x",
			frontmatter: "name: x\ndescription: \"\"\n",
			want:        []want{errorWith("description is empty")},
		},
		"description over the cap": {
			dir:         "x",
			frontmatter: "name: x\ndescription: " + overLong + "\n",
			want:        []want{errorWith("runes > 1024")},
		},
		"description at the cap is over the target only": {
			dir:         "x",
			frontmatter: "name: x\ndescription: " + atLimit + "\n",
			want:        []want{warnWith("> 300 target")},
		},
		"description at the target is fine": {
			dir:         "x",
			frontmatter: "name: x\ndescription: " + atTarget + "\n",
		},
		"description one over the target warns": {
			dir:         "x",
			frontmatter: "name: x\ndescription: " + overTarget + "\n",
			want:        []want{warnWith("301 runes > 300 target")},
		},
		"angle brackets are an Anthropic platform rule": {
			dir:         "x",
			frontmatter: "name: x\ndescription: wrap <tag> here\n",
			want:        []want{errorWith("Anthropic platform rule")},
		},
		"non-string description is a type error, not empty": {
			dir:         "x",
			frontmatter: "name: x\ndescription: [a, b]\n",
			want:        []want{errorWith("description must be a string, got a list")},
		},
		"missing name": {
			dir:         "x",
			frontmatter: "description: d\n",
			want:        []want{errorWith(`name "" is empty`)},
		},
		"non-string name is a type error": {
			dir:         "x",
			frontmatter: "name: 7\ndescription: d\n",
			want:        []want{errorWith("name must be a string, got a number")},
		},
		"name over the length cap": {
			dir:         longName,
			frontmatter: "name: " + longName + "\ndescription: d\n",
			want:        []want{errorWith("65 characters > 64")},
		},
		"uppercase in a name": {
			dir:         "Abc",
			frontmatter: "name: Abc\ndescription: d\n",
			want:        []want{errorWith("only a-z, 0-9 and hyphens")},
		},
		"non-ASCII letter in a name": {
			dir:         "caf\u00e9",
			frontmatter: "name: caf\u00e9\ndescription: d\n",
			want:        []want{errorWith("only a-z, 0-9 and hyphens")},
		},
		"leading hyphen": {
			dir:         "-x",
			frontmatter: "name: -x\ndescription: d\n",
			want:        []want{errorWith("starts or ends with a hyphen")},
		},
		"consecutive hyphens": {
			dir:         "a--b",
			frontmatter: "name: a--b\ndescription: d\n",
			want:        []want{errorWith("consecutive hyphens")},
		},
		"name differs from its folder": {
			dir:         "y",
			frontmatter: "name: x\ndescription: d\n",
			want:        []want{errorWith(`name "x" != folder "y"`)},
		},
		"reserved word is a warning": {
			dir:         "claude-tips",
			frontmatter: "name: claude-tips\ndescription: d\n",
			want:        []want{warnWith(`reserved word "claude"`)},
		},
		"compatibility over its cap": {
			dir: "x",
			frontmatter: "name: x\ndescription: d\ncompatibility: " +
				strings.Repeat("c", speclint.CompatibilityMaxRunes+1) + "\n",
			want: []want{errorWith("compatibility 501 runes > 500")},
		},
		"compatibility must be a string": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\ncompatibility: [a]\n",
			want:        []want{errorWith("compatibility must be a string")},
		},
		"metadata values must be strings": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\nmetadata:\n  ok: yes-string\n  n: 3\n",
			want:        []want{errorWith("metadata.n must be a string, got a number")},
		},
		"metadata must be a mapping": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\nmetadata: [a]\n",
			want:        []want{errorWith("metadata must be a mapping")},
		},
		"allowed-tools must be a string": {
			dir:         "x",
			frontmatter: "name: x\ndescription: d\nallowed-tools: [Read, Bash]\n",
			want:        []want{errorWith("allowed-tools must be a string")},
		},
		"several findings at once keep their severities": {
			dir:         "x",
			frontmatter: "name: x\ndescription: <x>\nbogus: 1\n",
			want: []want{
				warnWith(`"bogus" is not in the Agent Skills spec`),
				errorWith("angle brackets"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertDiagnostics(t, speclint.Frontmatter(load(t, tc.dir, tc.frontmatter)), tc.want)
		})
	}
}

func TestValidSkillName(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"a":                     true,
		"pdf-processing":        true,
		"v2-api":                true,
		"":                      false,
		"Pdf":                   false,
		"a.b":                   false, // a plugin name may contain "."; a skill name may not
		"a_b":                   false,
		"-a":                    false,
		"a-":                    false,
		"a--b":                  false,
		"caf\u00e9":             false,
		strings.Repeat("a", 64): true,
		strings.Repeat("a", 65): false,
	}
	for name, valid := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := speclint.ValidSkillName(name); got != valid {
				t.Errorf("ValidSkillName(%q) = %v, want %v", name, got, valid)
			}
		})
	}
}

// assertDiagnostics checks that got holds exactly the wanted diagnostics, in order.
func assertDiagnostics(t *testing.T, got []finding.Diagnostic, wants []want) {
	t.Helper()
	if len(got) != len(wants) {
		t.Fatalf("got %d diagnostics, want %d: %+v", len(got), len(wants), got)
	}
	for i, w := range wants {
		if got[i].Severity != w.sev {
			t.Errorf(
				"diagnostic %d severity = %q, want %q (%s)",
				i,
				got[i].Severity,
				w.sev,
				got[i].Message,
			)
		}
		if !strings.Contains(got[i].Message, w.sub) {
			t.Errorf("diagnostic %d = %q, want substring %q", i, got[i].Message, w.sub)
		}
	}
}

func TestFrontmatterReportsAParseFailureAsItself(t *testing.T) {
	t.Parallel()
	// The shape found in the real book trees: a quoted scalar followed by unquoted
	// text. Before this, the same skill produced "description is empty" and a
	// missing name — both symptoms, neither the cause.
	s := &skill.Skill{FrontmatterErr: errors.New("[3:45] value is not allowed in this context")}

	got := speclint.Frontmatter(s)
	if len(got) != 1 {
		t.Fatalf("a parse failure must be the only diagnostic, got %+v", got)
	}
	if !strings.Contains(got[0].Message, "not valid YAML") {
		t.Errorf("expected the cause to be named, got %q", got[0].Message)
	}
	if !strings.Contains(got[0].Message, "[3:45]") {
		t.Errorf("expected the underlying error to be carried through, got %q", got[0].Message)
	}
	for _, d := range got {
		if strings.Contains(d.Message, "description is empty") {
			t.Error("must not report the symptom alongside the cause")
		}
	}
}

func TestFrontmatterUnaffectedWhenParseSucceeded(t *testing.T) {
	t.Parallel()
	// FrontmatterErr nil is the ordinary path: the other checks still apply.
	s := load(t, "x", "name: x\ndescription: fine\nnope: 1\n")
	got := speclint.Frontmatter(s)
	if len(got) != 1 ||
		!strings.Contains(got[0].Message, `"nope" is not in the Agent Skills spec`) {
		t.Fatalf("expected the unknown-key diagnostic, got %+v", got)
	}
}

func TestLines(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in   string
		want int
	}{
		"empty":                 {"", 0},
		"one line with newline": {"a\n", 1},
		"one line without":      {"a", 1},
		"two lines":             {"a\nb\n", 2},
		"trailing line without": {"a\nb", 2},
		"blank lines count":     {"\n\n", 2},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := speclint.Lines(tc.in); got != tc.want {
				t.Errorf("Lines(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestFrontmatterReportsAnUnclosedHeader(t *testing.T) {
	t.Parallel()
	// The shape an automated rewrite left in twelve skills: the closing delimiter glued
	// to the last value. Reported as itself, not as an empty name and description.
	dir := filepath.Join(t.TempDir(), "x")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := "---\nname: x\ndescription: \"d\"---\n# Title\n"
	if err := os.WriteFile(filepath.Join(dir, skill.FileName), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := skill.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := speclint.Frontmatter(s)
	if len(got) != 1 || !strings.Contains(got[0].Message, "not closed") {
		t.Fatalf("want one not-closed diagnostic, got %+v", got)
	}
}
