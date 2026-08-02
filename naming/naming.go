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
)

var (
	reWordSep    = regexp.MustCompile(`[\s\-]+`)
	reAllWordSep = regexp.MustCompile(`[\s_\-]+`)
)

// Title converts a filename stem into a readable title:
// "my-source_file" -> "My Source File".
func Title(stem string) string {
	words := reAllWordSep.Split(stem, -1)
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
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
