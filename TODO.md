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
- **`stats` and `ratchet` gain a 2nd consumer: adh's `verdict` (2026-08-04).** skillet side (this
  PR): `stats` gains `McNemar` beside `Wilson`. adh side (implemented, lands with the bump below):
  adh's verdict consumes `stats.McNemar`, and adh deleted its duplicate `internal/gate` to adopt
  `ratchet` (its `Result` gained the superset `Status`/`Delta` fields — additive, no consumer broke).
  `auditlog` stays single-consumer: adh keeps no results.tsv log, so it has no auditlog consumer
  (not applicable, like the deferred provenance/neutrality). adh's `Decide`/`Replicate`/`Verdict`
  taxonomy stayed in adh — domain-specific, one consumer. **Landing order:** merge this, cut
  **v0.2.0** (`ratchet` already shipped in v0.1.0; `stats.McNemar` is new), then bump adh to v0.2.0
  and drop its temporary local `replace`.
  **Update (2026-08-05):** shipped well past this note — **v0.2.0** (`stats.McNemar`), **v0.3.0**,
  **v0.4.0** (`ruleset/synthesize`), and **v0.5.0** (`ruleset.SourceAnchor` + `toerr.WrapWithMessage`
  registered with `wrapcheck`) are all tagged. adh adopted **v0.3.0** and dropped its `replace`;
  consumers now span v0.1.0–v0.5.0 (see the version-skew item under Open Threads below).

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
- [x] `frontmatter` — **`Split(text) (block, body)`**, promoted out of `skill.splitFrontmatter`.
      `exegesis merge-index` must read the YAML header of
      `source-verification/<pair-id>-{r,a1}.md`, which is not a SKILL.md; the logic existed
      but was unexported, and importing `skill` for a non-skill file would misplace the
      dependency. Its own package, not `markdown` (which parses the *body*, with goldmark).
      **Only the split is shared.** Both callers unmarshal into their own type — `skill` into
      a map (it needs the key list), merge-index into a typed header — so a `Parse(text, v)`
      convenience would be used by neither.
      **Three defects fixed in the move**, each found by probing the old code rather than
      reading it, and each mutation-tested:
      (a) **an empty header** (`---\n---\nbody`) was not recognized — the closing delimiter
      leaked into the body — because after stripping the opening line the closer sits at
      offset 0 with no newline before it, which a `"\n---"` substring search cannot see. Now
      scanned by line.
      (b) **an unterminated header** silently ate the opening line. Now the document comes
      back whole, so the one rule a caller needs is "empty block ⇒ body is everything".
      (c) **CRLF normalization moved into `Split`.** It lived in `skill.parse`, i.e. in the
      caller, so every future caller had to remember it — and one that forgets gets no error,
      just an empty block, then reports fields as missing from a file that plainly has them
      (the false-diagnostic class already fixed twice in this family).
      `Skill.Raw` is untouched, so content identity cannot move: verified across the real
      232-skill tree against a manifest written by the released skillet — **232 checked, 0
      hashes moved**. Also verified that no skill in that tree is CRLF, empty-header or
      unterminated, so (a)-(c) provably change nothing that exists today.
      src: exegesis merge-index (2026-08-07)
