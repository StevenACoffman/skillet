# Skillet

**A Go toolkit for turning prose expertise into verifiable, runtime-neutral Agent Skills.**

`skillet` is the shared kernel behind a family of skill tools (exegesis, skillsaw,
canonizer, modelith). When two of them parse the same `SKILL.md` or hash the same
artifact, they must reach byte-identical answers. skillet provides that logic once, so the
tools never drift apart.

## Why Use It

- **One identity everywhere.** `identity.Hash` gives a byte-identical content hash across
  every tool, so hash-pinned manifests cross-check cleanly instead of trusting each tool's
  word.
- **Deterministic by default.** Parsing, scoring, and scanning are pure functions that run
  entirely in memory. A model handles only what a deterministic check cannot decide, so
  results stay reproducible and cheap.
- **Auditable verification.** A `Finding` is a static diagnostic tied to a re-runnable
  `Check`. A `Manifest` counts as verified only when every skill passes its gates. A
  `Proof` blocks closing until every declared artifact re-hashes to its recorded digest
  (no-proof-no-close).
- **Enforced runtime neutrality.** `neutrality.Scan` flags the wording or paths that bind a
  skill to a single agent runtime, the kind that makes other runtimes refuse to install it.
- **Proper Markdown parsing.** goldmark parses `SKILL.md` bodies, so a `#` inside a code
  fence never reads as a heading.
- **Small, focused packages.** Each one does a single job and depends on little. Take only
  what you need.

## Install

```sh
go get github.com/StevenACoffman/skillet
```

Requires Go 1.26+.

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

Each package does one job and depends on little. Take only what you need.

### Skills and Markdown

| Package       | Purpose                                                                                                                                                                                         |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `skill`       | Load and parse Agent Skills (`SKILL.md`): frontmatter plus Markdown body.                                                                                                                       |
| `markdown`    | Structured view of a skill body via goldmark (headings, fences, lists, tables).                                                                                                                 |
| `speclint`    | Validate `SKILL.md` frontmatter against the agentskills.io spec (description cap, allowed keys). The single source of truth for the spec's drift-prone data.                                    |
| `neutrality`  | Red-light scan that flags runtime-binding wording, so a skill installs in any runtime.                                                                                                          |
| `frontmatter` | Split a leading `---` YAML block from the Markdown body. Normalizes CRLF, so a caller that forgets to cannot get an empty header instead of an error.                                           |
| `redlines`    | book2skill's mechanical Quality Red Lines: the six RIA-TV++ segments, the per-quotation word ceiling, and a description that states its trigger.                                                |
| `skilllens`   | The three SkillLens quality detectors — failure mechanisms, softening phrases, blacklist sections — returning located evidence rather than diagnostics, so each consumer sets its own severity. |
| `timeseries`  | Detect a regression against a rolling baseline rather than a fixed threshold.                                                                                                                   |

### Rulesets and the Distillation Pipeline

| Package              | Purpose                                                                                                                                                     |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ruleset`            | Typed model of a distilled ruleset. Each `Rule` carries a severity, level, rationale, ✗/✓ example pair, and source anchor. `Render` and `Parse` round-trip. |
| `ruleset/distill`    | Fill a per-source distillation prompt for each source document.                                                                                             |
| `ruleset/synthesize` | Assemble distilled rulesets into one synthesis prompt.                                                                                                      |
| `naming`             | Derive filenames and human titles for the distillation pipeline.                                                                                            |
| `testprompts`        | Conservatively derive judge checks from expected-output prose.                                                                                              |

### Verification and Scoring

| Package      | Purpose                                                                                         |
| ------------ | ----------------------------------------------------------------------------------------------- |
| `finding`    | The one shared diagnostic type (severity, category, path, message) every linter and gate emits. |
| `judge`      | Deterministic rule-judge: hard = all checks pass, soft = fraction passed.                       |
| `manifest`   | Build the machine-readable record of a verified skill tree.                                     |
| `proof`      | No-proof-no-close gate: every declared artifact must exist and match its digest.                |
| `identity`   | Content-identity hashing shared across tools, so an artifact has one identity everywhere.       |
| `provenance` | Read and write the header that marks a file as a vendored copy of an upstream artifact.         |

### Optimization and Statistics

| Package       | Purpose                                                                                                               |
| ------------- | --------------------------------------------------------------------------------------------------------------------- |
| `stats`       | Small deterministic statistics (Wilson score intervals and friends).                                                  |
| `calibration` | Measure how well stated confidences match observed outcomes (ECE, MCE, Brier). The reliability complement to `stats`. |
| `ratchet`     | Keep/revert gate and activation confusion matrix for optimization loops.                                              |
| `auditlog`    | Read and write the optimization log (`results.tsv`).                                                                  |

### Foundations

| Package      | Purpose                                                                  |
| ------------ | ------------------------------------------------------------------------ |
| `errs`       | Shared error type with machine-readable codes (leaf/wrapper convention). |
| `atomicfile` | Atomic file writes: a crash never leaves a half-written file.            |
| `fsutil`     | Small `fs.FS`-shaped filesystem helpers.                                 |

See [`skillet.modelith.md`](./skillet.modelith.md) for the full domain model.

## License

See [LICENSE](./LICENSE).
