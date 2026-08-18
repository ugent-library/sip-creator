# Plan: format identification becomes an optional enricher

*Status: **done** (2026-07-16). Drafted 2026-07-14. Companion to the [refactoring plan](refactoring-plan.md) — depends on its Phase 1 (`store` package, assemble/write split). The decision it enacts is [ADR-0006](../decisions/0006-format-identification-optional.md).*

*Execution notes, deviations discussed:*

- *Sequenced **after** the refactoring plan's Phases 1–2 rather than "with Phase 1" — one variable at a time through the equivalence gate. The `CopyFile (Info, error)` amendment was banked in Phase 1's Step 1; the rest landed here.*
- *`Identify` also fails on siegfried's own per-file error channel (`files[].errors`), not just exec/parse failures — same misconfiguration-must-be-loud principle.*
- *A defect this plan missed: `cli.Run` made a missing `.env` fatal (`cobra.CheckErr(godotenv.Load())`), defeating "unconfigured means skip" — fixed (missing file tolerated, malformed file errors), and the redundant `godotenv/autoload` import (double-loading `.env`) removed.*
- *`formats.New` fixes while nearby: empty `ARGS` now means no arguments (was one empty-string argument), and the "uknown" typo.*
- *Verified per the list below: gate green with siegfried (identical trees, same PRONOM values); without config the build succeeds, `premis:format` is absent, and commons-ip reports the same two FAILED checks (no new ones); missing binary and NAME-without-COMMAND both fail cleanly before any writes.*

## Context — why this change

Siegfried is currently a hard dependency: every essence file passes through `Formats.Process`, and a missing or broken `sf` binary kills the build. That tightness is not warranted by the specs, and the dependency is doing hidden double duty:

1. **The specs don't require format identification at build time.** meemoo SIP 2.0 (representation-level `premis:objectCharacteristics`) makes `premis:fixity` and `premis:size` **MUST (1..1)** but `premis:format`/`premis:formatRegistry` only **SHOULD (0..1)**. E-ARK CSIP requires METS `@MIMETYPE` but no PRONOM or format identification anywhere. Downstream consumers (meemoo ingest, RODA) re-characterise on ingest regardless.
2. **Siegfried smuggles the MUST-level data in with the SHOULD-level data.** `formats/siegfried/siegfried.go` (`sf -hash md5 -json`) is today the *only* source of essence-file `Checksum`, `Size`, and `Created` — the mandatory fixity facts — alongside the PRONOM key/namespace, the merely-recommended facts. The mime type siegfried computes is parsed and discarded; all METS `@MIMETYPE` values are hardcoded literals in `encoders/mets`.
3. **The integration is fragile.** Three latent defects:
   - `siegfried.go:73` — `s.args = append(s.args, f)` mutates the receiver, so call *N* runs `sf` against files 1..*N* and `FirstFile()` returns file 1's result: **every essence file after the first gets the first file's checksum, size, and format**.
   - `siegfried.go:78` — the exec error is discarded (`_ = bin.Run()`); a missing `sf` surfaces as a panic on unmarshalling empty output.
   - `services/config.go:33` — `env.Parse`'s error is discarded, so the `notEmpty` validation on `SIP_FILE_FORMAT_*` is inert; misconfiguration fails late and loudly in the wrong place instead of early and clearly.
4. **The format result has exactly one consumer.** Only the `premis:format` block in `encoders/premis/encoder.go` reads it — and that template dereferences `.Format.FormatRegistry` unguarded, so a no-match file (nil `Format`) breaks rendering.

Direction: fixity/size become native, computed by the tool itself; format identification becomes an **optional enricher** — run it when a tool is configured (satisfying the SHOULD), skip it cleanly when not. Siegfried remains the one pluggable `formats/` implementation.

## Design

### Fixity moves to the store (stdlib, single pass)

`store.CopyFile` (refactoring plan, Step 1) gains the signature:

```go
// CopyFile streams src to rel, computing fixity in the same pass.
func (s *Store) CopyFile(src, rel string) (Info, error)
```

The copy streams through an MD5 hasher (`io.TeeReader` on the source, or `io.MultiWriter` on the destination — `md5.New()`, digest via `hex.EncodeToString`), size and mtime from `os.Stat` on the destination. One read per essence file — deliberately *not* a hash pass followed by a copy pass, which would read large essence files twice. MD5 matches the algorithm the PREMIS fixity block already declares (meemoo permits MD5 and SHA-2 alike). Errors are wrapped and returned.

