package neutrality_test

import (
	"testing"

	"github.com/StevenACoffman/skillet/neutrality"
)

func TestScanMatchesEachRedLight(t *testing.T) {
	t.Parallel()
	// One representative line per red-light alternative; each must produce a hit.
	lines := []string{
		"这在 Claude Code 里运行",
		"This is a Claude Code skill",
		"针对 Claude Code 用户",
		"Cursor only feature",
		"在 Codex 中使用",
		"[![Claude Code](badge.svg)]",
		"copy to ~/.claude/skills/foo",
		"run /plugin install bar",
	}
	for _, ln := range lines {
		t.Run(ln, func(t *testing.T) {
			t.Parallel()
			hits := neutrality.Scan([]neutrality.NamedFile{{Name: "SKILL.md", Content: ln}})
			if len(hits) != 1 {
				t.Fatalf("Scan(%q) = %d hits, want 1", ln, len(hits))
			}
		})
	}
}

func TestScanCleanContent(t *testing.T) {
	t.Parallel()
	content := "# Title\n\nThis is a runtime-neutral skill.\nInstall it anywhere.\n"
	hits := neutrality.Scan([]neutrality.NamedFile{{Name: "SKILL.md", Content: content}})
	if len(hits) != 0 {
		t.Fatalf("clean content produced %d hits: %+v", len(hits), hits)
	}
}

func TestScanAnchorIsLineStart(t *testing.T) {
	t.Parallel()
	// The "^[![Claude Code" alternative is anchored to line start: a mid-line
	// occurrence must NOT match; a line-start one must.
	midLine := "see the [![Claude Code](x)] badge"
	midHits := neutrality.Scan([]neutrality.NamedFile{{Name: "f", Content: midLine}})
	if len(midHits) != 0 {
		t.Fatalf("mid-line badge matched: %+v", midHits)
	}
	atStart := "[![Claude Code](x)]"
	startHits := neutrality.Scan([]neutrality.NamedFile{{Name: "f", Content: atStart}})
	if len(startHits) != 1 {
		t.Fatalf("line-start badge did not match: got %d hits", len(startHits))
	}
}

func TestScanLineNumbersAndCRLF(t *testing.T) {
	t.Parallel()
	// CRLF endings; the hit lands on line 3 with a trimmed Text.
	content := "clean line\r\nanother clean\r\n  Cursor only  \r\nlast\r\n"
	hits := neutrality.Scan([]neutrality.NamedFile{{Name: "SKILL.md", Content: content}})
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(hits))
	}
	if hits[0].Line != 3 {
		t.Errorf("Line = %d, want 3", hits[0].Line)
	}
	if hits[0].Text != "Cursor only" {
		t.Errorf("Text = %q, want trimmed %q", hits[0].Text, "Cursor only")
	}
	if hits[0].File != "SKILL.md" {
		t.Errorf("File = %q, want SKILL.md", hits[0].File)
	}
}

func TestScanMultiFileOrdering(t *testing.T) {
	t.Parallel()
	files := []neutrality.NamedFile{
		{Name: "a.md", Content: "Cursor only\nclean"},
		{Name: "b.md", Content: "clean\nCodex 中"},
	}
	hits := neutrality.Scan(files)
	if len(hits) != 2 {
		t.Fatalf("want 2 hits, got %d", len(hits))
	}
	// Order: file order first, then line number.
	if hits[0].File != "a.md" || hits[0].Line != 1 {
		t.Errorf("hit0 = %+v, want a.md:1", hits[0])
	}
	if hits[1].File != "b.md" || hits[1].Line != 2 {
		t.Errorf("hit1 = %+v, want b.md:2", hits[1])
	}
}