- [x] `manifest` — `Manifest{Tool,Tree,StructureVerified,Skills[]}` + per-skill sha256 (`Tool` is a Build param, not hardcoded).  src: exegesis
- [x] `manifest` — **the read half: `Parse`, `Diff`, `Delta.Stale()`.** The package could
      write a manifest but not consult one, so the skip-list it exists to enable had no
      reader. `Diff(base, cur) Delta` partitions every location into
      Added/Removed/Changed/Unchanged, and `Stale()` answers the one question a caller has
      (what must I reprocess?) so two consumers cannot union those fields differently.
      Three decisions the implementation turns on, each mutation-tested:
      **(a) match on location, not slug** — `DiscoverRoots` scans four runtime roots, so
      `.claude/skills/foo` and `.cursor/skills/foo` are distinct skills sharing a slug;
      keying on slug silently collapses them, in precisely the consumer this is for.
      **(b) take the location relative to each manifest's own `Tree`** — exegesis defaults
      `tree := "."` while `TREE` as an argument is absolute, so one tree yields `Dir: "foo"`
      or `Dir: "/abs/foo"` depending only on how it was invoked. Verified on the real
      232-skill steve-skill-market tree: the two spellings produce 70KB and 44KB manifests
      with no `Dir` in common, and `Diff` still reports 232 unchanged / 0 stale.
      **(c) an unknown hash is Changed, never Unchanged** — `Hash` is `omitempty` and
      exegesis leaves it empty when `skill.Load` failed, so calling it unchanged would
      permanently skip a skill that was never hashed. A location recorded twice with
      disagreeing hashes reuses the same rule rather than last-wins, which could otherwise
      hide a real edit. Not added: a `Scan(tree)` helper — one prospective consumer
      (`skillsaw changed`), held by the promote-on-second-consumer rule.
      Consumers unblocked: skillsaw's hash-keyed skip list and its `structure_verified`
      gate (both open in skillsaw/TODO.md).  src: skillsaw (2026-08-07)

### Verification

- [x] `finding` — `Diagnostic{Severity,Category,Path,Message}`; `Result`; deterministic `Sort`.  src: exegesis, modelith-shaped
- [x] `judge` — `Check{Op,Arg}`, op set + objective answer-scoring, `Score`→`Result{Hard,Soft,Why}`.  src: skillsaw⊃exegesis
- [x] `testprompts` — `File`/`Case`/`Parse`(3 shapes)/`Write`/`Validate`/`Scaffold`/`DeriveChecks`/`Behavioral`/`Decoys`/`Find`/`ChecksFor`.  src: exegesis, skillsaw
- [x] `testprompts` — **the case vocabulary is a value, not a hard-coded list.** `Validate`
      held the accepted types and their minimums as two separate hard-coded lists, so a
      caller with a fourth category could not gate at all: a conforming merged set returned
      `case N: unknown type "prefer_merged_over_source"` twice, and `Tally` silently dropped
      those cases because `Counts` has no field for them. New `Composition map[string]int` +
      `Standard()` + `ValidateAgainst(want)` + `CountOf(type)`. The keys *are* the vocabulary
      and the values are the minimums, so the two can no longer drift; a type mapped to 0 is
      accepted but not required, which falls out for free.
      **skillet does not learn what merging is** — exegesis supplies its own Composition.
      Putting `TypePreferMerged`/`MinPreferMerged`/`ValidateMerged` here would be
      special-purpose code inside a general mechanism.
      `Standard()` is a function, not an exported map var: a shared map is mutable global
      state, and one caller adding a type would change the rule for every other caller
      in the process (mutation-tested). Strictly additive — `Validate`, `Tally`, `Counts`
      and the `Min*` consts are untouched, and the `need >=N <type>, have M` message text is
      byte-identical because the type name *is* the message token. Problem order is now
      sorted (map iteration is randomized).  src: exegesis merge gate (2026-08-07)
- [x] `testprompts` — **`Parse` reports the rewrites it performed** (`File.Rewrites`, excluded
      from JSON). Third instance of the same defect class as `skill.parse` swallowing the YAML
      error: `Parse` accepts three container shapes and several legacy per-case spellings while
      `Write` emits one, so a write-back silently migrates the file — and the caller had no way
      to know, making skillsaw's "refuse to write back over a non-canonical file" option
      unimplementable. Reports the bare-array shape, the legacy `test_cases` key, per-case
      `expected_behavior`, and each id that was renumbered or retyped. Found while enumerating
      them: **when both `tests` and `test_cases` are populated the reader has always preferred
      `tests` and dropped the rest**, so a write-back deletes cases still visible on disk —
      now reported rather than silent. `[]string`, not a bool or an enum, because the caller
      needs a reason for its message; `len()==0` still answers "was it canonical?".  src: skillsaw
      test-prompts write-back (2026-08-07)
