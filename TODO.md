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
  Shape and consumers still to settle. Original framing:
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

## Contradiction Detection — Knowledge-Base Ingestion (Agent-Red Survey, 2026-08-15)

Source: a survey of `~/Documents/agent-red` (26 agent-tooling projects) against the
knowledge-base ingestion gap — taking in outside sources and accreting domain knowledge
without polluting the corpus with contradictory or low-quality material. The finding that
matters here holds across all 26: **every one of them detects similarity; none adjudicate
conflict.** llmwiki surfaces contradictions, mnemon deduplicates, coherence finds broken
support links — nothing decides which of two conflicting claims is authoritative, or
records why.

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
- [ ] **The canonical form cannot gain an optional field safely — fix this before adding
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
- [ ] **A coverage record beside findings: what the critic did NOT examine.** Two consumers,
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
- [ ] **Extract the checkable claims a document makes about its own repo.** Two consumers,
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
  **Trigger: the first repo that actually stores trust metadata** — a stored artifact, not a
  decision to adopt OKF someday.
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
- Note for whoever picks this up: pair comparison is O(n²) and that is fine. The corpus is
  233 skills; a ruleset is tens of rules. Do not build a candidate index until a
  measurement says the quadratic hurts — `skillex` (`agent-red/skillex`) is the reference
  design if one is ever needed, but an index bought on speculation is a second source of
  truth about which pairs exist.
