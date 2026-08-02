# Skillet

**A Go toolkit for turning prose expertise into verifiable, runtime-neutral Agent Skills.**

`skillet` is the shared kernel behind a family of skill tools (exegesis, skillsaw, modelith). When two of them parse the same `SKILL.md` or hash the same artifact, they must reach byte-identical answers. skillet provides that logic once, so the tools never drift apart.

## Why Use It

- **One identity everywhere.** `identity.Hash` gives a byte-identical content hash across every tool, so hash-pinned manifests cross-check cleanly instead of trusting each tool's word.
- **Deterministic by default.** Parsing, scoring, and scanning are pure functions that run entirely in memory. A model handles only what a deterministic check cannot decide, so results stay reproducible and cheap.
- **Auditable verification.** A `Finding` is a static diagnostic tied to a re-runnable `Check`. A `Manifest` counts as verified only when every skill passes its gates. A `Proof` blocks closing until every declared artifact re-hashes to its recorded digest (no-proof-no-close).
- **Enforced runtime neutrality.** `neutrality.Scan` flags the wording or paths that bind a skill to a single agent runtime, the kind that makes other runtimes refuse to install it.
- **Proper Markdown parsing.** goldmark parses `SKILL.md` bodies, so a `#` inside a code fence never reads as a heading.
- **Small, focused packages.** Each one does a single job and depends on little. Take only what you need.

## Install

```sh
go get github.com/StevenACoffman/skillet
```

## Example

```go
package main

import (
	"fmt"

	"github.com/StevenACoffman/skillet/identity"
	"github.com/StevenACoffman/skillet/judge"
	"github.com/StevenACoffman/skillet/skill"
)

func main() {
	// Load and identify a skill.
	s, err := skill.Load("skills/my-skill")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s  %s\n", identity.Hash(s.Raw), s.Name)

	// Score model output against deterministic checks.
	res, err := judge.Score("Result: 42", []judge.Check{
		{Op: judge.OpSectionPresent, Arg: "Result"},
		{Op: judge.OpContains, Arg: "42"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("hard=%.0f soft=%.2f\n", res.Hard, res.Soft) // hard=1 soft=1.00
}
```

## Packages

| Package                | Purpose                                                                                                                                |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `identity`             | Content-identity hashing shared across tools, so an artifact has one identity everywhere.                                              |
| `skill`                | Load and parse Agent Skills (`SKILL.md`): frontmatter plus Markdown body.                                                              |
| `markdown`             | Structured view of a skill body via goldmark (headings, fences, lists, tables).                                                        |
| `ruleset`              | Typed model of a distilled ruleset. Each Rule carries a severity, level, rationale, and ✗/✓ example pair. Render and Parse round-trip. |
| `neutrality`           | Red-light scan that flags runtime-binding wording so a skill installs in any runtime.                                                  |
| `finding`              | The one shared diagnostic type (severity, category, path, message) every linter and gate emits.                                        |
| `judge`                | Deterministic rule-judge: hard = all checks pass, soft = fraction passed.                                                              |
| `manifest`             | Builds the machine-readable record of a verified skill tree.                                                                           |
| `proof`                | No-proof-no-close gate: every declared artifact must exist and match its digest.                                                       |
| `provenance`           | Reads/writes the header that marks a file as a vendored copy of an upstream artifact.                                                  |
| `naming`               | Derives filenames and human titles for the distillation pipeline.                                                                      |
| `testprompts`          | Conservatively derives judge checks from expected-output prose.                                                                        |
| `stats`                | Small deterministic statistics (e.g. Wilson score intervals).                                                                          |
| `ratchet`              | Keep/revert gate and activation confusion matrix for optimization loops.                                                               |
| `auditlog`             | Reads/writes the optimization log (`results.tsv`).                                                                                     |
| `errs`                 | Shared error type with machine-readable codes (leaf/wrapper convention).                                                               |
| `atomicfile`, `fsutil` | Atomic writes and `fs.FS`-shaped filesystem helpers.                                                                                   |

See [`skillet.modelith.md`](./skillet.modelith.md) for the full domain model.

## License

See [LICENSE](./LICENSE).