- [x] `skill` — **stop swallowing the frontmatter YAML error.** `parse` discarded it, leaving `Name`/`Description`/`FrontmatterKeys` zero, so both consumers reported the *symptoms* (`description is empty`, `name "" != folder`) on a file whose description was plainly present. New `Skill.FrontmatterErr` records the cause; `speclint.Frontmatter` now reports it **and returns**, because every other check reads a field that could not be parsed and would dress a symptom up as an independent defect. `Load` still succeeds — one malformed skill must not halt a caller walking a tree, which is why this is not a `Load` error. Real case: a book skill with `source_book: "X" by Y` went from 3 defects (one of them false) to 1 naming `[10:45]` with a caret.  src: found while wiring skillsaw preflight (2026-08-06)
- [x] `redlines` — **skip the trigger check when the frontmatter did not parse.** The last
      surviving instance of the defect fixed in `skill`/`speclint`: `checkTrigger` reads
      `Description`, which an unparsed block leaves empty, so it demanded a trigger of prose
      the author did write. Guarded on `FrontmatterErr`. Only that check — `checkSegments` and
      `checkQuotes` read the body, which `splitFrontmatter` produces before the parse is
      attempted, and a blanket suppression would have hidden a real 219-word quotation on the
      very skill that exposed this. A skill with no frontmatter at all still gets the trigger
      diagnostic, since `yaml.Unmarshal("")` succeeds — silencing that would trade a false
      positive for a false negative. Verified end to end: `exegesis lint --check redlines` on
      the offending book skill went 3 diagnostics → 2.  (2026-08-06)
- [x] `redlines` — book2skill Quality Red Lines: `MaxQuoteWords`, `Check(s)→[]finding.Diagnostic` (six RIA-TV++ segments, quotation ceiling, description states a trigger). Deliberately **separate from `speclint`**: speclint encodes the agentskills.io spec and moves when the spec moves; the red lines encode book2skill's house rules and move when the methodology moves. Messages moved verbatim from exegesis so its CLI tests pass unchanged.  src: exegesis internal/lint (promoted 2026-08-06); 2nd consumer skillsaw, wired (`internal/edit`, `preflight --redlines`) — verified 2026-08-08
- [x] `speclint` — agentskills.io frontmatter spec: `DescriptionMaxRunes`, `AllowedFrontmatterKey`, `Frontmatter(s)→[]finding.Diagnostic`. Single source of truth so exegesis (gates the findings) and skillsaw (scores the cap) can't drift by hand. Name-format policy stays per-tool (exegesis=folder, skillsaw=kebab).  src: exegesis lint + skillsaw rubric (de-duplicated 2026-08-03)
- [x] `speclint` — **the allowlist did not match the spec; corrected and released as v0.12.0 (2026-08-08).**
      Checked against <https://agentskills.io/specification>: the defined keys are `name`,
      `description`, `license`, `compatibility`, `metadata`, `allowed-tools`.
      `AllowedFrontmatterKey` admitted four of them and **rejected `license`, `compatibility`
      and `metadata`** — a skill declaring its own license was told the key does not exist.
      Being the single source of truth for the spec is worth nothing if the copy is wrong.
      `tags` stays permitted as the one deliberate deviation, documented at the switch: the
      spec does not define it, but 163 installed skills and every book2skill output carry it,
      so rejecting it would report a defect on nearly every skill rather than describe one.
      `author` and `version` are **not** top-level keys at any level of the spec — its own
      example carries both inside `metadata` — and the merge-skills doc and template that
      claimed otherwise are corrected.
      Widening can only *reduce* findings, so no consumer newly fails — verified rather than
      asserted: exegesis and skillsaw are both bumped to v0.12.0 with their suites green, and
      a skill carrying `license:` + `metadata:` goes from two `disallowed key` errors to
      `ok`. **0 skills in `books/` use the three keys; 15 of the installed skills do**, so the
      corpus this repo gates was unaffected and the corpus an agent actually loads was not.
      Two consequences worth watching: `metadata` values are string-only and nothing checks
      that yet, and merge-skills' retirement Option 3 (`metadata: {superseded-by: …}`) stops
      needing a new key here.
