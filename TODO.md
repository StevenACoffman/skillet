# Skillet TODO

`github.com/StevenACoffman/skillet` — shared library extracted from
`ai-skill`/`distill`, `exegesis`, `agentic-dev-harness` (adh), and `skillsaw`.
Theme: **knowledge artifacts and their verification** — sources, rulesets, skills,
prompts, and the findings / checks / proofs that certify them.

Scope boundary: **CLI scaffolding and the machine-output envelope do NOT live here.**
They belong to `climax` (see that repo's TODO.md). skillet is domain code only.

## Preserve Mature Libraries (Hard Constraint)

Where two source repos agree, keep whatever third-party library the originals used to
offload a job; only resolve a library where the two apps genuinely conflict.

| Concern                   | Original library                                             | Repos              | Decision                                                       |
| ------------------------- | ------------------------------------------------------------ | ------------------ | -------------------------------------------------------------- |
| YAML frontmatter parse    | `github.com/goccy/go-yaml`                                   | exegesis, skillsaw | keep (both agree)                                              |
| Markdown structural parse | `github.com/yuin/goldmark` (+GFM)                            | skillsaw           | keep; exegesis regex is the weaker fork → converge on goldmark |
| Content hash              | stdlib `crypto/sha256`,`encoding/hex`                        | all                | keep stdlib (byte-identical to SkillOpt)                       |
| Red-light scan            | stdlib `regexp`                                              | exegesis, skillsaw | keep stdlib                                                    |
| Dir/marker discovery      | stdlib `os`,`path/filepath`                                  | exegesis, skillsaw | keep stdlib; add `io/fs` seam for testability                  |
| Atomic file write         | vendored Tailscale `atomicfile` (BSD-3) + `golang.org/x/sys` | adh                | copy verbatim incl. LICENSE; not a go-get dep to swap          |
| TOML config               | `github.com/BurntSushi/toml`                                 | adh                | keep when config lands                                         |
| Git provenance            | `github.com/go-git/go-git/v6`                                | adh                | keep when proof/provenance lands                               |

Pinned to the originals' versions: `goccy/go-yaml@v1.19.2`, `yuin/goldmark@v1.8.4`,
`golang.org/x/sys@v0.47.0`.

## Design Decisions (Locked)

- **`finding` = static diagnostic** `{Severity,Category,Path,Message}` (exegesis lint,
  modelith-shaped). adh's `Finding` is a *hypothesis to adjudicate* — it stays in adh,
  never merged here.
- **`Discover` conflict** (skillsaw multi-root/skip-missing vs exegesis single-tree/
  error-on-unreadable) is resolved by two named `fsutil` primitives, not one flag.
- **`identity.Hash` is frozen** at `sha256(content)` → first 16 hex chars. Changing it
  breaks cross-tool manifest checks and SkillOpt parity.
- **`atomicfile` is vendored Tailscale BSD-3 code, copied verbatim** (incl. LICENSE and the
  windows/notwindows syscall variants). It is excluded from house-rule linting via a path
  exclusion — the rule-*preserving* treatment adh uses for foreign code ("skip it entirely
  rather than relaxing any rule"), not a rule relaxation.
- **`skill` collapses the exegesis↔skillsaw fork** into one superset: it keeps exegesis's
  `FrontmatterKeys` and map-based parse (skillsaw loses nothing), preserves `goccy/go-yaml`,
  and sits the two `Discover` shapes on `fsutil` — `Discover(tree)` (single-tree) and
  `DiscoverRoots(base, roots)` (multi-root). `Hash` becomes the method `Skill.Hash()`
  delegating to `identity.Hash` (algorithm not duplicated).
- **`markdown` converges on goldmark** (skillsaw's parser); exegesis's regex section/code
  helpers are the weaker fork and are retired on migration.
- **`errs` is Ben Johnson's `Error` verbatim** (adh's `internal/adh` error machinery), renamed
  to package `errs`. `skill.Load` keeps `fmt.Errorf`+`%w` (matches both apps; propagates
  `os.ErrNotExist`) rather than coupling to `errs` — mapping to `ENOTFOUND` is a later refinement.
- **`testprompts` collapses the exegesis↔skillsaw fork onto `judge`**: skillsaw's lenient
  3-shape `Parse` + `Behavioral`/`Decoys`/`Find`/`ChecksFor`, exegesis's `Write`/`Tally`/
  `Validate`/`Scaffold` + composition minimums, one json-tagged `Case` using `judge.Check`.
  `DeriveChecks` is skillsaw's (returns `judge.Check`); the two were already byte-identical.
- **`judge`/`naming`/`testprompts` keep their package-level `regexp.MustCompile` vars** — verified
  gochecknoglobals permits them (judge ships 7 and lints clean), so they stay verbatim. (`neutrality`'s
  earlier const-in-func form was over-cautious but is left as-is.)
- **`finding.Severity` is `"error"`/`"warning"`** to match `skillet.modelith.yaml`; exegesis's
  `"warn"` maps to `"warning"` on migration. adh's adjudication `Finding` stays in adh (not merged).
- **`manifest.Build` takes a `tool` parameter** (exegesis hardcoded `"exegesis"`) so exegesis and
  skillsaw both use it; skills still sorted by slug for deterministic output.
- **`ratchet` is one package for gate + activation** (`gate.go` keep/revert decision, `activation.go`
  confusion matrix over `skillet/stats`) — the "ratchet subsystem" per the TODO name.
- **`provenance` is generalized from modelith's**, not copied: a skillet-neutral `# skillet-vendored-`
  prefix, the model's `{Vendored,Origin,Ref,Commit,Imported,Digest}` fields, digest-over-stripped-
  content (modelith ADR-0015 idea). modelith's fetch-method/validate/`Problem` machinery is dropped.
- **`ruleset` is greenfield** — real distilled files are free-form (`## N.` + `**Do**`/`**Do not**`),
  so `Parse` handles the *canonical* `§N.M [SEV][LEVEL]` form `Render` emits (round-trip), not every
  hand-authored file.
- **`stats`/`ratchet`/`auditlog` have one consumer (skillsaw) today** — extracted faithfully as
  working code; their shape may shift when adh's `verdict` becomes the 2nd consumer.

## Package Backlog (Dependency Order)

### Kernel (This Slice)

- [x] `identity` — `Hash(string) string` (sha256[:16]).            src: exegesis, adh, skillsaw
- [x] `neutrality` — `Scan([]NamedFile) []Hit` red-light regex.    src: exegesis, adh, skillsaw (byte-identical)
- [x] `fsutil` — dir/marker discovery over `io/fs`.                src: exegesis, skillsaw (conflict resolved)

### Kernel (Next)

- [x] `errs` — `Error{Code,Op,Message,Err}` + codes + `ErrorCode`/`ErrorMessage`.  src: adh
- [x] `atomicfile` — `WriteFile`, `Rename` (vendored Tailscale, lint-excluded).    src: adh

### Document

- [x] `markdown` — goldmark `Parse` → `Doc{Sections,Prose,Links,HasOrderedList}`.  src: skillsaw⊃exegesis (goldmark)
- [x] `naming` — `Title`/`RulesFilename`/`PromptFilename`/`TitleFromMarkdown`/`TitleFromFile`. (`Slug` lives in `skill`.)  src: distill

### Skill Artifact

- [x] `skill` — `Skill{...}`, `Load`, `Discover`/`DiscoverRoots`(over fsutil), `DefaultRoots`, `Slug`, `Hash()`→identity.  src: exegesis, skillsaw (goccy/go-yaml)
- [x] `manifest` — `Manifest{Tool,Tree,StructureVerified,Skills[]}` + per-skill sha256 (`Tool` is a Build param, not hardcoded).  src: exegesis

### Verification

- [x] `finding` — `Diagnostic{Severity,Category,Path,Message}`; `Result`; deterministic `Sort`.  src: exegesis, modelith-shaped
- [x] `judge` — `Check{Op,Arg}`, op set + objective answer-scoring, `Score`→`Result{Hard,Soft,Why}`.  src: skillsaw⊃exegesis
- [x] `testprompts` — `File`/`Case`/`Parse`(3 shapes)/`Write`/`Validate`/`Scaffold`/`DeriveChecks`/`Behavioral`/`Decoys`/`Find`/`ChecksFor`.  src: exegesis, skillsaw

### Experiment Adjudication (Single-Consumer Today; Shape May Shift When Adh `verdict` Is the 2nd Consumer)

- [x] `stats` — `Wilson(k,n)`.                                     src: skillsaw (adh `verdict` adjacent)
- [x] `ratchet` — `Evaluate`/`SelectScore` gate + activation `Score` confusion matrix (one package, 2 files).  src: skillsaw
- [x] `auditlog` — `Row` + `Read`/`Append` (results.tsv).          src: skillsaw

### Rules / Distillation

- [x] `ruleset` — typed `Rule`/`Ruleset` (§, Severity MUST/SHOULD/CONSIDER, Level CODE/ARCH/METHOD) + `Render`/`Parse` (canonical form).  src: distill (greenfield)
- [x] `ruleset/distill` — source-tree → prompt generation (`FillTemplate`/`Generate`, over `naming`).  src: ai-skill main.go

### Provenance / Proof

- [x] `proof` — `Artifact{Path,Digest}`, `Packet`, `Create`/`Save`/`Load`/`Verify` (on `errs`/`atomicfile`/`identity`).  src: adh
- [x] `provenance` — vendored header `{Vendored,Origin,Ref,Commit,Imported,Digest}` + `Stamp`/`Parse`/`Digest`.  src: modelith-style (generalized)

## Consumer Migration (After Each Context Lands)

- [x] exegesis → deleted `internal/{skill,neutrality,testprompts,manifest}`; repointed to skillet.
      Kept `internal/{lint,overview,registry}` (lint repointed to skillet skill+neutrality). `manifest.Build`
      call passes `"exegesis"`; `skill.Hash(s.Raw)` → `s.Hash()`. `lint.Finding` kept (finding→skillet
      deferred: exegesis-only lint output, JSON-identical to `finding.Diagnostic` for Error-only findings).
      `replace => ../../git/skillet`. Build/vet/test/golangci all green.
- [x] skillsaw → deleted `internal/{skill,markdown,neutrality,testprompts,judge,stats,gate,activation,store}`;
      repointed to `skillet/{skill,markdown,neutrality,testprompts,judge,stats,ratchet,auditlog,identity}`;
      kept `internal/{rubric,edit}`. Added `cmd/root.SplitRoots` (CSV `--roots` → skillet's slice API).
      Wired via `replace => ../../git/skillet` (skillet unpublished). Build/vet/`-race`/golangci all green;
      goccy/goldmark now indirect (still offloaded via skillet). `skill.Hash` func → `identity.Hash`.
- [x] adh → **errs adopted via alias** (`internal/adh/error.go`: `type Error = errs.Error` + re-exported
      codes/funcs) so all 54 `adh.*` call sites keep compiling and `proof` (returns `errs.Error`) stays
      compatible. Deleted `internal/{identity,atomicfile,proof}`; repointed to `skillet/{identity,atomicfile,proof}`.
      `replace => ../skillet`. Build/vet/`-race` (44 pkgs)/golangci all green; x/sys now indirect.
      **Deferred:** envelope → climax (climax has no `outcome` pkg yet — blocked); `provenance`/`neutrality`
      (adh has no consumer for either — `internal/skillsaw` is unrelated).
- [x] distill (ai-skill) → rewrote `main.go` as a thin CLI over `skillet/ruleset/distill.Generate` +
      `skillet/naming` (with the `run()`-returns-int shape). `replace => ../../../git/skillet`. Build + smoke
      (8 prompts, matches original) + golangci all green. Dropped `-dry-run` (skillet's Generate always writes).

## Domain Model

- [x] `skillet.modelith.yaml` + rendered `.md` capturing the entities/relationships above (authored with modelith; `modelith lint` clean).
