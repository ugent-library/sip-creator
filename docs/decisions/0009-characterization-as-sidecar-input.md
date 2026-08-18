# 0009 — Characterization is pre-computed sidecar input, not an in-process tool

Status: **Accepted** (2026-08-18, with the [characterization-sidecar plan](../plans/characterization-sidecar.md)).
Supersedes [ADR-0006](0006-format-identification-optional.md) **in part**: the
capability (format enrichment) and its optionality stand; the mechanism (exec
behind an `Identificator` interface) is replaced.

## Context

ADR-0006 made format identification optional but kept it in-process: the
`formats/` registry shells out to `sf` once per essence file, reloading the
~10MB signature file on every spawn (the batch problem tracked in TODO), and
the `Identificator` return type discards most of what sf reports — mime (the
candidate source for the hardcoded METS `@MIMETYPE` lie), format name/version,
and all provenance (siegfried version, signature file, scandate). The exec
path also drags an environment surface (`SIP_FILE_FORMAT_*`) that the
library/embeddability boundary explicitly excludes: embedding systems want to
supply data, not configure binaries.

[ADR-0008](0008-bag-layer-out-of-scope.md) set the precedent: bagging was
externalized to reference tooling because the loosely-coupled boundary is the
artifact, not an integration. Characterization is the same class of concern —
a specialist tool's job, done once, whose *output* is what the package
builder needs. The risk a pre-computed report introduces is **staleness**:
files can change after the report is generated, and a stale claim in
preservation metadata is worse than none.

## Decision

**SIP Creator consumes a pre-computed characterization report as optional
input data; it never executes a characterization tool.**

- The CLI input convention reserves `siegfried.json` at the input root
  (generated with `sf -hash md5 -json`), discovered via
  `Definition.CharacterizationSource` — profile-as-data, like
  `DescriptiveSource`. Library callers supply a decoded
  `characterization.Report` directly; the file is one transport, not the API.
- **Optional in contract** (ADR-0006's optionality survives): no sidecar →
  the build proceeds without format info. The meemoo/UGent runbook *requires*
  it operationally, like the external bagger.
- **Fully strict when present**: an unparseable or wrong-shape report, a
  checksumless report (sf run without `-hash md5`), an essence file absent
  from the report, a per-entry sf error, or a per-file MD5 mismatch against
  the source bytes aborts at assemble time — nothing is written. The MD5
  binding is the staleness defense; an entry with empty `matches[]` is an
  honest no-match, not an error. Documentation files are lenient (entry
  optional — they carry no premis:format and may postdate the sf run) but
  checksum-verified when an entry exists: a mismatch proves the report stale.
- The exec path (`formats/` registry, `formats/siegfried`, `SIP_FILE_FORMAT_*`)
  is removed in the same change, not deprecated.
- METS `@MIMETYPE` (a CSIP MUST) is emitted from the report's mime when
  non-empty, else `application/octet-stream` — a characterizer's assertion or
  an admission of ignorance, **never a guess**.

## Alternatives rejected

- **Keep the exec path alongside the sidecar.** Two mechanisms to test and
  document, and the per-file spawn/batch problem survives in one of them.
- **Embed siegfried as a Go library.** Weighed 2026-07-16: trades the
  operational binary dependency for a signature-currency and bus-factor
  dependency in `go.mod`, against the small-and-boring rule.
- **Trust the report without checksum binding.** Staleness would silently
  produce wrong preservation metadata — the worst failure mode this tool has.
- **Make the sidecar mandatory in contract.** The specs make format info a
  SHOULD; ADR-0006's reasoning stands.
- **Extension-based mime guessing as fallback.** A guess recorded as fact in
  a preservation manifest; `application/octet-stream` is the honest unknown.

## Consequences

- The siegfried dependency changes *kind*, not existence: no binary executed,
  no library in `go.mod`, but the file-input path speaks exactly one report
  format (siegfried JSON), contained at a single decode seam
  (`DecodeSiegfried`, mirroring the descriptive decode). A future FITS/DROID
  sidecar is one new decode function, not a registry built in advance.
- `SIP_FILE_FORMAT_*` is removed — **operator-breaking**; README documents
  the migration (stale vars in `.env` are ignored).
- `siegfried.json` becomes a reserved name in the input convention.
- Essence is read twice when a sidecar is present (MD5 verify at assemble,
  copy at write) — accepted for abort-before-write.
- The provenance TODO strengthens: the report header (siegfried version,
  signature, scandate) is now retained source material for a future PREMIS
  format-identification event.
- `build.sh` must regenerate the fixture sidecar (a stale one is a hard build
  failure), making the local CI loop — and the baseline gate — sensitive to
  the local sf install's signature currency: a signature update could someday
  move a PRONOM ID and surface as a baseline diff. The baseline is re-blessed
  deliberately when the MIMETYPE fix lands (plan Step B).
- The future `sip.Package.Validate()` is where a no-empty-Mime invariant
  check belongs.