- [x] `redlines` — **export `Quotes(body) []string`** so `exegesis quotecheck` can locate the
      same blockquote runs the `MaxQuoteWords` red line counts. Extraction was fused into
      `checkQuotes`; it is now one definition with two users. Exported from `redlines` rather
      than `markdown` deliberately: the point is that the fabrication guard and the red line
      agree on *what counts as a quote*, and a second extractor would disagree at the margins
      (fences, lazy continuation) — the two tools would then dispute the rule being enforced.
      `Quotes` strips fenced code itself, so a direct caller gets the same answer as `Check`
      (the strip is idempotent, so `Check` is unaffected); a `>` line inside a shell transcript
      is not a quotation. Runs of bare `>` markers carry no text and are not returned.
      src: exegesis quotecheck (2026-08-07)

### Experiment Adjudication (2Nd Consumer: Adh `verdict`, 2026-08-04)

- [x] `stats` — `Wilson(k,n)` + `McNemar(improved,regressed)`.     src: skillsaw (Wilson), adh verdict (McNemar)
- [x] `ratchet` — `Evaluate`/`SelectScore` gate + activation `Score` confusion matrix (one package, 2 files).  src: skillsaw; adh adopted it (deleted its duplicate internal/gate)
- [x] `auditlog` — `Row` + `Read`/`Append` (results.tsv).          src: skillsaw (single consumer — adh has no audit log)
- [x] `timeseries` — **regression gate: `Detect(history, current, Config) Verdict`.** A gate
      that asks "is this worse than we were?" rather than "is this good enough?" — it catches a
      slide that never crosses the absolute bar and tolerates a metric that is low but stable.
      Promoted on the 2nd consumer rule: wanted by skillsaw (TODO "regression gate") *and*
      exegesis (TODO, "the one plausible touch"). Pure; the caller owns storage, so the
      reference's `SaveToFile`/`LoadFromFile` stay out.
      **Four defects in the reference (`unified-thinking/benchmarks/reporting/timeseries.go`)
      deliberately not inherited**, each mutation-tested here:
      (a) it is **not a rolling window** despite both TODOs describing it that way —
      `DetectRegression` breaks at the first entry with a value, so the baseline is the single
      most recent run and one good run becomes the bar for every run after it. Averaging does
      not erase that run, it dilutes it to 1/N, which is the point.
      (b) **`baseline == 0` is read as "no baseline"**, so a metric genuinely sitting at zero
      can never report a regression — exactly when a gate matters. Absence of history is a
      different state, carried by `Verdict.Compared`.
      (c) **degradation is relative** (`(baseline-current)/baseline`), undefined at zero and
      explosive near it. `Tolerance` is absolute, in the metric's own units; the family's
      metrics (1-10 rubric, 0-1 accuracy) have meaningful absolute scales.
      (d) it returns **`(bool, string)`**, fusing the verdict with a formatted message so a
      caller cannot render it differently and a test must string-match.
      Too little history yields `Compared:false, Regressed:false` — a gate that fails the first
      time a metric is recorded can only be fixed by not measuring. `MinHistory` defaults to 2,
      not 1, so a lone prior run never becomes the bar.  src: skillsaw + exegesis (2026-08-07)

### Rules / Distillation

- [x] `ruleset` — typed `Rule`/`Ruleset` (§, Severity MUST/SHOULD/CONSIDER, Level CODE/ARCH/METHOD) + `Render`/`Parse` (canonical form).  src: distill (greenfield)
- [x] `ruleset/distill` — source-tree → prompt generation (`FillTemplate`/`Generate`, over `naming`).  src: ai-skill main.go
- [x] `ruleset/synthesize` — rulesets → single synthesis prompt (`Marker`/`Input`/`FillTemplate`/`LoadInputs`, over `naming`); sibling of `distill`.  src: canonizer internal/synth (2026-08-04)

