# Plan: characterization as sidecar input; fix METS MIMETYPE

*Status: **approved, not started**. Drafted 2026-08-18.
Companion decision: ADR-0009 (characterization via pre-computed sidecar,
superseding [ADR-0006](../decisions/0006-format-identification-optional.md)
in part) — written as Step A2 of this plan.*

## Context

Two connected defects/smells, resolved by one design decision:

1. **The METS `@MIMETYPE` lie** ([TODO](../TODO.md)): the representation METS
   stamps every essence file `MIMETYPE="text/xml"` (`encoders/mets/encoder.go:68`).
   CSIP makes `@MIMETYPE` a MUST (CSIP62, CSIP26, CSIP40 — all 1..1), but only
   *presence* is machine-checked, so both validation gates stay green while the
   manifest misdescribes every content file.
2. **The coupling smell**: format identification runs in-process behind an
   `Identificator` interface (`formats/` registry + `formats/siegfried` shelling
   out to `sf` per file, reloading the ~10MB signature file on every spawn). The
   interface's return type discards most of what sf reports — mime, format
   name/version, and all provenance (siegfried version, signature file, scandate).

**Decision (per [ADR-0008](../decisions/0008-bag-layer-out-of-scope.md)
symmetry — bagging was externalized to reference tooling; characterization is
the same class):** kill the exec path and consume a pre-computed siegfried
report (`sf -hash md5 -json` → `siegfried.json` sidecar at the input root) as
optional input data. The loosely-coupled boundary is *data, not an interface*.
The vestigial `tmp/basic/siegfried.json` / `tmp/eark/siegfried.json` fixtures
(2024, never referenced by any code path) are structurally exactly this format
already.

Decisions confirmed in design discussion:

