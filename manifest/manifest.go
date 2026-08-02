// Package manifest builds the skills-manifest.json a verify pass emits: a
// machine-readable summary of a verified skill tree that downstream tools read
// instead of rediscovering it. Build and Marshal are pure; the caller writes.
package manifest

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Manifest is the top-level skills-manifest.json document.
type Manifest struct {
	Tool              string  `json:"tool"`               // the emitting tool, e.g. "exegesis"
	Tree              string  `json:"tree"`               // the verified tree path
	StructureVerified bool    `json:"structure_verified"` // true iff every gate passed
	Skills            []Skill `json:"skills"`
}

// Skill is one verified skill's entry.
type Skill struct {
	Slug        string `json:"slug"`
	Dir         string `json:"dir"`
	Hash        string `json:"sha256,omitempty"`       // first-16 sha256 of SKILL.md
	TestPrompts string `json:"test_prompts,omitempty"` // path if present, else ""
}

// Build assembles a Manifest, sorting skills by slug so output is deterministic.
// tool names the emitting tool; verified reflects whether every gate passed
// across all skills. It is pure and does not mutate the input slice.
func Build(tool, tree string, skills []Skill, verified bool) Manifest {
	sorted := make([]Skill, len(skills))
	copy(sorted, skills)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slug < sorted[j].Slug })
	return Manifest{
		Tool:              tool,
		Tree:              tree,
		StructureVerified: verified,
		Skills:            sorted,
	}
}

// Marshal renders the manifest as indented JSON with a trailing newline.
func (m Manifest) Marshal() ([]byte, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	return append(b, '\n'), nil
}