### Provenance / Proof

- [x] `proof` — `Artifact{Path,Digest}`, `Packet`, `Create`/`Save`/`Load`/`Verify` (on `errs`/`atomicfile`/`identity`).  src: adh
- [x] `provenance` — vendored header `{Vendored,Origin,Ref,Commit,Imported,Digest}` + `Stamp`/`Parse`/`Digest`.  src: modelith-style (generalized)

## Consumer Migration (After Each Context Lands)

- [x] exegesis → deleted `internal/{skill,neutrality,testprompts,manifest}`; repointed to skillet.
      Kept `internal/{lint,overview,registry}` (lint repointed to skillet skill+neutrality). `manifest.Build`
      call passes `"exegesis"`; `skill.Hash(s.Raw)` → `s.Hash()`. `lint.Finding` kept (finding→skillet
      deferred: exegesis-only lint output, JSON-identical to `finding.Diagnostic` for Error-only findings).
      `replace => ../../git/skillet`. Build/vet/test/golangci all green.
      **Update 2026-08-03:** on `skillet v0.1.0` (replace dropped). lint delegates
      frontmatter to `skillet/speclint` and migrated off local `lint.Finding` onto
      `finding.Diagnostic` — the deferred finding→skillet is now done.
- [x] skillsaw → deleted `internal/{skill,markdown,neutrality,testprompts,judge,stats,gate,activation,store}`;
      repointed to `skillet/{skill,markdown,neutrality,testprompts,judge,stats,ratchet,auditlog,identity}`;
      kept `internal/{rubric,edit}`. Added `cmd/root.SplitRoots` (CSV `--roots` → skillet's slice API).
      Wired via `replace => ../../git/skillet` (skillet unpublished). Build/vet/`-race`/golangci all green;
      goccy/goldmark now indirect (still offloaded via skillet). `skill.Hash` func → `identity.Hash`.
      **Update 2026-08-03:** on `skillet v0.1.0` (replace dropped); rubric's description
      cap comes from `skillet/speclint.DescriptionMaxRunes`.
- [x] adh → **errs adopted via alias** (`internal/adh/error.go`: `type Error = errs.Error` + re-exported
      codes/funcs) so all 54 `adh.*` call sites keep compiling and `proof` (returns `errs.Error`) stays
      compatible. Deleted `internal/{identity,atomicfile,proof}`; repointed to `skillet/{identity,atomicfile,proof}`.
      `replace => ../skillet`. Build/vet/`-race` (44 pkgs)/golangci all green; x/sys now indirect.
      **envelope → climax: done** (climax v0.7.0 first shipped the `--jsonl` outcome surface; latest is v0.8.0). adh's
      generic envelope shell (`Status*`/`Outcome`/`Emit*`) now lives in `cmd/root/outcome.go`
      matching climax's generated template; adh keeps its words (specialized `CodeForError`,
      `ReasonForError`, `Reason*` tokens). `// climax:features jsonl` makes `climax lint`
      drift-check the outcome surface (no drift, verified 2026-08-03). **Note:** climax v0.7.0's
      lint grew an unrelated base-scaffold check — an unmatched-subcommand guard — that adh has
      not adopted (a mistyped subcommand falls through `ff.ErrNoExec` and exits 0), so `climax
      lint` overall reports 1 error. That's adh CLI-scaffolding work, outside skillet's scope.
      **Still deferred:** `provenance`/`neutrality` → skillet
      (adh has no consumer for either — `internal/skillsaw` is unrelated).
