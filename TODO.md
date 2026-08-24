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
- [x] `skilllens` — **the three SkillLens quality dimensions, promoted on the 2nd consumer
      rule.** DONE (2026-08-08).
      The dimensions come from `microsoft/SkillLens`
      (`data/meta_skills/quality_rubric_3dim.md`, arXiv:2605.23899) — failure-mechanism
      encoding, actionable specificity, and a high-risk action blacklist, each validated at
      65–66% predictive accuracy against downstream skill utility. skillsaw scores them as
      rubric dims 3/5/9 (35 of its 100 points) and adh scores them as `failure-handling` /
      `actionable-specificity` / `boundary-section` (**60** of its 100). Two consumers,
      two independent implementations, zero shared code — the exact drift `redlines` and
      `speclint` were promoted to prevent, and the one place it has already happened.
      **The detectors are unimportable, which is why adh reimplemented rather than reused.**
      They live in `skillsaw/internal/rubric/rubric.go`: the `failureEN`/`failureCN` regexes
      (inline `if/when X fails|errors|times out|not found` branches), and the `FailureSections`,
      `Softening` and `BlacklistHeadings` vocabularies on `Config`. `internal/` makes them
      unreachable from adh, exegesis or canonizer; nothing about them is skillsaw-specific.
      Shape — pure, over an already-parsed doc, matching `neutrality`/`redlines`:
      `FailureMechanisms(d markdown.Doc) []Span`, `SofteningPhrases(d markdown.Doc) []Span`,
      `BlacklistSections(d markdown.Doc) []Span`. Take `markdown.Doc` and not a raw string so
      `Doc.Prose` (code blocks and spans already blanked) is what gets matched — skillsaw
      relies on that today, and a caller passing raw text would match a `# Boundary` inside a
      shell transcript.
      **Return spans, not diagnostics.** `redlines` and `speclint` return
      `[]finding.Diagnostic` because each *is* a gate with a fixed severity. These are not:
      skillsaw turns them into 1-10 rubric penalties, adh into a 0..1 factor, and exegesis
      would want error-vs-warning per check. Returning `Diagnostic` would force a severity
      here that all three callers then have to unpick, so return the located evidence and
      let each consumer decide what it means.
      **Carry the bilingual vocabulary verbatim.** skillsaw's lists cover both the
      Chinese darwin-source terms and English equivalents, on the deliberate ground that
      "a China-only list scores every English skill as defect-free, which is the opposite
      of useful" (`rubric.go:88-92`). Dropping half on the way up would silently widen
      what passes.
      **Not promoted: the weights or the 1-10 scale.** Those are rubric policy and they
      legitimately differ — skillsaw weights the three at 35, adh at 60, because they grade
      different artifacts. Same rule that kept `speclint`'s cap here and the scoring
      penalties in skillsaw.
      Consumers on landing: skillsaw dims 3/5/9 and adh `failure-handling`/`boundary-section`
      (both delete a private copy), then exegesis's proposed `--check skilllens` tier and
      canonizer's `verify.Specificity` — see those TODOs.
      **Shape as landed.** `FailureMechanisms`, `SofteningPhrases`, `BlacklistSections`,
      each taking `*markdown.Doc` — a pointer, not the value the entry wrote: `Doc` is ~72
      bytes of headers and `gocritic`'s hugeParam rejects it, and skillsaw already passes a
      pointer throughout.
      **`Span` carries a `Kind`, which the entry did not specify and the detectors demand.**
      dim 3 counts inline `if X fails` branches *and* sections named for failure, weighing
      them differently, so one call returns both and a caller that could not tell them apart
      could not reproduce the existing score. `Span.Units` travels with a section span
      because the substance threshold is policy, not detection: skillsaw requires a boundary
      section to out-weigh the body it sits in, adh does not. Same reason these return
      evidence rather than `finding.Diagnostic`.
      The three vocabularies are exported as **functions** returning a fresh slice, so
      skillsaw's `Config` can source them without an exported slice becoming shared mutable
      state one consumer edits under the others (mutation-tested).
      **Equivalence proved, not assumed:** the new detectors and verbatim copies of
      skillsaw's private ones were run over the real 233-skill corpus and agree on all four
      counts for every skill — **233 compared, 0 mismatches**. A promotion that silently
      changed what is detected would have moved every dim-3/5/9 score in the tree.
      Found while testing, and left as-is because it is the existing behaviour: the failure
      vocabulary is literal, not lemmatised — `timeout` is listed, so "the API times out"
      reads as no mechanism. Recorded in the tests rather than fixed, since changing it here
      would break the equivalence this promotion depends on.
      Consumers can now delete their copies: skillsaw dims 3/5/9 and adh
      `failure-handling`/`boundary-section`, after the next release.
      src: skillsaw `internal/rubric` + adh `internal/rubric` (2026-08-08); analysis in
      `~/Documents/agent-orange/skillopt_changes_findings.md`

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
  **Both criteria have since resolved and neither was noticed, because they were recorded inside a
  ticked item.** See the open entry below (2026-08-22): `provenance` is still at zero importers
  after three surveys, so its criterion says delete; `auditlog`'s resolved *negatively* — its one
  plausible second consumer, gnosis, examined it and declined in writing.
