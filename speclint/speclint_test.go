package speclint_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/StevenACoffman/skillet/finding"
	"github.com/StevenACoffman/skillet/skill"
	"github.com/StevenACoffman/skillet/speclint"
)

func TestAllowedFrontmatterKey(t *testing.T) {
	for _, k := range []string{"name", "description", "tags", "allowed-tools"} {
		if !speclint.AllowedFrontmatterKey(k) {
			t.Errorf("AllowedFrontmatterKey(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"", "Name", "author", "license", "tools"} {
		if speclint.AllowedFrontmatterKey(k) {
			t.Errorf("AllowedFrontmatterKey(%q) = true, want false", k)
		}
	}
}

func TestFrontmatter(t *testing.T) {
	overLong := strings.Repeat("x", speclint.DescriptionMaxRunes+1)
	atLimit := strings.Repeat("y", speclint.DescriptionMaxRunes)

	tests := []struct {
		name     string
		skill    skill.Skill
		wantSubs []string // one substring per expected diagnostic; nil = no diagnostics
	}{
		{
			name: "clean frontmatter",
			skill: skill.Skill{
				FrontmatterKeys: []string{"description", "name", "tags"},
				Description:     "does x, use when y",
			},
		},
		{
			name:     "disallowed key",
			skill:    skill.Skill{FrontmatterKeys: []string{"author", "name"}, Description: "d"},
			wantSubs: []string{`disallowed key "author"`},
		},
		{
			name:     "empty description",
			skill:    skill.Skill{FrontmatterKeys: []string{"name"}, Description: ""},
			wantSubs: []string{"description is empty"},
		},
		{
			name:     "description over the cap",
			skill:    skill.Skill{FrontmatterKeys: []string{"name"}, Description: overLong},
			wantSubs: []string{"runes > 1024"},
		},
		{
			name:  "description exactly at the cap is fine",
			skill: skill.Skill{FrontmatterKeys: []string{"name"}, Description: atLimit},
		},
		{
			name: "angle brackets are not plain text",
			skill: skill.Skill{
				FrontmatterKeys: []string{"name"},
				Description:     "wrap <tag> here",
			},
			wantSubs: []string{"angle brackets"},
		},
		{
			name:     "several violations at once",
			skill:    skill.Skill{FrontmatterKeys: []string{"name", "bogus"}, Description: "<x>"},
			wantSubs: []string{"disallowed key", "angle brackets"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertDiagnostics(t, speclint.Frontmatter(&tc.skill), tc.wantSubs)
		})
	}
}

// assertDiagnostics checks that got holds one error-severity diagnostic per
// wantSub, each carrying that substring in its Message.
func assertDiagnostics(t *testing.T, got []finding.Diagnostic, wantSubs []string) {
	t.Helper()
	if len(got) != len(wantSubs) {
		t.Fatalf("got %d diagnostics, want %d: %+v", len(got), len(wantSubs), got)
	}
	for i, want := range wantSubs {
		if got[i].Severity != finding.SeverityError {
			t.Errorf("diagnostic %d severity = %q, want error", i, got[i].Severity)
		}
		if !strings.Contains(got[i].Message, want) {
			t.Errorf("diagnostic %d = %q, want substring %q", i, got[i].Message, want)
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
	s := &skill.Skill{FrontmatterKeys: []string{"name", "nope"}, Description: "fine"}
	got := speclint.Frontmatter(s)
	if len(got) != 1 || !strings.Contains(got[0].Message, `disallowed key "nope"`) {
		t.Fatalf("expected the disallowed-key diagnostic, got %+v", got)
	}
}