- [x] distill (ai-skill) → rewrote `main.go` as a thin CLI over `skillet/ruleset/distill.Generate` +
      `skillet/naming` (with the `run()`-returns-int shape). `replace => ../../../git/skillet`. Build + smoke
      (8 prompts, matches original) + golangci all green. Dropped `-dry-run` (skillet's Generate always writes).

## Release / Distribution

- [x] Bump every consumer off the local `replace`. skillet is published (**v0.1.0**,
      tagged 2026-08-03) and all four consumers now require it: **exegesis** and
      **skillsaw** dropped the replace and pin v0.1.0 (PRs merged 2026-08-03);
      **distill** dropped it (committed to knowledge-base `kb`); **adh** dropped it on
      `feat/survey-tier1` (in open PR #2 — lands when that merges). skillet is now a
      real, versioned shared dependency. (survey 2026-08-02; done 2026-08-03)
- [x] Cut the post-v0.1.0 releases (2026-08-04/05): **v0.2.0** `stats.McNemar`; **v0.3.0**
      (adh adopted this — `verdict` on `stats.McNemar`, `internal/gate` replaced by `ratchet`);
      **v0.4.0** `ruleset/synthesize`; **v0.5.0** `ruleset.SourceAnchor` + `toerr.WrapWithMessage`
      registered with `wrapcheck`. Consumer pins now diverge — **skillsaw** v0.1.0, **adh** v0.3.0,
      **exegesis** v0.4.0, **canonizer** v0.5.0 — see the version-skew item under Open Threads.

## Open Threads (2026-08-05 Cross-Repo Survey)

The checklist above is complete; these are surfaced by a survey of the consumer repos and are
not yet tracked elsewhere.

- [x] **Resolve the `errs` vs `toerr` split.** DONE (2026-08-05): consolidated on a clear division —
  **toerr owns wrapping/tracing; errs owns leaf classification and the shared code vocabulary.**
  `proof`'s 7 wrappers moved to `toerr.WrapWithMessage` (joining `ruleset/synthesize`), so skillet now
  wraps uniformly through toerr and gains a `%+v` trace frame; its 4 leaf classifications stay
  `errs.Error{Code,Message}`. `errs` became toerr-aware — `ErrorCode`/`ErrorMessage` read a toerr
  `errcode` error and an `*errs.Error` alike (one `codeFromStatus` map), so both representations
  classify identically. A *full* removal of `errs.Error` is intentionally deferred: adh composes it as
  a struct (`&adh.Error{Op,Err}`, 157 sites) and toerr exposes no composable struct, so `errs.Error` is
  retained as adh's compat type. `errs` now imports toerr directly, so adh gains toerr indirect on its
  next bump — setting up an eventual adh migration off struct-literal `Error` (the remaining follow-up).
  - [ ] Follow-up (needs an adh-side change + a wrapcheck sig for `errcode.WithCode`): migrate proof's
    leaf errors to `errcode.WithCode` and adh off `&adh.Error{}` literals, then retire `errs.Error`.
- [x] **`skill.Load` → `ENOTFOUND` mapping** DONE (2026-08-05): a missing SKILL.md is translated
  at the boundary to a leaf `errs.Error{Code: ENOTFOUND}` (classify via `errs.ErrorCode`); any other
  read error wraps with `Op: "skill.Load"`. `os.ErrNotExist` is no longer propagated (verified no
  consumer relied on `errors.Is` — exegesis/skillsaw only test `err != nil`), per go-advice §3
  translate-at-the-boundary; the test now asserts the code + "not found" message.
- [x] **Consumer version skew — bump train run (2026-08-05).** The point of skillet is
  byte-identical answers across tools, yet consumers had spanned v0.1.0–v0.5.0. Each lagging
  consumer was bumped to v0.5.0 on its own branch (PRs open): **adh** v0.3.0→v0.5.0
  (agentic-dev-harness#5), **exegesis** v0.4.0→v0.5.0 (exegesis#9), **skillsaw** v0.1.0→v0.5.0
  (skillsaw#3); **canonizer** was already current. Every bump was inert — go.mod/go.sum only,
  suites green, `golangci-lint` clean — confirming the extractions across v0.1.0–v0.5.0 were
  genuinely additive. Once the three PRs merge, all four consumers share one kernel again.
- [x] **Revisit single-consumer extractions.** DONE (2026-08-05): verified by grep across the whole
  family — `auditlog` has exactly one consumer (skillsaw's `cmd/history`), and `provenance` has
  **zero** (an unused speculative extraction generalized from modelith). Both are now flagged
  **provisional** in their package docs (discoverable via `go doc`), and kept — not deleted. Exit
  criteria recorded there: `auditlog` earns the "shared" designation on a 2nd consumer; `provenance`
  should be deleted if it is still unused at the next survey rather than carry unused public surface.

## Domain Model

- [x] `skillet.modelith.yaml` + rendered `.md` capturing the entities/relationships above (authored with modelith; `modelith lint` clean).

## Reasoning-Toolkit Survey — `skillet/calibration` (Unified-Thinking, 2026-08-05)

Source: a survey of `~/Documents/git/unified-thinking` (a deterministic Go reasoning
toolkit) for techniques the family could reuse. Its one clean, well-tested gap-fill is
**confidence calibration** — the reliability complement to the significance stats `stats`
already owns.

- [x] **Add `skillet/calibration` — Brier score + ECE + MCE.** DONE (2026-08-05): shipped as
  package `calibration` (`Compute([]Sample) Report` → ECE/MCE/Brier + per-bin breakdown), pure
  `math`-only, property-based + example tests. Fixes the reference's divide-by-input-length bug
  (metrics divide by the in-range samples actually scored) and omits its arbitrary quality-label
  thresholds. `stats` measures
  *significance* (Wilson interval, McNemar); nothing in the family measured *reliability* —
  whether a stated confidence matches the observed outcome rate. Lift the math from
  unified-thinking's `benchmarks/evaluators/calibration.go` (`ComputeCalibration`: Expected
  Calibration Error, Max Calibration Error, and Brier score over N confidence buckets, using
  per-bucket *mean* confidence — the correct variant; their production tracker uses the
  weaker bucket-midpoint form). Pure value→value, no deps; a natural sibling of `stats`.
  Consumers: adh (critic/judge/evaluation confidence vs closed-arc outcomes) and skillsaw
  (rubric/judge scores vs realized quality) — see their TODOs. Give it the same
  property-based + example test treatment `stats` warrants.
- [ ] Candidate refinement — **`testprompts.File.Rewrites` can be printed but not
  classified.** It is `[]string` written for a human, which serves the "say what changed"
  case it was built for. But `exegesis tests --migrate` needs one rewrite treated
  differently from the others: a file carrying both `tests` and `test_cases` loses cases
  on write-back, so migrating it must be refused rather than reported. Deciding that from
  the slice would mean substring-matching a human-readable string to decide whether to
  destroy someone's work, which breaks the first time the wording changes — so exegesis
  re-reads the raw JSON for the two keys instead (exegesis `cmd/tests`,
  `refuseIfCasesWouldBeLost`).
  That duplication is small and deliberate, and one consumer is not evidence for a
  redesign. Recorded so the second caller that needs to *act* on a specific rewrite —
  rather than print them all — is recognised as the trigger, at which point a typed kind
  beside the message is the obvious shape. Found while building `--migrate` (2026-08-07).
- [ ] Deferred — a possible `skillet/bandit` (Thompson Sampling: Beta-Bernoulli +
  Marsaglia-Tsang Gamma sampling, plus entropy/convergence diagnostics) if a 2nd consumer
  wants principled strategy selection under uncertainty. unified-thinking's
  `internal/reinforcement` is a clean, seedable, ~90%-covered reference. Only one
  prospective consumer today (adh strategy / rework-budget), so hold until a 2nd appears —
  the family's promote-on-2nd-consumer rule.
- Deliberately NOT adopted: unified-thinking's bias / fallacy / blind-spot / evidence /
  self-eval detectors are keyword-and-substring heuristics with hand-tuned, uncalibrated
  constants — the "instruct, don't enforce" shape the family rejects — and its
  significance handling (a hardcoded `p=0.05` placeholder, no held-out split) is weaker
  than `stats`. Only the calibration math (and a few isolated algorithms, noted in the
  consumer TODOs) are worth lifting.