The store is thereby the single source of fixity for **all** package files — metadata (via `WriteMetadata`) and essence (via `CopyFile`) — and the writer back-fills essence `File` nodes from `CopyFile`'s `Info`, the same pattern it already uses for metadata files. No external library: the computation is a few lines of stdlib, and the small-dependency-footprint rule wins.

### The identificator narrows to what it uniquely provides

`formats.Identificator` currently conflates node creation, fixity, and characterisation (`Process(string) *sip.File`). It narrows to:

```go
type Identificator interface {
	Identify(path string) (*sip.Format, error)
}
```

— named per the CLAUDE.md validation-primitive rule by what it returns. The assembler (refactoring plan, Step 3) constructs `sip.File` nodes itself; if an identificator is configured, it enriches `f.Format` by running against the **source** file.

Failure semantics:

- **Unconfigured** → skipped silently; the build succeeds and `premis:format` is omitted (spec-valid: it is a SHOULD).
- **Configured but the tool fails** (not installed, exec error, unparseable output) → the returned error aborts assembly. Misconfiguration must be loud, not degraded.
- **Tool runs but no match** → `(nil, nil)`; format omitted for that file only.

### Siegfried hardening (`Fixed:` commits)

- Build a fresh argument slice per call — kills the receiver-mutation bug above.
- Return exec and unmarshal errors instead of `_ = bin.Run()` + panic.
- Stop populating checksum/size/created (fixity now comes from `store.CopyFile`); return only the `*sip.Format` (PRONOM key + namespace).

### PREMIS template guard

Wrap the `premis:format` block in `encoders/premis/encoder.go` in `{{ with .Format }}…{{ end }}`. This both implements the optionality and fixes the existing nil-dereference for no-match files.

### Config: `SIP_FILE_FORMAT_*` becomes optional

- Drop `notEmpty` from the `Formats` struct tags: empty/unset `NAME` means "no identificator". If `NAME` is set, `COMMAND` must be set too — validate that pairing explicitly.
- Fix the discarded `env.Parse` error in `services.ConfigFromEnv` (return it) and surface it from `cli.Run`.
- Regenerate `CONFIG.md` (`go generate ./services`); document the optional semantics in the struct comments.
- `README.md`: siegfried moves from prerequisite to optional recommendation.

### Optionality is config-driven, not profile-driven

Whether format identification runs depends on the operator's environment (tool present or not), not on the profile: a profile shouldn't decide whether `sf` is installed. No `Spec` field for it (refactoring plan, Phase 2). Revisit only if a future profile must *forbid* format info.

### Out of scope, noted for later

Essence files currently get a hardcoded `MIMETYPE="text/xml"` in representation METS — wrong for essence, and CSIP makes `@MIMETYPE` a MUST. The identificator's mime output (currently discarded) is a future candidate source. Tracked in [TODO.md](../TODO.md); not part of this plan.

## Verification

1. **With siegfried configured** (current default `.env`): the refactoring plan's Phase-0 normalized diff must hold — structurally identical XML, `premis:format` present with the same PRONOM values. `./build.sh` reports no new FAILED checks.
2. **Without** (`.env` with the `SIP_FILE_FORMAT_*` vars removed): the build succeeds, `premis:format` is absent, and `./scripts/validate.sh` reports no *new* FAILED checks — a SHOULD-level WARNING for missing format info is acceptable and expected.
3. **Multi-file representation**: with siegfried configured, each essence file gets its *own* checksum/size/format (regression test for the args-mutation bug).
4. Negative path: `SIP_FILE_FORMAT_NAME` set but the binary missing → clean error before any writes, not a panic.
5. Go tests: `store.CopyFile` fixity correctness (known bytes → known MD5/size); assembler with a fake `Identificator` (enriched when present, skipped when nil, aborts on error).

## Docs housekeeping (same change as the code, per CLAUDE.md)

- `CLAUDE.md`: the "shells out to the external `sf` binary, which must be installed" line and the required-vars list change to reflect optionality.
- `docs/sip-creator-design.md`: pipeline description and the "format identification must run before PREMIS" ordering note (still true — but only when identification runs at all).
- `CONFIG.md` regenerated; `README.md` prerequisites updated.
- `docs/TODO.md`: remove the items this plan resolves.
