// Package distill generates the per-source distillation prompts: for each
// Markdown source under a tree it fills a template with links to the source and
// to the rules file it should produce, and writes a *_prompt.md beside the
// output directory. FillTemplate is pure; Generate walks the tree and writes.
package distill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/StevenACoffman/skillet/naming"
)

// The template must contain these two placeholders; Generate replaces them with
// Markdown links to the source and its rules file.
const (
	sourcePlaceholder = "<source>{{SOURCE_CONTENT}}</source>"
	destPlaceholder   = "<destination>{{DESTINATION_CONTENT}}</destination>"
)

// generator accumulates the prompts written while walking a source tree.
type generator struct {
	tmpl    string
	absOut  string
	root    string
	written []string
}

// FillTemplate replaces the source and destination placeholders in tmpl. It
// returns an error if either placeholder is absent, so a misconfigured template
// fails loudly rather than emitting a prompt that references nothing.
func FillTemplate(tmpl, sourceLink, destLink string) (string, error) {
	if !strings.Contains(tmpl, sourcePlaceholder) {
		return "", fmt.Errorf("distill: template missing %s", sourcePlaceholder)
	}
	if !strings.Contains(tmpl, destPlaceholder) {
		return "", fmt.Errorf("distill: template missing %s", destPlaceholder)
	}
	out := strings.ReplaceAll(tmpl, sourcePlaceholder, sourceLink)
	return strings.ReplaceAll(out, destPlaceholder, destLink), nil
}

// Generate walks sourceRoot for Markdown sources (skipping *_rules.md,
// *_prompt.md, and hidden directories), fills tmpl for each, and writes a
// *_prompt.md into promptOutDir. It returns the written prompt paths in walk
// order. The template is validated once before any file is written.
func Generate(tmpl, sourceRoot, promptOutDir string) ([]string, error) {
	if _, err := FillTemplate(tmpl, "", ""); err != nil {
		return nil, err
	}
	absOut, err := filepath.Abs(promptOutDir)
	if err != nil {
		return nil, fmt.Errorf("distill: resolve out dir: %w", err)
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return nil, fmt.Errorf("distill: create out dir: %w", err)
	}
	g := &generator{tmpl: tmpl, absOut: absOut, root: sourceRoot}
	if err := filepath.WalkDir(sourceRoot, g.walk); err != nil {
		return nil, fmt.Errorf("distill: walk %s: %w", sourceRoot, err)
	}
	return g.written, nil
}

// walk is the filepath.WalkDir callback: it fills and writes a prompt for each
// source file, skipping hidden directories and generated rules/prompt files.
func (g *generator) walk(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		if path != g.root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		return nil
	}
	if !isSource(path) {
		return nil
	}
	prompt, genErr := generateOne(g.tmpl, path, g.absOut)
	if genErr != nil {
		return genErr
	}
	g.written = append(g.written, prompt)
	return nil
}

// isSource reports whether path is a source Markdown file (not a generated rules
// or prompt file).
func isSource(path string) bool {
	return strings.HasSuffix(path, ".md") &&
		!strings.HasSuffix(path, "_rules.md") &&
		!strings.HasSuffix(path, "_prompt.md")
}

func generateOne(tmpl, source, absOut string) (string, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("distill: resolve source: %w", err)
	}
	title, err := naming.TitleFromFile(absSource)
	if err != nil {
		return "", fmt.Errorf("distill: %w", err)
	}
	base := filepath.Base(absSource)
	destPath := filepath.Join(filepath.Dir(absSource), naming.RulesFilename(base))

	sourceLink, err := link(absOut, absSource, title)
	if err != nil {
		return "", err
	}
	destLink, err := link(absOut, destPath, title+" Rules")
	if err != nil {
		return "", err
	}
	filled, err := FillTemplate(tmpl, sourceLink, destLink)
	if err != nil {
		return "", err
	}
	promptPath := filepath.Join(absOut, naming.PromptFilename(base))
	if err := os.WriteFile(promptPath, []byte(filled), 0o644); err != nil {
		return "", fmt.Errorf("distill: write %s: %w", promptPath, err)
	}
	return promptPath, nil
}

// link builds a Markdown link to target relative to the prompt's directory
// (base), using forward slashes so the link is portable.
func link(base, target, text string) (string, error) {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("distill: relative path to %s: %w", target, err)
	}
	return fmt.Sprintf("[%s](%s)", text, filepath.ToSlash(rel)), nil
}
