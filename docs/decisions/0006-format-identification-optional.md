# 0006 — Format identification is an optional enricher, not a build prerequisite

Status: **Accepted** (2026-07-14), **implemented** (2026-07-16, [format-identification plan](../archive/format-identification-optional.md)), **superseded in part by [ADR-0009](0009-characterization-as-sidecar-input.md)** (2026-08-18: the capability and its optionality stand; the exec-behind-an-interface mechanism is replaced by pre-computed sidecar input)

## Context

SIP Creator cannot build a package without a working siegfried install: every essence file is run through `sf`, and a missing or broken binary kills the build. Investigating whether that tightness is justified surfaced two facts.

First, the specs don't demand it. meemoo SIP 2.0 makes `premis:fixity` and `premis:size` **MUST** for file objects but `premis:format`/`premis:formatRegistry` (the PRONOM reference) only **SHOULD**; E-ARK CSIP requires no format identification at all. Downstream consumers — meemoo's ingest pipeline, RODA — re-characterise files on ingest regardless of what the SIP claims.

Second, siegfried is doing hidden double duty. It is invoked for format identification, but it is also the *only* source of essence-file checksums, sizes, and timestamps (`sf -hash md5 -json`) — the MUST-level fixity data arrives as a side effect of the SHOULD-level characterisation. That is why the dependency feels load-bearing: unplugging it would silently take mandatory data with it.

## Decision

**Fixity and size are computed natively; format identification is an optional PREMIS enricher.**

- The `store` package computes MD5, size, and mtime in the same streamed pass that copies each essence file into the package (stdlib only), making the store the single fixity source for every file in a SIP.
- The `formats.Identificator` interface narrows to the one thing only an external tool can provide: `Identify(path) (*sip.Format, error)` — the PRONOM registry reference. When a tool is configured, it enriches the file's PREMIS with `premis:format` (satisfying the SHOULD); when not, the build succeeds and the block is omitted.
- Failure rule: *unconfigured* is silent and fine; *configured but broken* aborts the build with a clear error. Degrading a misconfigured tool to a silent skip would hide operator mistakes.
- The choice is config-driven (`SIP_FILE_FORMAT_*`, now optional), not profile-driven: whether `sf` is installed is a property of the operator's environment, not of a SIP profile.

Sequenced in the [format-identification-optional plan](../archive/format-identification-optional.md), a companion to the [refactoring plan](../archive/refactoring-plan.md)'s Phase 1.

## Alternatives rejected

- **Keep siegfried mandatory and just harden the errors.** Rejected: it forces every operator to install and configure a tool the specs make optional, and it leaves MUST-level fixity data coupled to a SHOULD-level feature.
- **Drop format identification entirely.** Rejected: the PRONOM block is cheap to keep behind a seam, meemoo recommends it (SHOULD), and producer-side characterisation is honest provenance. Removing the capability forecloses it for no gain.
- **Pull in an external library for fixity.** Rejected: the computation is a few lines of stdlib (`md5` + `io.Copy`/`io.TeeReader` + `os.Stat`), and the small-dependency-footprint rule wins.

## Consequences

- `SIP_FILE_FORMAT_*` env vars become optional (empty `NAME` disables identification); `CONFIG.md` is regenerated and the README no longer lists siegfried as a prerequisite. The refactoring plan's "no env var changes" assumption is amended accordingly.
- The `premis:format` template block becomes conditional (`{{ with .Format }}`), which also fixes the pre-existing nil-dereference for files siegfried couldn't match.
- Three latent siegfried/config defects are fixed as part of the same work (receiver-mutating args, swallowed exec errors, discarded config-validation error) — see the plan's context for details.
- A SIP built without an identificator will draw SHOULD-level validator warnings for missing format info. That is expected and acceptable; MUST-level validity is unaffected.
- The seam invites future work: siegfried's mime output (currently discarded) is a candidate source for the METS `@MIMETYPE` values that are today hardcoded — tracked in [TODO.md](../TODO.md), not decided here.
