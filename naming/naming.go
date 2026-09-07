// Package naming derives filenames and human titles for the distillation
// pipeline: a source markdown file's title, and the *_rules.md / *_prompt.md
// output names beside it. Every function is pure except TitleFromFile, which
// reads the file and delegates to the pure helpers.
package naming

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var reWordSep = regexp.MustCompile(`[\s\-]+`)

// Title converts a filename stem into a readable label: runs of space, underscore and
// hyphen become single spaces, and nothing else changes.
//
// **It does not capitalize, and that is the point.** It used to upper-case the first letter
// of every word, which meant inventing a spelling it had no way to know. Measured over 342
// skill slugs it got **78 (22%) wrong** -- 76 of them acronyms (`composite-slo` became
// `Composite Slo`) and 3 project names (`climax-cli-scaffold` became `Climax Cli Scaffold`).
// A lowercase stem carries no capitalization to recover, so any capital produced here is
// guessed. A stem -> canonical-title table was the alternative and was refused on
// 2026-09-07: it needed ~78 entries on the day it shipped and one more per skill after,
// and no authored display title exists anywhere in the corpus to fill it from -- 273 of 288
// skills declare `name:` as the slug itself. See skillet's TODO.md.
//
// The name is unchanged because the package's other Title* functions return a document's
// real title and this one still answers "the title for this stem"; what changed is that it
// no longer claims to know its casing.
//
// Requires: stem is a filename stem; separators are whitespace, underscore or hyphen.
// Ensures:  pure. Separator runs collapse to one space, case is never altered, and leading
//
//	or trailing separators produce no leading or trailing space.
func Title(stem string) string {
	return strings.Join(strings.FieldsFunc(stem, isSeparator), " ")
}

// isSeparator reports whether r separates words in a filename stem.
//
// FieldsFunc rather than a regexp split because it drops empty fields: the old split kept
// them, so `Title("-a")` returned a leading space. Fixed in passing rather than preserved.
func isSeparator(r rune) bool {
	return unicode.IsSpace(r) || r == '_' || r == '-'
}

// RulesFilename derives the destination rules filename from a source filename:
// "My-Source File.md" -> "my_source_file_rules.md".
func RulesFilename(name string) string {
	ext := filepath.Ext(name)
	base := strings.ToLower(strings.TrimSuffix(name, ext))
	base = reWordSep.ReplaceAllString(base, "_")
	return base + "_rules" + ext
}

// PromptFilename derives the prompt filename from a source filename:
// "self-refinement.md" -> "self-refinement_prompt.md".
func PromptFilename(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext) + "_prompt" + ext
}

// TitleFromMarkdown returns the text of the first H1 ("# ") heading in content,
// or "" if there is none.
func TitleFromMarkdown(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// TitleFromFile returns the first H1 heading of the markdown file at path, or a
// title derived from the filename stem when the file has no H1.
func TitleFromFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("naming: read %s: %w", path, err)
	}
	if title := TitleFromMarkdown(string(b)); title != "" {
		return title, nil
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Title(stem), nil
}
