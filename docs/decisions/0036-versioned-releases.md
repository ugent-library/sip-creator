# 0036 — Versioned releases with go install

Status: Accepted (2026-10-12)

## Context

[ADR-0033](0033-ugent-first-profiles-of-your-own.md) held a tagged release and a changelog until UGent's own use called for them, such as a stable version for a program that embeds the library to pin. That use is here: UGent's operators prepare packages for RODA and Meemoo, and a program that embeds the library needs a version that does not move under it.

Every package already names the software that built it. `build.Version` returns the module version the Go toolchain stamps into the binary, and the METS software agent states it in its `SOFTWARE VERSION` note. Without a tag, that value is a pseudo-version such as `v0.0.0-20261012135305-f7ae667fd50f`, which an operator cannot match to a set of changes.

A handful of UGent staff use the tool. All of them can run `go install`.

## Decision

- **Versions follow semantic versioning, at major version 0.** A minor version may change the command line, the library API or the packages a profile builds. A patch version fixes a defect and changes none of them.
- **A version is an annotated git tag `vX.Y.Z` on `main`.** It is the version `sip-creator --version` prints and every package built from it states in its METS.
- **Distribution is `go install github.com/ugent-library/sip-creator@vX.Y.Z`.** No prebuilt binaries. The GitHub release page carries the changelog section and nothing else.
- **`CHANGELOG.md` is written by hand,** in the Keep a Changelog format, as changes land. Its headings match the commit prefixes.
- **A version is released when UGent's use needs a change on `main`:** a fix an operator needs, or a change a program that embeds the library wants to pin. CONTRIBUTING.md holds the procedure.

## Alternatives rejected

- **Start at v1.0.0.** Version 1 promises a stable API and stable output. The library still changes in both: essence as streams, checksums a program supplies, PREMIS events, and `meemoo/basic` moving to Meemoo SIP 2.1.
- **Prebuilt binaries, built by goreleaser on each tag.** A release workflow and a config to maintain, and binaries that macOS and Windows warn about unless they are signed. The users can run `go install`, which builds the binary on their own machine.
- **Wait for Meemoo SIP 2.1.** The UGent profiles are ready, and the move needs its own plan. A minor version carries it.

## Consequences

- An operator can tell from a package's METS which version built it, and look the version up in `CHANGELOG.md`.
- Every operator needs Go installed, in the version `go.mod` names or later.
- A change an operator or a program notices adds a line to `CHANGELOG.md` in the same commit.
- The README's experimental warning stays while the major version is 0.