- [x] **A derived applicability predicate — "does this document execute anything?"** DONE
  (2026-08-14) as **`markdown.Doc.HasCodeBlock`**, not a `skilllens` function. Writing the
  interface comment first showed the entry's placement was wrong for two independent reasons.
  **The evidence does not survive the parse:** `Doc.Prose` has code blocks *blanked* and
  `Doc.Links` keeps only code-span *contents*, so a `skilllens` predicate taking a `*Doc`
  cannot see a fenced block at all — it would have to re-parse the body, a second parser for
  a fact the first one already had. And once `markdown` records it, any
  `skilllens.Executes(d) { return d.HasCodeBlock }` is a **pass-through method**, which §4
  prohibits outright. `hasOrderedList` is the exact precedent: one `ast.Walk`, a bool on
  `Doc`, in the package that already owns "what is in this document".
  **Bool, not a count.** Nothing thresholds on *how many* code blocks; a count would invite a
  threshold nobody has calibrated. **Named for the fact, not the conclusion** — `Executes` is
  a skill-domain reading of the fact, and `markdown` does not know what a skill is.
  **Inline code spans deliberately do not count**: naming `foo` in backticks demonstrates
  nothing runnable, and counting spans would make nearly every prose document look
  executable. That is the one way this could be quietly wrong, so it has its own test and a
  mutation case.
  Verified over the 233-skill corpus: `HasCodeBlock` agrees with an independent scan on
  **233/233**, and 5 mutations (count code spans, drop indented, drop fenced, always-false,
  stop at top level) were all caught, each behind a build gate.
  **The corpus check corrected the measurement that motivated this.** Two earlier scans were
  wrong in the same way — a `^```` regex misses a fence indented inside a list item, which is
  common here — and the first probe parsed frontmatter as body. Re-measured properly:
  **154 of 233 skills (66%) have zero inline failure branches**, and gating a `units > 3`
  deduction cuts the docked set **36 → 26, suppressing 10**, not the 14 first recorded. The
  consumers' TODOs carry the corrected figures.
  No consumer scoring changed here, no `skilllens` change, and no `Applicability` type: this
  ships the fact, and adh and skillsaw decide policy against their own corpora.
- Superseded framing (kept for the argument, which still holds):
  **A derived applicability predicate for `skilllens` — "does this document execute anything?"**
  Surfaced by a 2026-08-09 survey of how the family handles checks that are a category error for some
  document kinds. The `skilllens` section detectors match on heading *title* (`"boundary"` is in both
  `FailureSectionTitles()` and `BlacklistTitles()`), so a bare `## B — Boundaries` heading satisfies
  them with nothing underneath. Measured over the 233-skill corpus: **144 skills (62%) have zero
  inline failure branches and pass on the heading alone.** Tightening that is only safe if the
  tightened check can be suppressed for documents that legitimately encode no failure mechanism.
  **Four consumers already need the same predicate**, which is what earns the promotion rather than a
  fourth private copy: adh's `KeyFailure` (weight 20) and `KeyBoundary` (weight 15) — neither
  `NeedsJudge`, so 35 points with no human backstop — and skillsaw's dim 3 and dim 4 (230 skills
  flagged "judge if this skill type needs them"). This is the same 2nd-consumer argument that promoted
  `skilllens` itself.
  Shape: a pure predicate over an already-parsed `*markdown.Doc`, matching the rest of the package —
  evidence out, policy left to the caller. Derived from content (any fenced code block), **not read
  from frontmatter**: a declared `category:` is a self-report written by the same generator being
  measured, letting a document opt out of its own worst dimension. The corpus has no type field today.
  Validated on the corpus: gating a `units > 3` deduction cuts the docked set from 34 to 20 and
  suppresses exactly the 14 zero-fence decision skills (`grpc-vs-rest-vs-graphql`,
  `ml-simplest-model-baseline-first`) while still docking CLI wrappers — `gh-cli` has 87 shell blocks
  and 0 failure branches, which is the defect the dimension exists to catch, not a category error.
  **Do not generalize further than this.** The family currently mitigates the same problem four
  different ways — a derived gate (`redlines.checkTrigger`'s `FrontmatterErr == nil`), a manual
  opt-in flag (`--check redlines`), a defer-to-judge flag (skillsaw dims 4/9), and an advisory
  severity (`canonizer verify.Specificity`). Those differ enough that a general `Applicability`
  mechanism now would be special-purpose code wearing a general name. Promote the one predicate that
  has repeated; leave the four gate styles alone until a shape repeats.
- [ ] **DECIDED 2026-08-15: the exit condition IS met — name the two-member family.**
  The shape to name is the one `HasCodeBlock` and `Convention` share: **derive a predicate
  from the artifact or its corpus, then suppress the deduction.** The other three answers
  (manual flag, defer-to-judge flag, advisory severity) are about *who decides* and stay
  out of it — a type covering all five would be the general name over special-purpose code
  the original note warned against.
  Constraint from skillsaw, already settled, which the type must not blur: **gate on the
  artifact, flag on the inputs.** Dim 3 is a category error and gets a suppressing
  predicate; dim 8 is missing input and gets a flag with nothing suppressed. An
  `Applicability` that could express dim 8 would launder unfinished work as inapplicable.
  **RESOLVED 2026-08-22: name the rule, promote no type.** The exit condition was evaluated
  against the wrong population, and counting properly answers it — in the direction of not
  building anything.
  **`Convention` does not exist.** Zero occurrences across all six repositories; it is a
  proposal in `exegesis/TODO.md` sourced from `coherence`. So the "two-member family" was
  one shipped predicate and one hypothetical, which is a family of one.
  **And the mature implementation was never counted, because it is not a predicate.**
  gnosis shipped `internal/lint` with `Check{Name, Applies func(*Snapshot) (bool, string),
  Run}`, a `Skip{Check, Reason}` record, and `Report.Skipped`. Its package doc states the
  principle better than this entry did.
  **`HasCodeBlock` is consumed four ways by two consumers, and every one of them is
  deliberate.** skillsaw dim 3 branch 1 applies the penalty; branch 2 flags and never docks
  (*"executes nothing to fail; judge the base"*, with the comment *"a mis-derived category
  must not hide a real gap"*); skillsaw dim 4/8 sets `NeedsJudge` and only rewords the
  message (*"likely not applicable"*); adh returns full credit plus *"the artifact executes
  nothing to fail"*. Add gnosis's skip record and that is five sites. **All five carry a
  reason. None is silent. Each suppresses a different thing** — a deduction, a whole check,
  or nothing at all — and each has a comment reasoning about which.
  **That convergence argues against a shared type rather than for one.** A type would have
  to be generic over score, factor, diagnostic, and nothing-at-all, and the hard question —
  *what do I do when it does not apply* — has a different right answer at each site,
  already documented at each site. It also walks into this entry's own guard: dim 8
  suppresses **nothing**, so a type expressive enough to cover it is a type that can launder
  unfinished work as inapplicable. And gnosis's `Check` is parameterized on a gnosis type;
  generalizing it needs generics or an interface, against the family's concrete-first
  preference.
  **The rule, lifted verbatim from gnosis's `internal/lint` package doc, which says it
  best:**
  > Applicability is derived, not declared, and a run states what it skipped. A check that
  > silently declines to run is indistinguishable from a check that found nothing.
  **With the corollary the five sites demonstrate and no type could carry:** *what* gets
  suppressed is the consumer's choice — a deduction, a whole check, or nothing — but the
  reason is never optional. **Suppressing nothing and only rewording is a legitimate
  answer**, and it is the right one when the input is missing rather than the category
  wrong. That sentence is what keeps dim 3 and dim 8 apart, which is the property this entry
  said must survive any answer: a type blurs it, a rule states it.
  **Placement:** this entry, plus a one-line obligation on `markdown.Doc.HasCodeBlock` —
  a derived fact, and a consumer that suppresses anything on it must say so and why. Same
  shape as the `skilllens` category constants decided the same day: the kernel publishes the
  fact and names the obligation, the consumer keeps the policy.
  **Trigger for the one thing that might still promote, and it is not `Applicability`:** the
  pair `Skip{Check, Reason}` plus *a report carries one*. That earns sharing when **a second
  consumer emits a check report** — today only gnosis does, since skillsaw emits scores and
  adh emits factors. Narrow, observable, and unlike this entry's original exit condition it
  names a thing that can be counted rather than a shape that has to be judged.
  Original framing:
  **The exit condition may now be met — a fifth way has appeared.**
  That note said to wait "until a shape repeats". `exegesis/TODO.md` records a fifth answer to
  "does this check apply to this document", from `coherence`'s `OrphanEndpoints` meter: a
  `Convention bool`, true only when the current graph contains any `verifies` edge — proof the
  repo actually uses the pattern being checked — which skips score-based promotion when false.
  **It is derived from the corpus rather than declared, judged, or opted into**, which puts it in
  the same family as `markdown.Doc.HasCodeBlock` and not in the same family as the other four.
  So the question is now answerable rather than deferred: two of the five (`HasCodeBlock`,
  `Convention`) share a shape — *derive a predicate from the artifact or its corpus, then suppress
  the deduction* — and three do not (a manual flag, a defer-to-judge flag, an advisory severity),
  because those three are about **who decides**, not about whether the check applies.
  Do not answer it by generalizing over all five. The honest options are: name the two-member
  family and give it a type, or note that two members is still not a mechanism and wait. The
  second is defensible; what is not defensible is leaving the note's exit condition unevaluated
  now that something has tripped it.
  Related and already settled, so do not re-derive it: skillsaw records **gate on the artifact,
  flag on the inputs** — dim 3 is a category error and gets a suppressing predicate; dim 8 is
  missing input and gets a flag with nothing suppressed. Any `Applicability` type must not blur
  those, because a category input for dim 8 would launder unfinished work as inapplicable.

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
- [x] **`testprompts.File.Rewrites` can be printed but not classified.** DONE 2026-08-23 as
  `func (f *File) DroppedCases() int`, per the 2026-08-22 decision below.
  **A count rather than the `bool` that decision named**, and the entry's own text is why:
  it says *"or a count, if a caller wants to say how many"*, and the caller does. exegesis's
  refusal reports how many cases sit under each key, which is actionable where a bare
  refusal is not — a bool would have forced a second method the moment that message was
  written. `> 0` is the predicate.
  Unexported field, exported method: like `Rewrites` it describes the file as read rather
  than the document, so an exported field would need the same `json:"-"` and the same
  caveat, and a method inherits both.
  **Verified against the detector it removes a copy of**, rather than against fixtures
  alone: a test reproduces exegesis's `refuseIfCasesWouldBeLost` raw-JSON check exactly and
  asserts the two agree on all four container shapes. If they had disagreed, what is being
  removed was not duplication.
  **exegesis cannot use it until the next release** — it pins by version with no `replace`,
  so its `:1549` stays open until then.
  Original entry: Candidate refinement — **`testprompts.File.Rewrites` can be printed but not
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
  **DECIDED 2026-08-22: a predicate, not a typed kind — and counting the kinds is what
  decided it.** Verified against the code rather than the entry: `Parse` and `normalize`
  produce **seven** distinct rewrite messages between them, not the handful the entry
  implies — top-level array, legacy `test_cases` key, both keys populated, legacy
  `expected_behavior`, id-is-a-string, id-is-not-a-number, and no-id.
  **Exactly one of the seven drops cases**, and it is the one exegesis already refuses on.
  A second is lossy in a weaker sense — renumbering a non-numeric id discards the original
  string — but it reshapes rather than destroys, which is exegesis's own distinction:
  *"the one rewrite which destroys work rather than reshaping it."*
  So a typed kind would be a seven-member closed vocabulary in the kernel, six of whose
  members exist so the seventh can be named, and every future legacy spelling would become
  a kernel release where today it is an appended string. **The demand is one bit.**
  **Add `func (f *File) DropsCases() bool`** (or a count, if a caller wants to say how
  many). `Rewrites` stays `[]string` and stays human-facing, which is what it is good at.
  A method rather than a field, because `Rewrites` is `json:"-"` and describes *the file as
  read, not the document* — a bool field would need the same tag and the same caveat, and a
  method inherits both for free.
  **What this actually buys is removing a knowledge duplication, not a few lines.**
  exegesis's `refuseIfCasesWouldBeLost` re-unmarshals the raw JSON into a throwaway struct
  to detect the two-keys case, and its comment gives a good reason — *"pattern-matching a
  human-readable string to decide whether to destroy data would break the first time that
  wording changed"* — which is right, and is an argument for a predicate rather than for
  the duplication. As it stands the judgement *which shapes destroy work* lives in two
  modules, so a future shape that also drops data would be recorded by `Parse` and silently
  not refused by exegesis. That is the §4 information-leakage red flag, and it is how the
  two `softening` spellings happened.
  **Deliberately not taken: returning the dropped cases** (`Dropped []Case`). Strictly more
  useful *if* a caller wants to merge them, and nobody does — the need is to refuse. It is
  a superset of the predicate and can be added later without breaking it, which is the
  right shape for a maybe.
  **And the entry's own trigger could not have fired.** It waits for "the second caller
  that needs to *act* on a specific rewrite", but only one rewrite admits a different
  action; the other six all mean *converted, carry on*. Waiting for a second is waiting for
  something the vocabulary cannot produce.
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

## quotecheck Promoted (2026-08-20)

- [x] **`quotecheck` promoted from `exegesis/internal/quotecheck` on its second
      consumer.** gnosis is the second, and its most important one — the fabrication guard
      over every ingested source. Lives at `skillet/quotecheck`.
      **Only the comparison moved.** `Segment` (RIA segment labels) and the
      `redlines.Quotes` extraction stayed in exegesis, and the promoted package takes
      `[]string` quotations rather than a body. Where a quotation *begins* is the one thing
      the shared package must not know: exegesis says blockquote-inside-R, gnosis reads
      `gnosis_evidence` frontmatter, and a kernel carrying either convention is a general
      mechanism the other consumer has to route around.
      **The third outcome is the substantive addition.** `Status` is
      `Unchecked`/`Found`/`Missing` with **Unchecked as the zero value**, following
      `finding.Action`'s absent classification and `timeseries.Verdict.Compared`. Two states
      conflated three: "searched and not found" is the guard firing, "no sources supplied"
      and "every passage below `MinPassageWords`" are not, and the last was worse than
      mislabelled — a quotation too short to split produced **zero findings**, so it vanished
      and a caller counting findings saw a clean pass over something nobody checked.
      Watch for: `Finding` now has two fields that can disagree. `Status` is authoritative and
      `FoundIn` is descriptive; a `Finding{FoundIn: "x"}` with no Status is an inconsistent
      value, and it caught exegesis's own test fixture on adoption.
      **Not yet released.** exegesis is on a `replace` directive until this is tagged, and
      gnosis cannot depend on it before then.

______________________________________________________________________

## Contradiction Detection — Knowledge-Base Ingestion (Agent-Red Survey, 2026-08-15)

Source: a survey of `~/Documents/agent-red` (26 agent-tooling projects) against the
knowledge-base ingestion gap — taking in outside sources and accreting domain knowledge
without polluting the corpus with contradictory or low-quality material. The finding that
matters here holds across all 26: **every one of them detects similarity; none adjudicate
conflict.** llmwiki surfaces contradictions, mnemon deduplicates, coherence finds broken
support links — nothing decides which of two conflicting claims is authoritative, or
records why.

**Update 2026-08-20, from the gnosis side.** `ruleset/conflict` shipped and is right for
rulesets, and it is **not** reusable for a knowledge corpus. It reads `Statement`,
`Severity`, `Level`, and `Section` off a `ruleset.Rule`; a gnosis claim has none of those,
and its decidable predicates are numeric, threshold, and enumeration comparisons over a
subject key. The two share a *shape* — pure, deterministic, findings not scores,
fold-normalised equality — and nothing else. Generalising `Find` across both would put
`if isRule … else if isClaim` inside a general mechanism. So gnosis writes its own, and this
reopens only if a third consumer wants the same comparison.

The family already owns both halves that bracket the gap. `ruleset` carries the typed
normative form (§, MUST/SHOULD/CONSIDER, CODE/ARCH/METHOD, `SourceAnchor`); `finding`
carries the verdict shape. What is missing is the comparison of one rule against another,
and it belongs here rather than in a consumer for the usual reason: **canonizer needs it to
certify that a ruleset is internally consistent** — which is the base rules' entire claim —
and **merge-skills needs the same comparison read the other way.** Convergence and
divergence are one predicate over one pair, not two implementations. The 2nd-consumer bar is
met before the knowledge-base tool exists at all.

- [x] **`ruleset/conflict` — decidable inconsistencies between two rules, as findings.** DONE
  2026-08-15. Three predicates, no tunable constant: severity divergence, level divergence,
  section-identity collision. Emits `finding.Diagnostic` with **severity left unset** — whether
  a divergence blocks is canonizer's policy, and a package deciding it here would be the ship
  threshold this repo refuses, one level down. Pinned by `TestFindAssignsNoSeverity`.
  Severity and level divergence are found in one pass because both key on the same grouping —
  rules that say the same thing — and splitting them would let the two drift apart on what
  "the same statement" means.
  Original entry:
  Pure, over already-parsed `[]ruleset.Rule`; evidence out, policy to the caller, matching
  `neutrality.Scan` and the `skilllens` detectors. Emits `finding.Diagnostic`, **never a
  score** — canonizer's charter is findings-based precisely so no threshold can become a
  ship gate, and a "contradiction score" would be that threshold wearing a different name.
  Three predicates are decidable over the canonical form as it stands today, and each is
  exact — no tunable constant anywhere in them:
  (a) **Severity divergence** — two rules whose normalized text is equal but whose
  `Severity` differs (the same rule asserted `MUST` here and `CONSIDER` there). Real,
  common when two sources are distilled independently, and currently invisible.
  (b) **Level divergence** — the same equality with differing `Level` (CODE vs ARCH vs
  METHOD), which means two rulesets disagree about what kind of thing the rule governs.
  (c) **Section-identity collision** — two rules claiming one `§` in a merged ruleset.
  Not a semantic contradiction but a *provenance* one: after a merge, `↦` anchors and
  `CONTRACT:`-style references resolve to whichever copy won. rebar (`agent-red/rebar`)
  documents the same failure from the other end — numbered identifiers collide on merge,
  renumbering breaks every reference, and the number means nothing across repos. Worth
  catching mechanically since we cannot adopt their fix (our § numbering is load-bearing
  in the canonical form).
  Equality throughout is `textnorm.Fold`-normalized, **not** byte equality — see the
  promotion item below. Case is preserved, per that package's existing decision.
- [ ] **DEFERRED 2026-08-15 with a trigger: the canonical form has no subject slot.**
  Decision: do not build it yet, and do not approximate it. **The value side had never been
  measured, and it measures zero.** The only real corpus is
  `go-advice/Sources/command_rules.md` (24 rules); everything else is a 1-4 rule prompt
  example. **Not one of the 24 constrains a named quantity to an interval or an
  enumeration** — they are structural imperatives ("do not use `init()` to register
  commands", "export exactly one symbol", "return `error` from every command execution
  function"). The conflicts this slot would decide have no instance anywhere in the family.
  **Trigger to revisit: a ruleset whose rules constrain named quantities** — SRE latency or
  error budgets, config policy, resource limits. There the predicate fires immediately;
  here it cannot fire at all. The rule class is real, the corpus for it is not, and a
  feature whose only exercise is its own tests is how `provenance` ended up flagged for
  deletion with zero consumers.
  **It is also blocked on the format item below**, which is owed by the first new marker
  whatever it is, and which is cheap now and dearer with every ruleset written.
  Original entry:
  **Prerequisite, not a shortcut: the canonical form has no subject slot.** The
  conflicts worth the most — two `MUST` rules constraining the same named quantity to
  disjoint intervals, or the same slot to disjoint enumerations — are decidable by
  interval and set arithmetic with no judgment and no constants. We cannot compute them,
  because `§1.1 [MUST][CODE] Always close what you open.` carries its subject only as
  prose. Adding a structured subject/quantity slot to `ruleset` is a canonical-form change
  that breaks `Render`/`Parse` round-trip and every stored ruleset, so it is **its own
  decision**, recorded here rather than smuggled in under contradiction detection. Do not
  approximate it: extracting a subject from prose by pattern would put an uncalibrated
  heuristic underneath a blocking gate, which is the exact shape rejected in the
  unified-thinking survey above.
- [x] **The canonical form cannot gain an optional field safely.** DONE 2026-08-16 — the
  version *reader* shipped, and **`FormatVersion` is 1**. Adding the ability to declare a
  version changed no grammar, so no stored file changes and no marker was added. Framing this
  as "add versioning" invites bumping to 2 in the same change; a version whose first act is
  to break compatibility teaches every reader that the field is where breakage lives.
  **Inert, proven on the real corpus rather than fixtures:** all **29** stored rulesets in the
  family render byte-identically under v0.16.0 and under this change. `Render` emits the block
  only above version 1, which is what keeps canonizer's forthcoming canonical-form `--check`
  from reporting drift on files nobody touched — the new feature would otherwise manufacture
  the failure the next feature exists to detect.
  `Parse` resolves an undeclared file to 1, and `Render` treats 0 and 1 alike, so a `Ruleset`
  built in Go without setting `Format` is a valid v1 ruleset rather than a malformed one.
  Unlike `finding.Action`, the zero value here *is* a real answer: a file without a version is
  version 1.
  A future major is an error naming both versions, so an operator can tell "upgrade the tool"
  from "fix the file". Four mutations caught behind a build gate, including one that accepted
  a future major — a refusal nothing tests is decoration.
  **Trade-off worth recording: `ruleset` was stdlib-only and is not any more.** Reading the
  block uses `goccy/go-yaml`, already skillet's dependency via `skill` and `provenance`, so
  consumers pick it up on their next bump (`go mod tidy` resolves it; canonizer verified).
  A hand-rolled scan for one integer would have kept the package stdlib-pure, and was
  rejected: it reimplements a YAML subset in a shared kernel, and a second reader for the same
  block shape `skill` already parses is the drift this module exists to prevent.
  **Deliberately not done: absorbing `Source:` and `Scope:` into the block.** That is a second
  and larger migration, and doing it here would break the inert property above.
  Original entry:
- [x] **The canonical form cannot gain an optional field safely — fix this before adding
  one.** Found while weighing the subject slot, and independent of whether that is ever
  built: **an unknown marker line is folded into `Rationale` rather than rejected.**
  Verified against v0.16.0 rather than read from the code — parsing a rule carrying a
  hypothetical `⊕  timeout <= 30s` line yields
  `Rationale: "because latency budgets ⊕  timeout <= 30s"`. No error, no marker, no way for
  a reader to tell. So a file written by a newer version silently mis-parses in an older
  one, and `Render`/`Parse` stops round-tripping across versions while appearing to work.
  **`Parse` cannot simply be made strict**, and that is the whole difficulty:
  `applyBody`'s `default` case *is* rationale continuation, so rejecting unknown indented
  lines would reject the multi-line rationales the form is designed to carry. The fix has
  to give continuation its own representation — an explicit marker, or a version line the
  parser can refuse — before any new marker is introduced.
  **Cheap now, dearer later.** There are roughly ten stored rulesets in the family and one
  24-rule corpus; every ruleset written after this costs more to migrate. This is owed by
  whatever optional field comes first, and the subject slot is only the one that asked.
  **DECIDED 2026-08-15 — the resolution is a format version, shipped alone and first.**
  **Scope first, because it makes the cost small: this is the `ruleset` canonical form, not
  `SKILL.md`.** Its only consumers are canonizer and skillet itself — exegesis, skillsaw and
  adh have zero references — and there are roughly ten stored files, most of them 1-4 rule
  prompt examples. No external spec governs it, so a version means whatever skillet says.
  A breaking change here is a canonizer bump, not a family migration. `SKILL.md` needs none
  of this: agentskills.io owns its shape, it already has YAML frontmatter, `speclint`
  validates the key set rather than folding unknown content into a neighbouring field, and
  the spec reserves `metadata` for client-specific keys — a `format:` key there would be a
  spec violation.
  **The version is what makes strictness affordable, which is the point.** Today leniency
  *is* the forward-compatibility mechanism, so `Parse` cannot reject an unknown line without
  also rejecting rationale continuation. With a declared version those separate: across
  versions the version check does the rejecting, and within a known version an unknown marker
  can be a hard error, because the file has asserted which grammar it is written in.
  **Placement: a YAML frontmatter block, via `skillet/frontmatter.Split`.** Compatibility
  cannot decide this — **both placements were tested against v0.16.0 and both are silently
  ignored**, since the header switch has no default for top-level lines and a YAML block
  precedes any `§`. So choose on fit: `frontmatter` is already the family's metadata
  convention, has a parser, is extensible without touching the rule grammar, and can absorb
  the ad-hoc `Source:`/`Scope:` lines, which are unvalidated and cannot grow. A bespoke
  `Format: 2` header line is smaller and would do, but leaves a second metadata mechanism.
  **Sequencing, and step 1 is the one with a deadline:**
  1. Ship the version *reader* alone, changing nothing else — parse the block, treat a
     missing version as v1 so no stored file needs migrating, refuse an unknown major. Inert
     on every existing file, and worthless the day it is needed if it did not ship a release
     earlier.
  2. Then make `Parse` strict within a known version, once rationale continuation has its own
     representation.
  3. Then any new marker, subject slot or otherwise, is an ordinary v2 addition.
  **Caution:** a version field invites use. Reach for v2 when the grammar genuinely changes —
  not to record provenance, tool identity or scoring metadata. That is how a format version
  becomes a second manifest, and `identity.Hash` already pins which bytes produced what.
- [ ] **Deliberately NOT built: near-duplicate and semantic-similarity detection.** The
  obvious next detector — "these two rules are 0.87 similar, probably in conflict" —
  requires a threshold nobody has calibrated, over an embedding or an edit distance,
  gating adoption. That is the same defect as unified-thinking's bias detectors and it is
  refused for the same reason. Semantic contradiction between two rules that survive the
  decidable predicates is **judge work**, relayed through the existing prompt-filler
  boundary: canonizer already emits a cold-critic prompt that sees the source and the
  ruleset but never the reasoning that produced them, and its findings already come back
  as `finding.Diagnostic`. The residue routes there; nothing new is needed in skillet for
  it.
- [x] **A coverage record beside findings: what the critic did NOT examine.** Two consumers,
  **both with a present defect rather than a conditional want** — which is what separates this
  from the OKF entry below.
  canonizer: `critic_prompt.md` asks for `unsupported`/`vague`/`duplicate` findings, so an
  empty category is indistinguishable from an uninspected one and `gate` ships on that
  silence. adh: the same silence, and `evaluation` disposes of the arc on it. Same source for
  both (`agent-blue/super-hermes` `skills/prism-scan/SKILL.md:57`), same mechanism — a
  constraint footer, *"This analysis maximized X. It did not examine: …"*, with a persisted
  report later runs read to steer away from exhausted angles.
  **Both entries independently argue it does not compromise the cold split**, and that
  argument is the reason it is admissible: a coverage record says *what was looked at*, never
  *what was concluded* or how the artifact was produced. Feeding prior coverage to a fresh
  critic biases it toward unexamined ground — the opposite of the contamination the cold split
  prevents.
  Shape, from canonizer's entry: an additive `examined` / `not_examined` block beside the
  findings array, **advisory only — a critic that declares a gap must not thereby block, or it
  will learn to declare none.** That constraint is the whole design and must survive
  promotion. `finding.Result` is the natural home (19 references in canonizer alone), but the
  field must not be reachable by `Severity`.
  **DONE in skillet 2026-08-22 as `finding.Unexamined`; consumers wait on a release.**
  Four things changed on the way from this entry to the code, each from a rules pass.
  **`examined` was dropped.** The value is in what was *not* looked at; what *was* is either
  implied by the findings or is unverifiable testimony, and inviting a critic to claim total
  coverage invites the one claim nobody can check. Dropping it also dissolved a constraint
  the two-list shape would have needed — with one list, a name cannot be in both.
  **The name changed.** `Coverage` collides with test coverage everywhere in this family,
  and `limitations` is reserved for canonizer's authored, required, per-artifact one.
  `Unexamined` names the thing and cannot be confused with either.
  **The type earned its place, which it had not.** A struct of two string slices with no
  behaviour is the header file this file refuses elsewhere. What justifies it is the
  **required reason**: `{Aspect, Reason}` with `Valid()` rejecting either empty or
  whitespace-only, because *"I did not examine X"* with no why is exactly the boilerplate a
  required-but-unread field fills with. `Valid()` rather than a validated constructor
  because the real construction path is `json.Unmarshal` of a critic's reply, which no
  constructor intercepts.
  **And the entry's own constraint got a structural guarantee instead of a discipline.**
  *"Must not be reachable by `Severity`"* now holds because `Unexamined` sits beside
  `Diagnostics` on `Result` and `HasBlocking` iterates diagnostics only. Pinned by
  `TestUnexaminedCannotBlock`, checked against two planted defects — `Valid` ignoring
  `Reason`, and `HasBlocking` folding gaps in — and it fails on both.
  One distinction recorded in the doc so nobody unifies them later: this is **testimony**,
  a critic's claim about its own behaviour, where gnosis's `Skip{Check, Reason}` is
  **derived**, code stating mechanically that a check did not apply. Same shape, different
  epistemic status, deliberately different types.
  Consumer adoption is filed in canonizer and adh; both pin v0.18.0 and ride the same
  release as the `skilllens` category constants.
- [x] **Extract the checkable claims a document makes about its own repo.** Two consumers,
  both present: exegesis (a skill body naming `make verify` or `bw sync` makes a claim nothing
  checks; `lint` covers links and `skillsaw` dim 6 covers reachability, so **commands are the
  uncovered half**) and adh (`doctor` checks harness integrity and `context verify` checks
  drift; neither checks that an instruction's commands resolve). Same source for both —
  `agent-blue/agentic-harness-bootstrap` Principle 1, whose `verify-harness.sh.tmpl` parses
  the module table out of `ARCHITECTURE.md` and checks each path exists.
  **Split the promotion at the environment boundary**, which is the design decision: finding
  command-shaped claims in a body is a pure text function and belongs here; *resolving* one to
  an executable is environment-dependent in a way link resolution is not, and belongs in the
  consumer. Evidence out, policy to the caller — the same boundary `skilllens` draws.
  exegesis's caution carries up: because resolution is environmental, this is a **warning**
  tier and probably opt-in, never a hard gate.
- [ ] **EVALUATED and NOT promoted 2026-08-17: OKF's trust fields (`generated`/`verified`).**
  Three repos reference the Open Knowledge Format (`agent-blue/knowledge-catalog/okf/SPEC.md`,
  v0.2, Apache-2.0) and I recommended promoting its trust vocabulary as the strongest
  candidate in the family. **Checking the consumers says otherwise, and that recommendation
  was wrong.** All three needs are conditional and none is present:
  canonizer wants it *"if rulesets ever carry provenance metadata"*; exegesis records
  *"Not scheduled: this waits on the knowledge-base decision"*; adh's is a follow-up inside an
  already-closed item. Three repos wanting a thing *if something else happens* is one
  prospective consumer counted three times, and this file already says one prospective
  consumer is not evidence for a type.
  **The cautionary precedent is in this module.** `skillet/provenance` is tested, complete,
  and has **zero importers anywhere** — its own doc reads *"provisional… a speculative
  extraction awaiting its first use. Delete it if it stays unused."* An OKF type promoted now
  ships with the same comment.
  **REVIEWED 2026-08-22: the decision stands for the storage types, and three things
  changed.** Two of them strengthen it and one is a defect the first pass could not have
  seen.

  - **A third argument against the record types appeared, and it is the strongest one.**
    gnosis's `okf` package now exists, and building it established that **re-encoding YAML
    cannot round-trip** — every encoder normalises quoting and key order, and comments do
    not survive a decode — so gnosis retains the frontmatter block *verbatim* and re-emits
    it. A `Generated`/`Verified` struct here would therefore be **decode-only**: usable for
    reading, useless for the write path the one real consumer actually has. The first
    evaluation rejected the promotion on consumer count; it is also the wrong *shape*.
  - **`skillet/provenance` still has zero importers** across all five consumer repos. The
    cautionary precedent is unrefuted.
  - **gnosis adopted `stale_after` (§5.5) with no shared type and no friction** — parsed in
    `bundle`, stored on `Document`, read by `lint`'s `stale` check and by `show`. The
    freshness half of "adopt OKF" answered itself empirically, which is evidence this was
    never one question.

  **The trigger as written fires too late, and gnosis proved it.** *"The first repo that
  actually stores trust metadata"* is both ambiguous — gnosis stores `stale_after` today,
  which is OKF §5.5 and not the trust family — and misaimed. `internal/gnosis/actor.go` is
  built, tested, and shipped with a **closed three-kind enum** (`human:` / `agent:` /
  `check:`) whose `ParseActor` rejects everything else, while SPEC §14.1 states it
  implements OKF §5.3's fold verbatim. Two of OKF §7's three actor forms —
  `<producer>/<version>` and `process:<id>` — do not parse. **That divergence was written
  without touching trust metadata at all**, so a storage trigger could not have caught it.

  **Replacement trigger: the second repo that classifies an actor or derives a trust
  tier.** Storage is not the event; classification is, because that is where getting it
  wrong is silent — a mis-classified tier reads as a stronger claim than it is.

  **Two repos have now implemented OKF's tier vocabulary and neither derives it as
  specified — which is the demand this entry called hypothetical, arriving as drift
  instead.** adh's `contextstore.Unit.Verified` is a `TrustTier` *string* holding
  `unverified` / `machine-confirmed` / `human-reviewed` directly, with `Valid()` and a
  `Rank()` for routing tie-breaks. OKF §5.2's `verified` is a **list of `{by, at}`
  events** and §5.3 folds the tier out of it. Same field name, same three tier names,
  different type: adh stores the answer where the spec stores what the answer came from.
  gnosis diverged the other way — it specified the fold correctly (§14.1) and built an
  actor parser that rejects two of the three forms the fold must accept.

  canonizer, the third referencing repo and the one this entry once called *"the nearest of
  the three to real"*, has now been re-ranked in its own file: it is third rather than
  nearest, it stores no tier, and its entry records that if it ever spells the tier names it
  should spell them from the spec rather than from a sibling — because the match between the
  two existing implementations is luck.

  **The `okf-fold` hold reported `met` on 2026-08-23 and was wrong; converted to `manual`.**
  Two separate defects, and both are worth keeping because each would recur on a different
  hold. **A tier name is not a fold**: the pattern was `machine-confirmed`, both matches
  were real code, and neither derived anything — adh stores the answer in a `TrustTier`
  string and gnosis's `FoldTrust` was an *untracked* file. Counting the vocabulary cannot
  detect use of the mechanism, and here the two diverge precisely because one repo has the
  words without the fold. **And the scan reads the working tree**, so untracked work counts
  toward a threshold — a hold can fire on a file in no commit. Recorded in invigilator's
  backlog, since whether that is the right default is its decision rather than this file's.
  No pattern replaces it: *derives a tier from an actor list* is a claim about what a
  function does, and matching it needs the type-aware query invigilator declines to build.
  **Neither has tripped the new trigger**, and that is the right call rather than a
  technicality: adh derives nothing, and gnosis has not built the fold. But the tier names
  matching across two independent implementations is luck, not agreement, and it is worth
  recording that the thing eventually promotable may be smaller than the fold — **the
  three tier names and their order**, which is a five-line type both repos have now spelled
  by hand. Hold anyway. A shared enum with two consumers and no shared behaviour is a
  header file, and the family's rule is about behaviour.

  **And when it fires, promote the fold, not the records.** §5.3's classification is a pure
  function over a list of actor strings with a contract §7 states outright: *"Consumers
  that classify trust key off the `human:` prefix."* Only `human:` needs recognising;
  everything else is non-human by definition. That piece has no serialisation surface, no
  round-trip problem, and a silent failure mode — which is the profile that earns a shared
  definition. The record types do not follow it up.

  **What gnosis should do locally, and it needs nothing from here:** two populations, two
  treatments. Keep the closed enum for actors gnosis *mints* — the type comment defending
  it is right, since a `check:` that could pass for a person makes §10.6.4's count wrong in
  the flattering direction — and give OKF frontmatter a separate permissive read that only
  asks *is this `human:`?*. Recorded in gnosis's TODO as a conformance test to write before
  §14.1 is built, so the divergence becomes a decision with a reason rather than a surprise.

  **The spec details worth not re-deriving**, read from the spec rather than from the three
  TODO summaries:
  - §5.2 — `generated: {by, at}` and `verified: [{by, at}]`, distinct *because who wrote a
    concept need not be who confirmed it*. A bare mapping MUST read as a one-element list.
    `verified` is independent of `generated.at`, so *changed without re-confirmation* and
    *re-confirmed without regeneration* are separately representable — which is precisely the
    distinction canonizer's `anchor-absent` split needs.
  - §5.3 — trust tiers are a fold over an actor prefix: no `verified` ⇒ unverified;
    non-`human:` only ⇒ machine-confirmed; any `human:<id>` ⇒ human-reviewed.
  - §7 — actors are `<producer>/<version>`, `human:<id>`, `process:<id>`.
  - §5.5 — `stale_after` is an absolute `YYYY-MM-DD`, deliberately not a TTL, so staleness is
    a date comparison with no reference to read time. That reasoning is this family's house
    standard, stated by someone else.
  - §11 — consumers **MUST NOT** reject a concept for missing any optional family, so
    adoption is incremental by specification.
  When it does land it belongs **in the one consumer that needs it** until a second appears,
  exactly as `quotecheck` stayed in exegesis.
- [ ] **Adjudication is a distinct artifact from detection, and has no type yet.** When two
  rules conflict and a human picks one, the decision is knowledge present in neither source
  — so it can carry no `↦` anchor and **fails `verify.Provenance` by construction.** That
  is the highest-value thing the team produces and the corpus has nowhere to put it. Shape,
  when a second consumer wants it: a supersession edge (`supersedes`, plus the warrant —
  who, when, which review) beside `SourceAnchor` in `ruleset`, so an adjudicated rule is
  *sourced differently*, not *unsourced*. Hold until the knowledge-base tool is real; one
  prospective consumer is not evidence for a type.
  **REVIEWED 2026-08-22: still held, and the stated reason is now wrong.** Three things.
  **There are three specifications of this, not one prospective consumer.** canonizer's
  entry, this one, and gnosis's SPEC all describe it, and gnosis's uses the same sentence —
  *"an adjudicated claim is sourced differently, not unsourced."* So "one prospective
  consumer is not evidence" no longer describes the situation and cannot be the reason to
  hold.
  **The real blocker is L745, and this is its second asker.** A per-rule warrant cannot be
  frontmatter, so it is a marker line — an **optional field on the canonical form**, which
  that entry says cannot be added safely until the format version ships: an unknown marker
  is folded into `Rationale` rather than rejected, so a file written by a newer version
  silently mis-parses in an older one. The subject slot was the first asker and this is the
  second, which strengthens the case for shipping the version reader alone and early.
  **When it unblocks, the shape is deliberately smaller than gnosis's.** `ruleset.Rule`
  gains `{By, At, Rationale}` with rationale required, and nothing else. No tiers, no
  co-signers, no reversal links — those belong to gnosis's §10.6 authority model, and
  putting them here would export one consumer's governance to four. gnosis keeps its own
  warrant and will not use this one: its `Actor` is a deliberately closed three-kind enum
  because §10.6.4 counts distinct humans, and a shared warrant with `By` as a plain string
  is strictly weaker. **A canonizer-only type living in the kernel is `provenance` and
  `auditlog` a third time**, both of which left this repo the same day this was reviewed.
  Policy stays with the consumer: canonizer gates `Provenance` on *warrant present* where
  the anchor is absent. Same split as the `skilllens` category names and the manifest
  test-prompts hash — kernel carries the datum, consumer keeps the decision.
  **One thing worth extracting now, at no cost, because it is not the type.** The shared
  part is the *rule*: an adjudicated artifact is sourced differently rather than unsourced,
  and a check that protects evidence must not reject the one artifact that cannot carry
  any. Three repos derived that independently; writing it once is what stops a fourth
  re-deriving the false rejection. gnosis's formulation is the sharpest and should be the
  one quoted — *"a decision that weighed two published positions names both, even though
  the decision appears in neither."*
- [x] **`finding.Diagnostic` needs a who-acts axis.** DONE 2026-08-15: `Action` with
  `automatic` / `guided` / `human`, orthogonal to `Severity` and documented as such, since the
  whole risk is a reader collapsing them.
  **The zero value is the unclassified state and there is deliberately no `ActionUnknown`.**
  "Nobody classified this" is not the claim "a human is required" — same distinction
  `timeseries.Verdict.Compared` keeps — and a named constant invites being set deliberately.
  Incidental find: `Result.Add` has **zero callers anywhere in the family**, so taking a
  pointer to satisfy `hugeParam` broke nothing. `finding.Result` itself has 19 callers in
  canonizer, so the type stays; the unused method is worth a look at the next survey.
  Original entry: Two repos ask for the same thing independently, which is the promotion bar:
  `canonizer/TODO.md` wants it because `loop` and `budget` govern rework rounds — "a rework
  budget spent on findings a human must adjudicate is not the same expenditure as one spent on
  findings the agent can close, and today the two are indistinguishable to the loop" — and
  `skillsaw/TODO.md` wants it because `skillsaw-skill` picks one edit per round, so "is this
  finding safe to apply unattended" is a decision the loop is **already making implicitly**.
  `Diagnostic` is `{Severity, Category, Path, Message}` and lives here, so either it gains the
  axis once or the two consumers invent incompatible vocabularies for it. Severity already
  answers *does this block*; nothing answers *who acts*.
  Prior art in both entries: `AgentLint`'s `fix_type` (`guided` — tool proposes, human
  confirms; `assisted` — tool can generate the fix) and `agentsys`' HIGH/MEDIUM/LOW meaning
  safe-to-auto-fix / needs-context / needs-human-judgment.
  **Not a severity change**, which canonizer states explicitly: `Specificity` stays advisory.
  A fixed classification per check, no new measurement. The axes are orthogonal — a blocking
  finding may be safely auto-fixable, and an advisory one may need a human.
- [x] **Promote `textnorm` from exegesis.** DONE 2026-08-15. `quotecheck` is **not** promoted:
  its 2nd consumer is the knowledge-base ingestion tool, which does not exist yet.
  Verified byte-identical over the 233-skill corpus before exegesis switched — 0 mismatches —
  because a promotion that silently altered normalization would move every `quotecheck`
  verdict in the tree. exegesis now imports it and its copy is deleted; canonizer adopting it
  is that repo's commit.
  It had **no tests upstream**, which is not acceptable for a shared-kernel package; written
  now, pinning whitespace folding, each typographic class, case preservation, and the
  canonizer disagreement itself.
  `staticcheck` then proved the package doc's own point: it proposed rewriting the
  zero-width-space case to `{"ab", "ab"}` — passing, testing nothing — because the character
  is invisible in source. The escapes are deliberate; do not "simplify" them.
  Original entry: `textnorm.Fold`
  already folds whitespace runs and typographic variants for two exegesis guards that must
  not disagree (`quotecheck`, `a2check`); `ruleset/conflict` is the third caller and the
  first outside that repo. `quotecheck` is the stronger find: it is the family's
  **fabrication guard** — does this run of words appear in the source at all — and it is
  the same trust property llmwiki enforces, in a better form. llmwiki validates a
  *byte-exact* substring against the live source (`llmwiki/internal/db/db.go` stores only
  `content_hash`, never the bytes), so a curly apostrophe or a rewrapped line fails it, and
  a moved source makes it unverifiable. `quotecheck` folds first and drops passages under
  `MinPassageWords`, which is why it survives contact with a real corpus. The
  knowledge-base ingestion tool is `quotecheck`'s 2nd consumer; that is what earns the
  promotion, not the survey.
  - [x] **The 2nd consumer needs a third outcome: not-applicable.** **DONE in v0.18.0**,
    and this box was left unticked. `quotecheck/status.go` carries `Status` with
    `Unchecked` as the **zero value**, and `locate` returns it when `len(haystacks) == 0` —
    which is exactly the state described below. `Finding.Missing()` is deliberately false
    for an `Unchecked` finding, so a caller gating on it asks *"did the check find this
    absent"* rather than *"did the check pass"*.
    **Both downstream entries are now reconciled (2026-08-22).** `exegesis`'s *"when it is
    promoted, expect a signature change exegesis does not need"* records that the prediction
    held in both halves — the signature changed and exegesis did not need it. `canonizer`'s
    three-way split records that its `unverifiable` row is already defined and shared, and
    that the remaining work is canonizer's alone: map `Provenance` onto the three states.
    Both now carry the same note about the **zero value**, which is the part worth
    propagating — `Unchecked` being the zero means a `Finding` nothing populated reads as
    *not checked* rather than *checked and clean*, so a caller that forgets to run the guard
    fails closed. A port that spells the state as an absent value loses that.
    Original entry: `gnosis`
    (`~/Documents/git/gnosis/SPEC.md` §4.3) admits sources it cannot archive — a PDF, an
    image, anything binary — as `referenced`: hash and URI recorded, no local text kept,
    deliberately no PDF extractor. For those, **there is nothing to compare a quote
    against**, and that is neither a pass nor a failure. Today `Passages`/`Segment` answer
    "which quotations are absent from the source"; a caller with no source text can only
    fake an empty haystack, which reports every quote as fabricated — the worst possible
    default, since it would either block admissible claims or teach a caller to skip the
    guard.
    Shape: keep the pure core exactly as it is and let the *result* carry the distinction —
    a checked/unchecked flag beside the absent list, so "no source text was available" is a
    stated fact rather than an inferred zero. This is the same discipline as
    `timeseries.Verdict.Compared` and `stats`' empty-input handling: absence of a comparison
    is a distinct state, never a passing one. exegesis has no `referenced` sources today, so
    this is a gnosis-driven addition to a shared package — recorded here so the signature
    change has a reason attached when exegesis sees it.
- Note for whoever picks this up: pair comparison is O(n²) and that is fine. The corpus is
  233 skills; a ruleset is tens of rules. Do not build a candidate index until a
  measurement says the quadratic hurts — `skillex` (`agent-red/skillex`) is the reference
  design if one is ever needed, but an index bought on speculation is a second source of
  truth about which pairs exist.

## Agent-Fuschia Survey (2026-08-18)

Source: a survey of `~/Documents/agent-fuschia` (26 repositories), driven by the gnosis
work. Three items, each checked against the code here as well as there.

- [ ] **`finding.Category` is an untyped string while `Severity` and `Action` are typed.**
  `finding.go:46` is `Category string \`json:"category,omitempty"\``, so nothing prevents
  two tools — or two checks in one tool — from spelling the same failure differently, and
  `Sort` orders on a free-form field. `agent-fuschia/vac-protocol` §4 takes the other road:
  a **closed vocabulary of nineteen named reasons** (`missing-artifact`, `sha256-mismatch`,
  `unlisted-file`, `summary-mismatch`, `summary-outruns-checks`, `stamp-mismatch`,
  `issuer-commit-mismatch`, `unsafe-archive`, …), "one named reason per failure", with the
  vocabulary written into the spec so a consumer can exhaustively handle it.
  **The tension is real and this entry does not resolve it:** a closed enum here would be a
  kernel type that every consumer's private categories must fit, and exegesis, skillsaw,
  adh, and canonizer have genuinely different failure taxonomies — which is presumably why
  it was left open. The tractable middle is a *registration* seam rather than an enum: each
  consumer declares its category set once, `finding` offers a validator, and an unregistered
  category is a programming error rather than a silent typo. Do not build until a second
  consumer has actually mis-spelled one; recorded so the option is visible when it happens.
  **DECIDED 2026-08-22, and the trigger had already fired unnoticed.** It happened in the
  worst available place: **exegesis emits `skilllens-softening` and canonizer emits
  `softening` for the same `skilllens.SofteningPhrases` call** — one kernel detector, two
  names. Measuring the rest settled the design, because the numbers point away from both
  options this entry proposed. Across all thirty category values in the family there is
  **not one same-word-different-meaning collision**, and the near-misses are all synonyms
  (`unsupported` / `drift-unsupported`, `duplicate` in two repos, `conformance`). These
  tools grade one conceptual domain, so when two reach for a word they usually mean the
  same thing.
  - **The closed enum is refused.** Thirty values with near-zero semantic overlap; the enum
    would be a union of private vocabularies, and every new check anywhere would become a
    kernel PR plus a release plus five bumps — taxing the activity that should be cheapest.
    It also promotes policy (which failures exist) rather than mechanism.
  - **The registration seam is refused, and this is the decisive one:** it would not have
    caught the defect that occurred. `softening` and `skilllens-softening` are each validly
    registered in their own repo. Registration catches drift *within* a consumer; the drift
    was *between* consumers.
  - **`Category` stays an untyped string**, because the risk is not proportional to shared
    vocabulary. It is proportional to **shared mechanism**. Two tools naming their own
    private domain failures differently is not a defect; two tools naming one kernel
    detector's output differently is.
  **The rule, which generalizes past this case: where the kernel owns the detector, the
  kernel owns the category name.** `skilllens` now exports `CategoryNoFailureMode`,
  `CategorySoftening`, and `CategoryNoBoundary` (2026-08-22, three constants and a
  distinctness test — no `finding` import, so the return-spans-not-diagnostics boundary is
  untouched). `quotecheck` is next by the same rule once gnosis and exegesis both run it;
  `redlines`, `speclint`, and `neutrality` follow if they ever feed two consumers.
  **Unprefixed, and deliberately.** A package prefix defends against homonyms — zero
  observed — and manufactures synonyms, which is the one failure that did occur: a second
  mechanism detecting the same class would have to spell it differently. It also puts
  provenance into a field that classifies, the collapse `Severity` and `Action` are kept
  apart to prevent, and it names a package that can be renamed. Twenty-seven of the thirty
  existing values are unprefixed; exegesis's three were the outlier, not the convention.
  **Polarity fixed at the same time, on canonizer's convention** (`no-anchor` = never
  declared, `anchor-absent` = declared and not found). Two of the three detectors fire on
  *absence*, so a category named for the dimension read as its opposite — `skilllens-failure`
  meant *no failure handling was written*. The constants name the defect.
  **Landing sequence, because consumers pin skillet by version and none carries a
  `replace`:** skillet is done and green. Cut a release, then (1) exegesis swaps its three
  literals for the constants — `skilllens-failure` → `CategoryNoFailureMode`,
  `skilllens-boundary` → `CategoryNoBoundary` — and rewrites `lint_test.go:249`, which
  groups on the `skilllens-` prefix and must switch over the constants instead; (2)
  canonizer imports `CategorySoftening`, whose value is unchanged, so **its output does not
  move** — a small confirmation the naming is right, since canonizer reached it
  independently. No production code breaks either way: exegesis, skillsaw and adh have zero
  `.Category` read sites. Filed in both consumers' backlogs.
- [ ] **A set hash beside `identity.Hash`.** `identity.Hash` fingerprints one artifact;
  nothing fingerprints *a collection*. `agent-fuschia/gradecore`'s `suite_hash`
  (`gradecore/freeze.py:20`) is `sha256[:12]` over `"|".join(identities)`, and it exists to
  make "these two implementations agree" a checkable claim rather than an asserted one —
  which is this package's entire reason for existing, applied one level up. Its docstring
  also names the weakness in the scheme it is compatible with: an `id:prompt` fingerprint
  "misses an edited answer-key", so the grader id and expected value must be folded into
  each identity string. Prospective consumers: `manifest` (does this tree hash to what the
  last verify saw), skillsaw (has the rubric changed), gnosis (has the corpus). Hold for the
  2nd — `manifest.Diff` already answers the tree question a different way, and two answers
  to one question is the defect this repo exists to prevent.
- Note, not a work item: `gradecore`'s README **corrects itself in public** — *"'Shared by'
  was the older wording here and it was false in code"* — and replaces the claim with the
  checkable one (same `suite_hash`, all 35 graders lift through `bool_grader`, 0 of 175
  verdicts differing). That is the standard for how this family should word any "these two
  tools cannot drift" claim, several of which appear in this file.
- Deliberately NOT adopted: `agent-fuschia/claim-segmenter-kit`'s deterministic sentence
  segmentation, despite being an excellent fit for the kernel's theme. It is Swift, so
  nothing is importable, and it currently has exactly one prospective consumer (gnosis).
  If canonizer needs the same segmentation for rule bodies — plausible, since a `§` rule
  statement can carry two assertions exactly as a wiki sentence can — that is the 2nd
  consumer and it moves here. Recorded in gnosis's TODO with the algorithm's guarantee
  ("every emitted claim stands on its own, or the cut is not made") so the design is not
  re-derived.

## `superpowers` Deep Read (2026-08-22)

Source: `~/Documents/agent-green/superpowers` at v6.3.0, read after the `agent-green`
survey had filed it twice by size — once as a harness, once as a skill catalogue. It is
neither: it is a **measurement discipline for skills**, and the closest peer this family
has on the axis `skilllens` sits on. Written up in gnosis's `manifesto.md`.

**Two defects below were found by running `skilllens` over that corpus, not by reading
it.** A first pass at this section asserted a third defect — that the detectors count a
skill's own quoted counter-examples as hits — and measurement refuted it:
`SofteningPhrases` returns **zero** spans across all fourteen skills, because the
softening vocabulary is hedges (`as appropriate`, `feel free`) and a rationalization
table quotes *excuses* (`"Too simple to test"`), which is a different vocabulary.
`BlacklistSections` fires on `## Red Flags` and `## Common Mistakes` exactly as intended.
Recorded because the wrong version was plausible enough to have shipped, and because it
is the failure `superpowers` warns about from the other side: *"Manually read every
flagged match… automated counts alone overstate both failure and success."*

- [x] **`markdown.Doc.Prose` does not blank a fenced code block nested inside an HTML
  block, and `skilllens` reads the code as the skill's own instruction.** This
  violates the guarantee stated in `skilllens.proseSpans`' own comment — *"markdown
  has already blanked code blocks and spans, so a conditional inside a shell
  transcript is not read as the skill's own instruction."*

  Minimal reproduction, verified three ways. A fence on its own is blanked and
  `FailureMechanisms` returns nothing. The same fence wrapped in `<Good>`…`</Good>`
  with no blank line is **not** blanked, and the detector returns the example's own
  code as a span. Adding a blank line after the opening tag restores the blanking and
  the empty result.

  Cause: with no blank line, goldmark takes the whole `<Good>`-to-`</Good>` region as
  one HTML block, so the fence inside it is never a `FencedCodeBlock` and `prose()`
  never reaches it. The blank line closes the HTML block and the fence parses
  normally.

  This is not contrived. `<Good>` / `<Bad>` wrappers around code examples are a
  documented convention in `superpowers/skills/writing-skills/SKILL.md`, and the live
  hit is `test-driven-development/SKILL.md:81` — a TypeScript example throwing
  `new Error('fail')`, scored as failure-mode encoding on dims 3/5/9. The fix belongs
  in `markdown`, not `skilllens`: `prose()` should blank fenced content wherever it
  occurs, including inside an HTML block, since the whole point of `Prose` is that a
  caller never has to think about this. Add the three-case table above as the
  regression test.

- [ ] **A blanked code span leaves unreadable evidence text.** The sentence *If (code
  span) fails, stop.* yields a span whose `Text` is the word `If`, then a run of
  spaces where the code span was, then `fail` — the match is semantically right and
  the *evidence* is whitespace. Span length is preserved by
  blanking, so the regex window is unaffected and no score changes; what breaks is the
  thing `Span.Text` exists for. `skillsaw` and `adh` surface these spans to a person as
  the justification for a penalty, and a justification made of spaces is not one.
  Either carry the original substring alongside the matched prose, or record the span
  offsets so a caller can re-slice the raw source. Low severity, and it only shows up
  once somebody reads the output.
  **RESOLVED 2026-08-22: build neither, document the hazard, pre-decide the mechanism.**
  Two of this entry's claims were wrong, and measuring fixed the framing.
  **The field has no readers.** The only consumer of `Span.Text` anywhere in the family is
  `canonizer/internal/verify/verify.go:107`, and it reads a `SofteningPhrases` span — which
  sets `Text` to the **vocabulary term it searched for**, never to matched source, so it
  can never be blank. Section spans carry `sec.Title`, also clean. **Exactly one path can
  produce the whitespace value** — `proseSpans` over the two failure regexes — and all
  three of its consumers count and discard: exegesis tests `len(...) == 0`, skillsaw
  switches on `Kind` to count branches, adh returns 1.0 on the first prose span. So the
  claim that skillsaw and adh "surface these spans to a person" is false; neither reads
  `Text` at all.
  **And a recovered substring would not fix it anyway**, which settles the mechanism
  independently of cost. The pattern allows forty characters between the conditional and
  the failure word, so matches routinely end mid-word — the corpus run produced *"if the
  skill prevents the right fail"* and *"When you have multiple unrelated fail"*. The window
  is an arbitrary cut, so faithfully reproducing it reproduces something unusable.
  **If a consumer ever appears, the answer is offsets**, and the reason is structural
  rather than aesthetic: `Doc` does not carry the body, so carrying the substring would
  mean either a `Body` field — a second full copy of every source, since `Prose` is already
  one — or a signature change across all three detectors. `FindAllStringIndex` costs
  neither, and offsets let a caller widen to a sentence boundary, which a fixed match
  cannot.
  **Landed instead, at no behavioural cost:** the hazard is now stated on `Span.Text`'s doc
  comment with the mechanism pre-decided, and `markdown.TestProsePreservesOffsets` pins the
  precondition any future offset work depends on — that `prose()` copies and masks in
  place, so `Prose` is byte-offset-identical to the source. That property was an
  undocumented accident of implementation that three tools would silently inherit; the test
  was checked against a planted defect (rebuild the buffer instead of masking it) and fails
  on it, so it is a control rather than an assertion.
  **Trigger:** the first consumer that wants to *show* a failure branch rather than count
  one.

- Note, not a work item, and the honest limit of the approach: **methodology prose about
  failure scores as failure-mechanism encoding.** Real hits from the corpus run — *"If you
  didn't watch the test fail"*, *"If the control doesn't exhibit the fail[ure]"*, *"if the
  skill prevents the right fail[ures]"* — are statements *about* testing, not branches
  saying what to do when something breaks, and `(if|when)\s.{0,40}(fail|…)` cannot tell
  them apart. It inflates dims 3/5/9 in the flattering direction on exactly the skills
  most likely to discuss failure as a subject. No fix is proposed: distinguishing them
  needs intent, intent needs a model, and the package is pure by charter. Worth one
  sentence in the package doc so a consumer knows the signal is *mentions of failure
  conditions*, which is what it measures, and not *encodes failure handling*, which is
  what its name suggests.

- [x] **Say what class of skill the three detectors are valid for.** DONE 2026-08-23 as a
  package-doc paragraph: no code, no fourth detector, and nothing that classifies a skill's
  failure type — that is a judgement, and a detector which guessed would put an
  uncalibrated heuristic under a scoring dimension.
  **Refined by measuring rather than warning uniformly: the exposure is `BlacklistSections`
  alone.** `SofteningPhrases` was checked against its own vocabulary — thirteen genuine
  hedges, none a conditional on an observable — and a row-4 skill written *"if the response
  has a non-empty `next` cursor, page again"* scores dim 5 = 10 with no softening flag. A
  uniform caution would have been easier to write and easier to ignore.
  **The reachable harm is in skillsaw, not here, and is recorded there.** A doc comment
  cannot reach the person reading a diagnosis. Reproduced with a purpose-built wrong-shape
  skill: `dim 9 base = 2` for a missing boundary section makes it the weakest dimension, so
  `diagnose` returns *"add counter-examples"* — the form superpowers reports as worse than
  no guidance for that class.
  Original entry: `superpowers`'
  *Match the Form to the Failure* classifies four baseline failure types and pairs each
  with the form of guidance that fixes it — and reports that the form fixing one
  **measurably backfires** on another. Their evidence is a head-to-head of their own,
  not a citation: *"the prohibition arm produced clearly more of the unwanted content
  than the recipe arm (fully separated distributions), and trended worse than even the
  no-guidance control."*

  Their four rows, in order: *skips a rule under pressure* wants a prohibition plus a
  rationalization table and red flags, and is spoiled by soft guidance. *Complies but
  the output has the wrong shape* wants a positive recipe stating what the output IS,
  in order, and is spoiled by a prohibition list. *Omits a required element* wants a
  REQUIRED slot in the template, not prose reminders. *Behaviour depends on a
  condition* wants a conditional on an observable predicate, not an unconditional rule
  with exemption clauses bolted on.

  Row one is the discipline-skill case the SkillLens rubric was validated on, and it is
  where a boundary section and an absence of hedging are the right signals. For rows
  two through four they are not, and a prohibition can be worse than silence. The
  package already draws the correct line on *policy* — *"return the located evidence
  and let each consumer decide what it means"* — and says nothing about **validity
  scope**, so a consumer can run all three over a reference skill and read three empty
  results as three passes. Package doc, not code, and no fourth detector.

- Deliberately NOT adopted: `superpowers`'s `persuasion-principles.md`, which grounds an
  Authority-first recommendation in Meincke et al. (2025), *"Call Me A Jerk: Persuading AI
  to Comply with Objectionable Requests"* (33% → 72% compliance). By its own title that
  study measures defeating refusal training, not improving process adherence, and
  transferring the effect size is the uncalibrated-heuristic class this repo already
  rejected family-wide in the unified-thinking survey above. Their own wording tests are
  better evidence and one level closer to the task; those are what the entry above cites.

- Held for a 2nd consumer: **variance across repetitions as a bindingness metric** —
  *"When guidance lands, reps converge on the same shape. Five different interpretations
  across five reps means the wording isn't binding."* `stats` and `calibration` hold the
  machinery. The one prospective consumer is `skillsaw`, and it is recorded there.

## Commissioned Gap Report — Checked (2026-08-22)

Source: `~/Documents/agent-green/FPF/skillet_topten.md`, ten proposed gaps with Go
implementation plans, one of seven such files covering the family. Checked against this
codebase rather than filed, per the discipline the `hindsight` summary review established:
**a commissioned survey is a claim to be checked, not a finding to be filed.**

**Nothing from it lands here, and the reason generalises.** All seventy findings across the
seven files cite one of two documents, and the larger one — `detailed-corpus-relevance.md`
— is a **re-survey of the same 144 repositories** this family already absorbed
(`agent-blue`/`fuschia`/`green`/`magenta`/`purple`/`red`), naming the same sources:
`llmwiki`, `coherence`, `katalyst`, `qvr`, `stringer`, `4x`, `agentsys`, `ailloy`,
`skillex`, `mdm`. So the findings are largely **this family's own recommendations,
re-derived from the same corpus and re-presented as defects in the tools that already
implemented them.** The tell is that the citations point at a survey rather than at code.

The four items aimed at this repo, and why each was refused:

- **`FixClass` on `finding.Diagnostic`** — `Action` already is the fix class:
  `automatic` / `guided` / `human`, documented as *"who can close a finding"*, with the
  zero value meaning nobody classified it. The proposal's two values are a strict subset
  of three that exist, and its glossary is inverted (it defines `assisted` as fully
  automated). Refused as a duplicate axis.
- **`Certainty` on `finding.Diagnostic`** — the same axis a third time. `agentsys`'s
  HIGH/MEDIUM/LOW maps one-to-one onto `Action`'s three values. **This one was worth
  chasing**: gnosis's SPEC §16.1 proposed it too, describing `Diagnostic` without `Action`
  and arguing that severity does not say who acts. Severity does not and `Action` does, so
  §16.1 has been corrected — the mapping table is now in that section, `certainty` is kept
  only as a *rule about when `Action` may be set* rather than as a field, and the
  `findings` schema drops both columns for `action`.
- **Age-based confidence decay in `stats`/`timeseries`** — decay curves are already
  rejected family-wide as an uncalibrated threshold, recorded against FPF's C.27 in
  gnosis's backlog. The proposal supplies a half-life parameter and no way to calibrate it.
- **A SQLite `evidence` package with `sources(uri, content_hash, body BLOB)`** — this is
  gnosis's tier 0, in the wrong repository. skillet is domain code with **zero**
  `database/sql` imports and a scope boundary that says so; gnosis already stores bytes at
  `evidence/text/<sha[:2]>/<sha><ext>` and one immutable record per source version at
  `evidence/fetch/<h[:2]>/<h>.json`. The proposal is a description of `llmwiki`'s schema —
  including its `internal/db/db.go` path, which exists in `llmwiki` and not in gnosis —
  offered as a fix for a defect gnosis was built to avoid.

Two more were checked and are simply wrong about the code: `skilllens` *"does not inspect
for actionable specificity"* (that is `SofteningPhrases`; the package doc names the
dimension), and `frontmatter` *"bypasses schema validation"* (that is `speclint`, which is
the reason `frontmatter` only splits). The remaining two — parameterised rule templates via
`text/template`, and an `[AUTHORITY:Role]` block in the canonical form — both mutate the
canonical form, and the second walks directly into the open defect above: **an unknown
marker line is folded into `Rationale` rather than rejected.** Adding a field before that is
fixed is the specific thing that entry warns against.

- Worth keeping as a method note rather than a work item: **a re-survey of an absorbed
  corpus reliably produces the absorbed recommendations as fresh gaps**, and no amount of
  depth in the re-survey prevents it, because the corpus genuinely does contain those ideas.
  The cheap discriminator is where a finding's evidence points. A finding citing *the code
  it claims is defective* can be checked in one grep; a finding citing a survey cannot be
  checked at all without redoing the work. Ask for the former.

### Round Two, and What Asking for Code-Reality Verification Actually Bought

A second round arrived the same evening (`FPF/*_todo.md`, 19:05, against `*_topten.md` at
15:30). It is much shorter — four concrete findings across seven files rather than seventy —
and each now carries a **Code-Reality Verification** line, which is exactly the discriminator
the note above asked for. Checked again, and the result is worth recording because it is not
the one the improvement predicts.

**The verification line reads: *"Confirmed via `git diff HEAD` and `TODO.md`."*** Neither
half does what it claims. `git diff HEAD` shows uncommitted working-tree changes, which
cannot establish whether a feature exists — a clean tree produces no output whatever the
code contains. And `TODO.md` is a backlog, not code.

So the second clause is the mechanism, and it explains the result exactly: **the findings
that survived are the ones already written in the backlogs.** All three concrete gaps are
restatements of existing entries, *including the corrections made to the first round*:

| File | Finding | Already at |
| ---- | ------- | ---------- |
| `skillsaw_todo.md` | externalise the rubric, TOML not JSON, fail closed | `skillsaw/TODO.md`, which already credits `skillsaw_topten.md` Gap 1 and derives both those constraints itself |
| `canonizer_todo.md` | map `Provenance` onto three states | `canonizer/TODO.md`, as a **three-way** split with the states named, and already noting the `quotecheck` v0.18.0 prerequisite is met |
| `exegesis_todo.md` | artifact drift gate, diff-scoped | `exegesis/TODO.md`, under a heading reading *"Commissioned Gap Report — One Item Survives"*, which also records why the first round's mtime proposal fails |

The corrections are the tell. Round one proposed JSON and an mtime comparison; those were
refused *here*, in these files, with reasons. Round two proposes TOML and a diff-scoped
check. A report cannot independently arrive at a correction whose only written statement is
the backlog it read.

**The method note therefore needs a second clause.** Asking a finding to cite code is
necessary and not sufficient, because *"I checked the repository"* can mean reading the
backlog, and a backlog is where this family writes down its conclusions. The stronger form:

> **Verification must name the symbol, file, and line it inspected, and a finding whose
> evidence is the backlog is a finding the backlog already has.** The four "no gaps found"
> verdicts in round two are the honest output of exactly this process, and they are its most
> accurate part.

- Worth recording separately, because it is checkable and it is the reason nothing from the
  addenda was taken: **the "Systems-Thinking & Cybernetic Mappings" citations do not survive
  a lookup.** `canonizer_todo.md` §1 and `steve-skill-market_todo.md` §1 cite an *identical*
  list of source lines — same five, same trailing "and 2 other articles" — for two unrelated
  recommendations. And the lines are mislabelled: 123–124 of `book_corpus_findings.md`, given
  as `cloudstrategy_book.md` and `eip_book.md`, are extracts from Russell's *A History of
  Western Philosophy* ("the terror of cosmic loneliness"; the Pythagorean "ethic which
  praised the contemplative life"), and 187/201, given as agile-organisation posts, are
  `cli-guidelines.md`. A third item cites "65 other articles" for the proposition that
  backend and frontend systems have different performance profiles.
  This is the failure gnosis's whole evidence apparatus exists to make impossible, arriving
  as a document *about* this family's tooling: a citation that names a source, points at a
  line, and does not support the claim — and that survives review because nobody looks up
  the line. It is the best specimen of the problem anyone has handed us, and it is recorded
  in gnosis's manifesto as one.

## Two Provisional Packages, Both Criteria Resolved (2026-08-22)

Recorded because these were surfaced in review on 2026-08-22, stated as "the criterion has
fired and the decision is unmade" — and then **not written down**, which is the failure the
same review had just named two paragraphs earlier: *promote-on-second-consumer is a good
rule and it has no observer.* Both exit criteria live inside a `[x]` item above, so neither
appears in the open list and neither is checked by anything.

- [x] **Delete `provenance`. Its own criterion says so and has said so for three surveys.**
      **DONE 2026-08-22** — `git rm -r provenance`, plus its row in the README package table.
      Build, vet, test and `golangci-lint run ./...` clean afterwards, and no consumer needed a
      change because there were none. Recorded for the release notes: an exported package left
      a published module, which is permitted at v0.x and should still be a line rather than a
      silent removal. The OKF entry above still cites it as a cautionary precedent; that
      argument survives the package as history and the citation now reads as past tense.
      The exit criterion recorded 2026-08-05 is *"deleted if it is still unused at the next
      survey rather than carry unused public surface"*, and the package doc repeats it —
      *"a speculative extraction awaiting its first use. Delete it if it stays unused."*
      Measured 2026-08-22: **zero importers across all five consumer repos**, zero internal
      use in skillet, not wired into skillet's own modelith render workflow, and its only
      importer is `provenance_test.go`. Three surveys have passed since the criterion was
      set (agent-fuschia 08-18, agent-green 08-21, the deep reads 08-22). 168 lines of
      production code and 64 of test.
      **Nothing found in favour of keeping it.** The one thing that might have — that
      skillet is itself authored with modelith, and this package was generalized from
      modelith's vendored header — does not hold: skillet's own render path does not use it.
      Two notes for whoever runs `git rm`. It is a **public package in a published module**,
      so the survey covers this family and not the internet; that is acceptable at v0.x, and
      worth one line in the release notes rather than a silent removal. And the OKF entry
      above cites `provenance` as its cautionary precedent — the argument survives the
      package as history, but the citation should say "deleted 2026-08-22" so a later reader
      does not go looking for it.
- [x] **Decide what `auditlog`'s resolution means, because it resolved the other way.**
      **DECIDED and DONE 2026-08-22: it goes home to its consumer.** The entry below reasoned
      toward keeping it as a recorded standing exception; the better answer is that a
      single-consumer package in a shared kernel is surface every other consumer pays for and
      none of them uses, and "standing exception" is how that surface becomes permanent.
      Moved verbatim to `skillsaw/internal/auditlog` — it had **zero skillet dependencies**
      (stdlib only), so the move was a copy, three import repoints, and a `gofmt`. skillsaw
      builds, tests and lints clean and no longer imports it from here; skillet does the same
      with the package gone. Its doc now records why it left and that promoting it back is the
      right move if a second tool ever wants an experiment log — which is cheap precisely
      because it stayed a clean stdlib-only unit.
      **The general lesson, which is the one worth keeping:** `auditlog` and `provenance` were
      both extracted on the guess that a second consumer would appear. Neither did, over five
      months. The promote-on-second-consumer rule exists to prevent exactly this and both
      predate it being applied consistently; the rule's converse — *demote on second survey
      without one* — had been written down for `provenance` and never for `auditlog`, which is
      why one had a criterion to fire and the other had to be reasoned from scratch.
      Its criterion was *"earns the 'shared' designation on a 2nd consumer"*, which reads as
      a wait-for-evidence hold. The evidence arrived and said no: gnosis considered it for
      the mutation-row job and **declined in writing** — SPEC §15 records *"an earlier draft
      named `skillet/auditlog` for this and that package is the wrong shape — it reads
      `results.tsv`, nine columns describing an optimization experiment."* One consumer
      (skillsaw's `cmd/history`, 3 files) and a documented refusal from the only candidate is
      a different state from "still waiting", and the criterion has no branch for it.
      Not the same answer as `provenance` — `auditlog` is *used*, so the unused-surface
      argument does not apply, and it should almost certainly stay. What needs writing down
      is that it stays as **single-consumer code that happens to live in the kernel**, which
      is a standing exception to the promote-on-second-consumer rule rather than a pending
      promotion. Say so, and the hold stops looking open.
- [x] **The general fix, since this is now three for three.** `provenance`'s criterion,
      `auditlog`'s criterion, and `quotecheck`'s not-applicable box all fired without anyone
      noticing, and the OKF trigger turned out to be unfireable by construction. A
      `skillet triggers` report — each hold, its stated condition, and the current consumer
      count computed by grep across the family — is perhaps forty lines and would have
      caught all four. Until it exists, every criterion in this file is a note to a reader
      who has to already be looking for it.
      **DONE 2026-08-22: `holds.toml` + `bin/triggers.sh`, over
      `github.com/StevenACoffman/invigilator`.** Nine holds declared, eight after one was
      withdrawn — see below. `bin/triggers.sh ROOT...` prints a verdict per hold and exits
      2 when any condition is met; roots are arguments and there are no defaults, because
      which checkouts constitute "the family" is a fact about one machine.
      **The design point is not the counting.** Of the four failures this entry cites, only
      three were countable — the OKF trigger was *unfireable by construction*, and no
      amount of grepping would have found it. So every hold declares a `kind`, and a
      condition with no mechanical test is `manual`: still listed, never resolved, visible.
      An unfireable condition and one that simply has not fired look identical from
      outside, and that is the failure the `kind` field exists to separate. Four of the
      eight are manual today, which is the honest proportion and would have been invisible
      in a report that only counted.
      `unknown` is the zero verdict, so a failed scan or a mistyped root reads as *not
      measured* rather than as *no consumers* — the same rule as `quotecheck.Unchecked`.
      **It found something on the first real run.** `okf-fold` sits at 1 of 2: `adh` already
      carries the tier vocabulary in `contextstore.Unit.Verified`, which is the divergence
      recorded against the OKF entry above. One more repo deriving a tier trips it.
      **And it caught a modelling error in its own first draft.** A ninth hold asserted "no
      package is marked provisional", which reported `met` while meaning *all is well* —
      making `met` mean both "act now" and "nothing to do". That is an **invariant**, not a
      deferred decision; it was withdrawn, and `holds.toml` now says so. The CI guard it
      wants — a provisional package that gains an importer must stop being provisional —
      is still worth building and is not this.

## `manifest.Skill` Records Whether Test Prompts Exist, Not What They Say (2026-08-22)

Found while deciding who owns the prose↔test-prompts coupling gate (exegesis and skillsaw
both had a claim; the answer turned out to be neither, quite). The gate needs one fact this
package does not record.

- [x] **Hash the test-prompts file beside the skill's, and give `Delta` the axis.**
      `manifest.Skill` is `{Slug, Dir, Hash, TestPrompts}` where `Hash` is the first-16
      sha256 of `SKILL.md` and `TestPrompts` is *"path if present, else empty"*. So a
      manifest can see a skill's prose change and is **structurally blind to whether its
      test prompts changed with it** — it records that the file exists and nothing about
      its content.
      That asymmetry is the whole reason the coupling defect is invisible: a `SKILL.md` can
      be rewritten while its behavioural assertions still describe the previous version,
      and every gate in the family passes, because the only thing comparing versions is
      `manifest.Delta` and it has nothing to compare on. **One field closes it**, and
      `Delta` then distinguishes *both changed* from *only the skill changed*.
      **Why here rather than in a consumer.** Two tools build manifests (`manifest.Build`
      takes a `tool` parameter precisely so exegesis and skillsaw share it), so the fact
      belongs to the shared entry rather than to whichever tool notices first — and once it
      is here, the coupling check is available to anything holding two manifests, not just
      to the tool that asked. `Delta`'s doc already promises totality (*"every location
      present in either manifest appears in exactly one of the four slices"*); adding an
      axis must preserve that, which is the one design constraint.
      **What this deliberately avoids.** The obvious implementations are filesystem mtimes
      and `git diff --name-only`, and both were rejected: git does not preserve mtimes, so
      an mtime check reports nothing on CI and everything after a rebase, and shelling out
      to git puts an environment dependency inside a pure comparison. A content hash in a
      manifest is what this package already is.
      Consumer side is `skillsaw preflight`, filed there. exegesis owns none of it: it has
      no baseline — it calls `manifest.Build` once and never `Diff` — and a snapshot cannot
      see a drift by construction.

## Four Items Built (2026-08-22)

`markdown` fence-in-HTML, the canonical form's unknown marker, `manifest`'s
test-prompts hash, and the `claims` promotion. What each turned up beyond its entry.

- **A one-line fix was available and wrong.** Blanking the whole HTML block fixes the
  fence and destroys `<Good>Always validate input.</Good>`, which is a real instruction
  reaching `Prose` correctly today. Neither the entry nor the report that prompted it
  mentions that case; it surfaced from asking what *else* lives in an HTML block. The
  chosen fix — a second parse with the HTML block parser removed — keeps it, and there
  is now a test for it.
- **Measuring the blast radius was worth more than arguing about it.** The change to
  `HasCodeBlock` looked like it would make skillsaw stricter for `<Good>`-wrapped
  skills, which would have been a scoring change in another repo. Over 2200 real
  `SKILL.md` files: **16 produce different `Prose` and zero flip `HasCodeBlock`.** The
  coordination item that concern implied does not need writing.
- **Half of the canonical-form entry was already built.** `readFormat` refuses a
  `format:` newer than `FormatVersion` and its own comment says that is what the block
  is *for* — so the cross-version half was closed and the entry did not know. Verified
  before writing anything, and there is now a test pinning it. This is the third entry
  in this family found stale by checking its premise first, and the pattern is
  consistent: **an entry describing a defect is a claim about the code, and it ages.**
- **A heuristic's error rate is measurable, so it was measured.** `claims` shipped its
  first version, ran over 233 real skill bodies, and the head of the false-positive list
  was `defer cancel()`, `go func()`, `getenv func(string) string` — Go statements whose
  first token is a lowercase word. One rule removed the class. Reasoning about the
  heuristic in advance would not have produced that list; running it did, in a minute.
- **Two of `claims`' rules came from failing tests rather than from design.** `var Doc
  struct` parsed as an invocation, and `x = 1` did. The uppercase rule that fixes the
  first also loses `git push origin HEAD`, which is recorded as a test case so the price
  stays visible instead of being rediscovered.
- **`Delta`'s totality promise shaped the API.** A fifth slice was the obvious way to
  report which file moved and would have broken it — a location whose prompts changed is
  also a location that changed. A map keyed by a subset of `Changed` cannot, and the
  subset relation is asserted rather than commented.

- [ ] **The release now carries four changes and two package removals.** `auditlog`
  moved to skillsaw and `provenance` is gone; no consumer imports either, verified by
  grep across the family. Additive: `skilllens` category constants, `Doc.CodeSpans`,
  `Skill.TestPromptsHash`, `Delta.ChangedAxes`, the `claims` package. Behavioural:
  `Prose` blanks more, and `ruleset.Parse` now refuses an unrecognised marker. Consumers
  pin by version with no `replace`, so none of it reaches them until this is cut — and
  the two consumer swaps for the category constants are already filed in exegesis and
  canonizer, waiting on it.

## Two Items Transferred From gnosis's Backlog (2026-08-23)

A re-read of gnosis's `TODO.md` on 2026-08-23 found fourteen entries filed against
sibling repositories, nine of which were already in their real homes and five of which
were nowhere. Two of the five are skillet's.

The accounting is worth one line, because this repository is where the family's
one-home rule is decided: gnosis was still carrying five `skillsaw` items as open that
`skillsaw` had already closed. **A backlog that mirrors another repository's work goes
stale in the direction that flatters.** The rule the kernel already applies to shared
code applies to shared backlogs — one home, and a pointer from everywhere else.

- [x] **Name the object/metalanguage split in `ruleset/conflict`.** DONE 2026-08-23, as a
  paragraph in the package comment. Comment-only: 13 insertions, no code, and build, tests
  and lint all unmoved. Original entry: It is a
  **metalanguage** check: its subject is rules, not the thing the rules are about.
  `conflict.Find` compares two `ruleset.Rule` values for severity divergence, level
  divergence, and section collision — every one of those is a property of the *rules*,
  and none is a claim about the domain either rule governs.
  Saying so in the package comment costs a sentence and buys the thing a future
  contributor will otherwise get wrong: adding an object-level check here because it is
  "the conflict package". An object-level check — two rules that disagree about the
  world — needs a model of the world, which is exactly what this package does not have
  and must not acquire. gnosis reached the same conclusion from the other side when it
  declined to import this package for claim predicates: what the two share is a shape,
  and a shape is followed rather than imported.
- [ ] **A skill is graded against contracts it never claimed, and the entry below names the
  wrong axis for it.** REVISED 2026-08-23 after measuring; the original framing follows.
  **There are two orthogonal axes, not one binary, and the one this entry proposed is the
  smaller half.** Measured with `lint --check redlines` over the corpus:

  | skill                                      | audience       | lineage      | redline errors |
  | ------------------------------------------ | -------------- | ------------ | -------------- |
  | `gh-cli`, `vale`, `unconventional-commits` | shipped        | hand-written | **7 each**     |
  | `book2skill`                               | repo-governing | hand-written | 3              |
  | `go-beyond-packages-as-layers`             | shipped        | book-derived | **0**          |

  The axis firing in `redlines` is **lineage** — book-derived versus hand-written — not
  audience. A shipped hand-written skill collects seven errors for a format it never claimed,
  so a `Kind` carrying only shipped/repo-governing would fix a minority of the misjudgment.
  Both axes are real and independent; `speclint`'s published-artefact contract keys on
  audience, `redlines`' RIA-TV++ contract keys on lineage.
  **Two single-valued closed fields, not a set.** A `Kinds []Kind` makes
  `{shipped, repo-governing}` representable, and a contradiction the type accepts is one
  validation has to catch forever — model the constraint in the type instead. A set also
  invites the bag: once membership is a list, every later convention gets appended rather
  than reasoned about. If only one ships first it should be **lineage**, where the
  seven-error case lives.
  **Closed and typed, and this deliberately inverts the `finding.Category` decision.**
  Category stays an untyped string because its drift is cosmetic — `softening` versus
  `skilllens-softening` named one finding twice and nothing branched on it, so canonizer's
  output did not move when it adopted the constant. A kind's drift is **behavioural**: a tool
  writing `repo-local` where another expects `repo-governing` falls back to a default and a
  *different set of checks runs*. Same rule as Category, applied with more force — where the
  kernel owns the rules a value selects, the kernel owns the value's vocabulary.
  **No aliases.** An unknown kind is a typo or a newer vocabulary, and both want the same
  handling: strictest interpretation, and report it. Silence is what makes a typo dangerous —
  `repo-govening` must not quietly buy lenient treatment. An alias table also manufactures
  synonyms, which the Category entry names as the failure that actually occurred, and it
  never shrinks. Vocabulary evolution belongs to a format version, which `ruleset` now has.
  **DECIDED 2026-08-23: which rules each lineage keeps, measured rather than argued.** This
  was the unwritten list the field had nothing to select without. Decomposing the redline
  errors settles it, because they are not spread across the four rules — they are one rule
  plus one input:

  | redline rule                  | hand-written      | why                                                |
  | ----------------------------- | ----------------- | -------------------------------------------------- |
  | six RIA-TV++ segments         | **drop**          | methodology: how the skill was made, never claimed |
  | quotation ceiling (150 words) | keep              | contract: constrains quotations, not provenance    |
  | description states a trigger  | keep              | contract: about being loadable; also in `speclint` |
  | `test-prompts.json` exists    | **not a redline** | completeness: an unauthored input, not a defect    |

  `gh-cli`'s seven errors are **six missing segments plus one missing `test-prompts.json`**,
  and `gh-cli`, `vale` and `unconventional-commits` trip the quotation ceiling and the
  trigger rule **zero** times between them. So the two kept rules cost nothing to keep and
  are not vacuous — they would fire if broken.
  **The lineage field therefore selects one rule group, not a per-rule table.** That is a
  much smaller thing to get right, and each rule now has a reason rather than a tag:
  methodology travels with lineage, contract does not.
  **`test-prompts.json` leaves the redline set rather than being dropped for hand-written
  skills**, and the distinction matters: dropping it would convert *missing input* into *not
  applicable*, which is exactly what skillsaw's **gate on the artifact, flag on the inputs**
  rule forbids — a hand-written skill with no test prompts is not inapplicable, nobody has
  written them. Its absence is already reported where it belongs: `skillsaw eval` flags
  *"none of N behavioral case(s) specify checks; dim 8 cannot be scored"*. Blocking cannot
  author the file; reporting can prompt someone to.
  **Owed before the field ships:** decompose across all 48 no-RIA skills to confirm the
  six-plus-one pattern holds and that no hand-written skill trips the quotation ceiling. If
  one does, that rule's *keep* is confirmed by a violation rather than by zero.
  **Still not built, and the trigger is unchanged: a second checker that would branch on the
  field.** `Check.Applies`, cited below as the precedent, **does not exist in this module** —
  grep returns nothing — so building to it would be building to a shape that was never built.
  And nobody has written which rules each kind keeps: `speclint`'s description cap plausibly
  survives for a repo-governing skill, its trigger-condition rule plausibly does not, and
  until that list exists the field is a kind nobody branches on.
  `provenance` is the completed cautionary case rather than a live one — carried tested with
  zero importers until v0.20.0 deleted it.
  **exegesis is where this fires**, and its `--check redlines` mixed-tree entry is still open
  with a recorded argument against a derived gate. One question, two repositories.
  Original entry, whose framing the measurement above corrects:
  **A shipped skill and a repo-governing skill are graded against one rubric, and
  should not be.** `gentle-ai` keeps `internal/assets/skills/` — embedded, ships to
  users — apart from `skills/`, which is repo-local and governs work on the tool
  itself. The distinction is real and this family does not draw it: `speclint`'s rules
  are written for a published artefact, and a repo-governing skill is closer to a
  `CONTRIBUTING.md`.
  Grading them against one rubric misjudges both. A shipped skill held to
  `CONTRIBUTING.md` standards under-specifies what a consumer needs; a repo-governing
  one held to `speclint`'s is penalised for omitting an audience it does not have. The
  cheap version is a declared kind on `manifest.Skill` with the rubric reading it,
  which is a smaller change than the two rubrics it avoids — and it is the same
  shape as `Check.Applies`: state which convention applies rather than applying all of
  them.