- Exec path removed in the same change, not deprecated.
- **Optional in contract** (ADR-0006's optionality survives): no sidecar →
  build proceeds, no format info. Required only in the meemoo/UGent runbook,
  like the external bagger.
- **Fully strict when present**: unparseable/wrong-shape JSON, checksumless
  report (sf run without `-hash md5`), essence file absent from the record,
  per-file MD5 mismatch vs the source, or a per-entry sf error → abort at
  assemble time, nothing written. Entry present with empty `matches[]` → nil
  Format for that file (as today).
- **Discovery via profile-as-data**: new `CharacterizationSource` field on
  `profiles.Definition` (the `DescriptiveSource` pattern), `"siegfried.json"`
  for both profiles. Library callers supply a decoded report as data instead.
- **Two gated steps**: Step A (exec→sidecar swap) is byte-equivalent, the
  baseline-equivalence gate stays green; Step B (mime fix) deliberately breaks
  and re-blesses the baseline. One variable at a time.
- `Mime` lives on `sip.File`, not `Format`: the METS attribute is a MUST;
  `Format` stays the optional enrichment. Resolution in assemble, never in
  templates; the field is never empty by write time. **No extension guessing** —
  sidecar mime when non-empty, else `application/octet-stream` (admission of
  ignorance).
- Documentation files: sidecar entry *optional* (they carry no premis:format
  and may postdate the sf run → octet-stream fallback, no abort), but
  checksum-verified **when present** (a mismatch proves report staleness →
  abort).

## Execution checklist

Progress tracking only — each item's full detail lives in the section it
names. Check an item off when its commit lands (or its gate passes).

### Step A — sidecar swap (gate stays green)

- [ ] **A1 — `characterization/` package**: `DecodeSiegfried` + `Report`/`Record`,
      shape guard, key normalization, migrated testdata, unit tests.
      Commit: `Added: characterization package …`
- [ ] **A2 — ADR-0009** written; ADR-0006 marked "superseded in part".
      Commit: `Added: ADR-0009 …`
- [ ] **A3 — the swap**: `Config.Characterization` replaces `Config.Formats`;
      `Definition.CharacterizationSource`; assemble threads `chars` (essence
      strict, documentation lenient-but-verified); `formats/` tree deleted;
      CLI/env unwired + `go generate ./cli`; assemble tests reworked
      (`writeSidecar` helper, new negative cases); docs swept (CLAUDE.md,
      README, `.env.example`, design doc, TODO, input-spec, refactoring-plan
      note). Commit: `Changed: format enrichment consumes the siegfried.json sidecar …`
- [ ] **A4 — build.sh** regenerates the fixture sidecar (capture-then-write;
      warnings when `sf` is absent). Commit: `Changed: build.sh regenerates …`
- [ ] **Gate A**: `go test ./...`; `./build.sh basic` and `eark` → VALID;
      `baseline-diff` → OK (premis:format still `pronom`/`fmt/44`); manual
      negatives — sidecar removed → builds without format info; sidecar
      corrupted → clean abort, nothing written.

### Step B — mime fix (deliberate baseline re-bless)

- [ ] **B — the fix**: `sip.File.Mime` + invariant doc; assemble stamps
      essence/documentation (sidecar mime or octet-stream), schemas
      `application/xml`, descriptive `text/xml`; write stamps `text/xml` on
      metadata nodes; all seven METS template literals → `{{ .Mime }}`;
      tests; docs (TODO MIMETYPE item deleted, design doc, ADR note).
      Commit: `Fixed: METS MIMETYPE emitted from resolved per-file mime …`
- [ ] **Gate B**: `go test ./...`; `./build.sh basic` → VALID, no new FAILED
      checks; `baseline-diff` FAILs on MIMETYPE-only hunks (each reviewed);
      re-bless `tmp/baseline/` + README note → `baseline-diff` OK;
      `./build.sh eark` → VALID; re-bless recorded in the commit message.

## Step A — sidecar swap (gate stays green)

### A1. New package `characterization/`

Commit 1: `Added: characterization package — decode siegfried -hash md5 -json sidecar reports`

Delete-source is `formats/siegfried/siegfried.go`. Public surface (all else
unexported):

```go
type Record struct {
    Format *sip.Format // nil: tool ran, no match
    Mime   string
    MD5    string      // hex digest binding record to bytes
    Errors string      // sf's per-file error, verbatim
}
type Report map[string]Record // keyed by input-relative slash path
func DecodeSiegfried(r io.Reader) (Report, error)
```

- The `Output`/`File`/`Match` structs (`siegfried.go:30-65`) move here
  **unexported** (`sfOutput`/`sfFile`/`sfMatch` — they lose their cross-package
  caller), gaining `md5` and `filesize` JSON fields.
- **Shape guard**: the top-level `"siegfried"` version string must be
  non-empty, else error "not a siegfried report" (any JSON object would
  otherwise decode as an empty report → baffling downstream errors).
- Key normalization: `path.Clean(filepath.ToSlash(filename))` (defends `./`
  prefixes, backslashes).
- First match wins; `Format` built exactly as `siegfried.go:97-101`
  (`sip.NewFormatRegistry()`, `Name←ns`, `Key←id`) — **invariant:
  `FormatRegistry` never nil inside non-nil `Format`** (the premis template at
  `encoders/premis/encoder.go:83-84` dereferences it unguarded).
- Per-entry `errors`/`md5` are carried, not judged — policy belongs to the
  consumer (a whole-tree sf run legitimately contains entries never looked up,
  e.g. the sidecar listing itself).
- Migrate `formats/siegfried/testdata/*` → `characterization/testdata/`,
  extended with md5/filesize; add a wrong-shape fixture.
- Unit tests: match / no-match / errors-carried / malformed / wrong-shape /
  key normalization.

### A2. ADR-0009

Commit 2: `Added: ADR-0009 — characterization via pre-computed sidecar, superseding ADR-0006 in part`

`docs/decisions/0009-characterization-as-sidecar-input.md`:

- **Context**: exec costs (per-file spawns/signature reloads; operational
  binary coupling; env surface the library boundary excludes); ADR-0008
  symmetry; embeddability wants data, not interfaces; staleness is the risk a
  pre-computed report introduces.
- **Decision**: sidecar via `Definition.CharacterizationSource` or a
  caller-supplied `characterization.Report`; optional in contract, required in
  the meemoo runbook; fully strict when present (the MD5 binding is the
  staleness defense); exec path removed; (Step B) sidecar mime → METS
  MIMETYPE with octet-stream fallback, no guessing.
- **Rejected**: keep exec alongside (two mechanisms, the batch problem
  survives); embed siegfried as a Go library (signature-currency + bus-factor
  in go.mod, per the 2026-07-16 weighing); trust without checksum binding;
  sidecar mandatory in contract; extension-based mime guessing.
- **Consequences**: the siegfried dependency changes *kind*, not existence —
  no binary executed, no library in go.mod, but the file-input path speaks
  exactly one report format (siegfried JSON), contained at a single decode
  seam (`DecodeSiegfried`, mirroring `decodeDescriptive`); library callers
  bypass it entirely by supplying `characterization.Report` structs, and a
  future FITS/DROID sidecar is one new decode function, not a registry built
  in advance; `SIP_FILE_FORMAT_*` removed (operator-breaking);
  `siegfried.json` becomes a reserved input name; essence read twice at build
  when a sidecar is present (MD5 verify at assemble + copy at write — accepted
  for abort-before-write); provenance TODO strengthened (report header now
  retained as source material); the baseline gate becomes sensitive to the
  local sf signature currency via build.sh regeneration; baseline re-blessed
  at Step B.
- Mark ADR-0006 status: "Superseded in part by ADR-0009" (capability and
  optionality stand; mechanism replaced).

### A3. The swap

Commit 3: `Changed: format enrichment consumes the siegfried.json sidecar; removed the sf exec path`

**Library surface** (`profiles/builder.go`): `Config.Formats
formats.Identificator` (:17) → `Config.Characterization
characterization.Report` (same on `Builder` :28, `New()` :36). A non-nil
caller report wins; nil → assemble reads the profile's sidecar file.
Strictness applies identically to both transports.

**Definition** (`profiles/definition.go`): add `CharacterizationSource string`
after `DescriptiveSource`; `"siegfried.json"` on both `basic` and `eark`
entries. Reword the stale ":47 mirrors the formats/ registry pattern" comment.

**Assemble** (`profiles/assemble.go`):

- New step at the top of `assemble()` (before `assembleDocumentation` at :37 —
  it consumes the report too): `chars, err := b.assembleCharacterization(def)`
  — caller report / `""` source / `os.ErrNotExist` → nil; present → open +
  `DecodeSiegfried`, errors wrapped with the path, abort assembly.
- Thread `chars` as a **parameter** through `assembleDocumentation`,
  `assembleRepresentations` → `assembleEssenceFiles` (explicit dataflow,
  assemble stays pure).
- Essence (replacing :189-198): key =
  `path.Clean(filepath.ToSlash(filepath.Rel(b.InDir, src)))` — **from
  `Source`, not `f.Path`** (Path flattens nested dirs; pinned by
  `TestAssembleRepresentations`' `representation_2/sub/deep.tif`).
  `chars == nil` → skip. Entry missing → abort with a message that prints one
  sample report key and the canonical invocation
  (`cd <input> && sf -hash md5 -json .`) — makes wrong-cwd sf runs
  self-explaining. Entry `.Errors != ""` → abort. `.MD5 == ""` → abort naming
  `-hash md5`. Streamed MD5 of `src` (small helper next to its caller)
  mismatch → abort "changed since the report was generated". Else
  `f.Format = entry.Format`.
- Documentation (:113-148): entry optional; checksum-verified when present.

**Unwiring**: delete the `formats/` tree entirely (`formats.go`,
`siegfried.go`, `siegfried_test.go` — its 5 tests are exec-specific);
`cli/cli.go:10` blank import; `cli/config.go` `Formats` env struct (:14-26) +
cross-validation (:55-57) + stale comment; `cli/create_cmd.go` import,
construction (:39-47), `Formats:` field (:53). Then `go generate ./cli`
regenerates `CONFIG.md`.

**Test rework** (`profiles/assemble_test.go`):

- `fakeIdentificator` (:17-26) → a `writeSidecar(t, inDir, files...)` helper:
  computes real MD5s, writes a `siegfried.json` with a canned `fmt/999` match
  (mime `image/test` for Step B). Called from `newTestBuilder`; **re-called
  after tests add essence files** (ordering hazard:
  `TestAssembleRepresentations`).
- `TestAssemble` :154-157 assertion unchanged in meaning (fmt/999 via
  sidecar); `TestAssembleWithoutIdentificator` → `TestAssembleWithoutSidecar`
  (Format nil); NoMatch → empty-`matches[]` entry (nil Format, no error);
  IdentificatorError → `TestAssembleSidecarMalformed` (error + outDir empty —
  the pinned nothing-written property).
- New: missing-essence-entry aborts; checksum-mismatch aborts (outDir empty);
  checksumless entry aborts; entry-with-errors aborts; extra entries ignored
  (the sidecar itself, `dc+schema.json`); caller-supplied
  `Config.Characterization` overrides file discovery.

**Docs (same commit, per CLAUDE.md)**: `CLAUDE.md` :28, :37 (formats/ bullet →
characterization/ sidecar semantics + strictness), :38, :88 (build.sh needs sf
on PATH for sidecar regeneration, not `.env`); `README.md` :17-19, :24,
replace the "Siegfried (optional)" section :48-60 with sidecar docs
(`sf -hash md5 -json`, strictness, migration note: stale `SIP_FILE_FORMAT_*`
in `.env` is now silently ignored); `.env.example` :7-12 → pointer comment;
`docs/sip-creator-design.md` :3, :16, :17, :67, :76, :106 (formats/ section →
characterization/), :111 (external-binary dependency line — deleted),
:120-121, :150; `docs/TODO.md` — **delete the batch-identification item**
(dissolved: one sf run per package by construction), **update the
provenance-event item** (the sidecar header's siegfried/scandate/signature is
now retained source material) and the multiple-formats parenthetical →
ADR-0009; `docs/input-spec.md` :41 (fifth reserved name `siegfried.json`),
:57 (rewrite: the tool computes checksums and sizes itself; file formats come
from an optional pre-computed report the tool verifies against the files),
:173; `docs/plans/refactoring-plan.md` — one editorial note under the title
("the formats/ registry pattern referenced below was removed by ADR-0009;
this plan is an execution record"), no rewriting of the six historical
mentions. `docs/plans/format-identification-optional.md` untouched (history).

### A4. build.sh

Commit 4: `Changed: build.sh regenerates the input fixture's siegfried sidecar`

Before the `create` invocation: `command -v sf` → capture
`(cd $SRC && sf -hash md5 -json .)` to a variable, then write
`$SRC/siegfried.json` (capture-then-write so sf never scans its own
half-written output); no sf but a sidecar exists → warn "may be stale — a
stale sidecar aborts the build"; neither → warn "building without format
info". Update the :15 comment. Rationale: the checksum binding makes a stale
sidecar a *hard build failure*, so the local CI loop must refresh it to stay
deterministic. The 2024 vestigial fixtures get regenerated on the first run.

### Gate A

`go test ./...`; `./build.sh basic` and `./build.sh eark` → VALID;
`./scripts/baseline-diff.sh tmp/baseline/pkg basic-uuid/uuid-*/` → **OK**
(premis:format must still emit `pronom`/`fmt/44` — baseline-diff does NOT
normalize it). Manual negatives (mirroring the
[ADR-0006 plan](format-identification-optional.md)'s verification matrix):
sidecar removed → build succeeds, premis:format absent, VALID with a SHOULD
warning; sidecar corrupted → clean abort, no package dir.

## Step B — mime fix (deliberate baseline re-bless)

Commit 5: `Fixed: METS MIMETYPE emitted from resolved per-file mime instead of hardcoded literals`

- `sip/file.go`: `Mime string` on `File`, doc comment carrying the invariant
  (*never empty by write time; a characterizer's assertion or
  `application/octet-stream` — no guessing; CSIP62/26/40 are MUSTs*).
- Assemble stamps: essence + documentation → sidecar mime if non-empty else
  `application/octet-stream`; `schemaFileNodes()` (:98-108) →
  `application/xml` (XSDs; currently mislabeled octet-stream); the descriptive
  node (:67-70) → `text/xml`.
- Write stamps `text/xml` where metadata nodes are born (`profiles/write.go`):
  rep PREMIS :131-133, rep METS :145-147, package PREMIS :163-165, package
  METS :179-181 (unreferenced by any template, set for the invariant).
- `encoders/mets/encoder.go`: all seven `MIMETYPE="..."` literals →
  `MIMETYPE="{{ .Mime }}"` (:59, :68 the lie, :118, :126, :135, :143, :151) —
  the invariant lives in one place, the graph. The PREMIS template is
  untouched.
- Tests: essence Mime from sidecar; octet-stream fallback (no sidecar; match
  with empty mime); schemas `application/xml`; descriptive `text/xml`.
  Writer-stamped mimes have no unit-test home — the baseline diff and
  commons-ip cover them.
- Docs: delete the TODO MIMETYPE item; `sip-creator-design.md` `sip.File`
  inventory (:16) plus a MIMETYPE note; note in ADR-0009 that the future
  `sip.Package.Validate()` is where a no-empty-Mime check belongs.

### Gate B

`go test ./...`; `./build.sh basic` → VALID (verify no new FAILED checks —
`image/jpeg` is safe; odd sf mimes pass through by design). Run baseline-diff
expecting **FAIL**; manually review that every hunk is a MIMETYPE-only change
(essence `text/xml`→`image/jpeg` in rep METS; schemas
octet-stream→`application/xml` in package METS); re-bless:
`rm -rf tmp/baseline/pkg && cp -R basic-uuid/uuid-*/ tmp/baseline/pkg`, update
`tmp/baseline/README.md`; baseline-diff → OK. `./build.sh eark` → VALID (its
documentation file's MIMETYPE now resolves via the sidecar). The commit
message records the re-bless (the baseline is untracked `tmp/` — the message
is the record).

## Risks

1. Double essence read at assemble (MD5 verify) + write (copy) — inherent to
   abort-before-write; streamed; recorded in the ADR.
2. Key-form drift (absolute paths, `./`, backslashes between sf invocations) —
   decode normalization + a pinned runbook invocation + sample-key error
   messages.
3. The shape guard is load-bearing (any JSON decodes as an empty report
   without it).
4. `writeSidecar` must run after tests add files — a helper, not baked only
   into `newTestBuilder`.
5. Never trust the vestigial 2024 sidecars for Gate A — build.sh regeneration
   (or a manual `sf -hash md5 -json`) must precede judging gate results.
6. Signature-file drift could someday move fmt/44 → baseline sensitivity to
   the local sf's signature currency; a sentence in the ADR consequences.
