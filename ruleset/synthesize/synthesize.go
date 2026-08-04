// Package synthesize assembles distilled rulesets into a single synthesis prompt:
// it replaces a template's {{RULESETS}} marker with one <ruleset> block per input.
// FillTemplate is pure; LoadInputs reads a directory of *_rules.md files. It is the
// sibling of ruleset/distill: distill fans a source tree out into per-source
// prompts, synthesize folds the resulting rulesets back into one.
package synthesize

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/StevenACoffman/skillet/naming"
	errors "github.com/StevenACoffman/toerr/errors"
)

// Marker is the placeholder a synthesis template must contain; FillTemplate
// replaces it with the assembled <ruleset> blocks.
const Marker = "{{RULESETS}}"

// rulesSuffix is the filename suffix LoadInputs treats as a distilled ruleset,
// matching naming.RulesFilename's output.
const rulesSuffix = "_rules.md"

// Input is one distilled ruleset to synthesise: a human title used to label its
// block and the ruleset body verbatim. It is deliberately not the structured
// ruleset.Ruleset — synthesis operates on raw ruleset text, not parsed rules.
type Input struct {
	Title string
	Body  string
}

// FillTemplate returns tmpl with its {{RULESETS}} marker replaced by one
// <ruleset id="N" source="Title"> block per input, preserving order. It fails
// loudly when the marker is absent (a template without it would silently drop
// every ruleset) or when there is nothing to synthesise, mirroring distill's
// placeholder validation.
func FillTemplate(tmpl string, inputs []Input) (string, error) {
	if !strings.Contains(tmpl, Marker) {
		return "", errors.New("synthesize: template missing " + Marker)
	}
	if len(inputs) == 0 {
		return "", errors.New("synthesize: no rulesets to synthesise")
	}
	var b strings.Builder
	for i, in := range inputs {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "<ruleset id=\"%d\" source=%q>\n%s\n</ruleset>", i+1, in.Title, in.Body)
	}
	return strings.Replace(tmpl, Marker, b.String(), 1), nil
}

// LoadInputs reads every *_rules.md in dir into synthesis inputs, titling each
// from its first H1 heading and falling back to a title derived from the filename.
// os.ReadDir returns entries name-sorted, so the inputs — and thus the assembled
// prompt — are deterministic. An empty result is not an error here; the caller
// decides whether "no rulesets" is a problem (FillTemplate treats it as one).
func LoadInputs(dir string) ([]Input, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.WrapWithMessage(err, "synthesize: read dir", slog.String("dir", dir))
	}
	inputs := make([]Input, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, rulesSuffix) {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return nil, errors.WrapWithMessage(
				readErr,
				"synthesize: read ruleset",
				slog.String("file", name),
			)
		}
		title := naming.TitleFromMarkdown(string(body))
		if title == "" {
			title = naming.Title(strings.TrimSuffix(name, rulesSuffix))
		}
		inputs = append(inputs, Input{Title: title, Body: string(body)})
	}
	return inputs, nil
}
