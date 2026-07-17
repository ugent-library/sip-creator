# SIP Creator — design

SIP Creator is a Go library and CLI that assembles a producer's essence files and descriptive metadata into a standards-conformant Submission Information Package (SIP). It walks an input directory, copies the essence into the package layout (computing fixity natively during the copy), optionally runs format identification over each file, and generates the descriptive, preservation, and structural XML that describe and bind the whole. The output targets are a [meemoo SIP v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) `basic` package (the stable meemoo spec; 2.0/2.1 are release candidates) layered on the [E-ARK](https://earksip.dilcis.eu/) profile, and a plain E-ARK SIP (`eark` profile) for RODA-class repositories. The BagIt envelope meemoo's transfer requires is deliberately out of scope ([ADR-0008](decisions/0008-bag-layer-out-of-scope.md)): bag the package directory with a reference BagIt implementation.

This document describes **what the system is today** — the domain model, the package layout it produces, and the build lifecycle. It reflects the code as it stands, including its known gaps and rough edges; where the code is mid-evolution, that is called out rather than smoothed over. The *why* behind key choices lives in the [decision records](decisions/); planned changes live in [plans/](plans/); the backlog lives in [TODO.md](TODO.md). See [README.md](README.md) for how the docs are organized, and the repo-root `CLAUDE.md` for coding conventions.

> **Status: experimental, work in progress.** Two profiles are registered: `basic` (meemoo SIP 1.2) and `eark` (plain E-ARK SIP). Both sample packages validate **VALID** against commons-ip, each against the E-ARK spec version of its era (2.0.4 and 2.2.0 respectively).

## Domain model

The domain lives in the `sip/` package as a graph of plain structs, built entirely in memory before anything is written to disk. It mirrors the OAIS/METS vocabulary: an intellectual entity, its representations, and the files within them.

- **Package** (`sip.Package`) — the unit of work: one SIP. Carries its `Identifier` and `Location` (the on-disk package directory), a single root `Entity`, and the generated top-level files (package `MetsFile`, package `PremisFile`, `SchemaFiles`). Created by `NewPackage(baseDir)`, which mints the identifier and sets `Location = baseDir/uuid-<uuid>`.
- **Entity** (`sip.Entity`) — the intellectual entity being preserved (the "what this is"). Holds its `Identifier`, a map of `AdditionalIdentifiers` (e.g. `MEEMOO-LOCAL-ID`), its `Representations`, its `Description` (the decoded descriptive metadata, a `sip.Descriptive`, awaiting serialization by the writer), its `DescriptionFile` (the descriptive-metadata file node), and a slice of sub-`Entities`. The sub-entity slice exists in the model but the `basic` profile only ever builds a single root entity with no children.
- **Representation** (`sip.Representation`) — a typed grouping of the essence (e.g. a master vs. a derivative). Holds its `Label` (the source directory name, e.g. `representation_1`), `Identifier`, its `Files`, its own representation-level `MetsFile` and `PremisFile`, and a back-reference to its `Entity`.
- **File** (`sip.File`) — one file in the package, whether essence or generated metadata. Records `Identifier`, `Name`, `Checksum` (MD5), `Size`, `Created` (RFC3339Nano), `Path`, `Source` (the absolute input path an essence file is copied from; empty for generated metadata), and — for essence — a `Format`. `Path` is the href **relative to the METS document that references the file**: essence and representation PREMIS are representation-relative, schema/descriptive files and representation METS entries are package-relative. The assembler declares `Path` up front; the writer back-fills `Checksum`/`Size`/`Created` as each file lands on disk.
- **Format** / **FormatRegistry** (`sip.Format`, `sip.FormatRegistry`) — the format-identification result attached to an essence file: the PRONOM registry key and name from Siegfried, plus a `Role` of `specification`. Nil when no identificator is configured or the tool found no match ([ADR-0006](decisions/0006-format-identification-optional.md)); the PREMIS template omits `premis:format` accordingly.
- **Identifier** (`sip.Identifier`, `sip.UUID`) — an interface + a UUID implementation. **Note:** this type is defined but effectively unused; every struct above carries its identifier as a bare `string` minted inline as `uuid-<uuid>`, not via this interface.
- **Event** (`sip.Event`) — an empty stub. PREMIS events are not modeled yet.

**All identifiers take the form `uuid-<uuid>`** and are minted locally with `github.com/google/uuid`. SIP Creator asserts no authority over these identifiers — they are meaningful only within the package (see [ADR-0001](decisions/0001-package-builder-not-archive.md)).

## Package layout

The `basic` profile produces this on-disk tree under `Location` (`dest/uuid-<uuid>/`):

```
uuid-<uuid>/
  METS.xml                              package-level structural map
  metadata/
    descriptive/
      dc+schema.xml                     descriptive metadata (Dublin Core + schema.org)
    preservation/
      premis.xml                        package-level PREMIS (the intellectual entity)
  representations/
    representation_1/
      METS.xml                          representation-level structural map
      data/
        <essence files>                 the copied content
      metadata/
        preservation/
          premis.xml                    representation-level PREMIS (representation + its files)
  schemas/
    *.xsd                               all bundled XSDs, copied verbatim
```

The whole tree is then zipped **uncompressed** (`zip.Store`) to `dest/uuid-<uuid>.zip` — the deliverable meemoo ingests.

### The three metadata standards

Each metadata file corresponds to one preservation standard, generated by its own encoder package:

- **Descriptive** — `metadata/descriptive/dc+schema.xml`, Dublin Core terms + schema.org in the meemoo `1.2/basic` namespace (basic profile), or a simple-DC document (eark profile). Encoder: `encoders/metadata`, one document define per shape.
- **Preservation** — `premis.xml` at both package and representation level ([PREMIS](https://www.loc.gov/standards/premis/) 3.0): the objectIdentifiers, fixity, format registry, and the structural relationships (`represents`, `is represented by`, `includes`, `is included in`) between entity, representation, and files. Encoder: `encoders/premis`.
- **Structural** — `METS.xml` at both package and representation level ([METS](https://www.loc.gov/standards/mets/)): the manifest tying descriptive and preservation metadata, the file inventory (with checksums/sizes), and the structMap together. The package METS points at each representation METS via `mptr`. Encoder: `encoders/mets`.

## Build lifecycle

Entry point: `main.go` → `cli/cli.go` → `cli/create_cmd.go`. The command is:

```
sip-creator create [src] [dest] --profile basic
```

`create_cmd.go` resolves `--profile` against the profile registry (`profiles.Get`); an unknown or empty value returns an error listing the available profiles (`profiles.Names`). It then builds a format identificator from config (`formats.New`), constructs a `profiles.Builder`, and calls `Build(def)`. On success it hands the assembled `*sip.Package` to `archive.Zip`.

A profile is **data, not code**: a `profiles.Definition` (descriptive source filename, local-identifier scheme, PREMIS emission flags, and the `sip.Spec` METS values — profile URL, content typing, agents) resolved from the in-package registry. One engine reads it: `Builder.Build(def)` runs the build in **two phases with a hard seam between them** — assemble the complete package graph in memory, then emit it to disk. Errors are returned, not panicked, and assembly failures happen before anything exists on disk — a bad input leaves no partial package directory behind. Adding a meemoo content profile (material-artwork, newspaper, …) is one registry entry, not a new build path.

**Phase 1 — assemble** (`profiles/assemble.go`, zero writes):

1. Mint the package (`sip.NewPackage`) and the root `Entity`.
2. Decode `src/dc+schema.json`, swap the entity's UUID in as `dcterms:identifier`, lift the source `dcterms:identifier` out to the entity as `MEEMOO-LOCAL-ID`, and park the decoded description on the entity (`Entity.Description`) for the writer to serialize. The descriptive `File` node is declared with its path; decoding stays behind a single call (`decodeDescriptive`) so future input formats plug in there.
3. Declare one schema `File` node per bundled XSD, in sorted order (deterministic METS emission).
4. Walk `src` for directories matching `representation_([0-9]+)$`; for each essence file, record `File.Source` plus the representation-relative `Path`, and — when an identificator is configured — run format identification **on the source file** ([ADR-0006](decisions/0006-format-identification-optional.md)): unconfigured is skipped, a broken tool aborts assembly, a no-match leaves that file's `Format` nil.

**Phase 2 — write** (`profiles/write.go`, backed by the `store/` package), in a dependency order that is load-bearing and encoded exactly once, top to bottom in `write()`:

5. Skeleton directories.
6. Schema files (package METS references their fixity).
7. Per representation: directories, then essence copies — fixity is computed during the streamed copy and back-filled onto the graph nodes, so it describes the bytes actually in the package.
8. Descriptive XML (serialized from `Entity.Description`).
9. Per representation: `premis.xml`, then `METS.xml` (the representation METS embeds its PREMIS file's fixity). These `File` nodes are born in the writer, not the assembler — their existence is toggled by the definition's `EmitRepresentationPremis`/`EmitPackagePremis` flags, and the METS templates render their references conditionally so a premis-less profile stays valid.
10. Package `premis.xml` (the intellectual entity).
11. Package `METS.xml` — **strictly last**, because it references every representation METS, the package PREMIS, the descriptive file, and every schema file by checksum.

The assembled package is returned to the CLI, which zips it.

### Ordering dependencies

The dependency order is still not enforced by types, but it now lives in exactly one place — the numbered steps of `write()` — instead of being re-hand-sequenced per profile:

- Format identification (when configured) runs at assembly, **before** any PREMIS is rendered (PREMIS records the format registry).
- Representation PREMIS is written **before** its representation METS (which embeds its fixity).
- Everything is written **before** the package METS; it is always **last**.

## Code organization

Layered by responsibility. The `sip/` graph is the shared domain; everything else reads or writes it.

- **`sip/`** — the domain model (above). Plain structs and constructors, no IO.
- **`profiles/`** — profile-driven package building, split along the assemble/emit seam. `definition.go` holds `Definition` (a profile as data) and the registry (`Get`/`Names`); `builder.go` holds the `Builder` engine and `Build(def)`; `assemble.go` is the pure input-to-graph phase; `write.go` is the canonical emission order. Covered by `assemble_test.go` (graph shape, walk edge cases, the zero-writes guarantee) and `definition_test.go` (registry, definition-driven behavior).
- **`store/`** — dumb filesystem primitives rooted at the package directory (`MkdirAll`, `CopyFile`, `WriteMetadata`). Callers deal only in package-relative paths; writes truncate (safe re-runs); `CopyFile` computes MD5/size in the same streamed pass as the copy; `WriteMetadata` renders to memory first, so a failed template leaves no partial file.
- **`encoders/`** — one package per metadata standard (`metadata`, `mets`, `premis`), each a thin `Encode*(io.Writer, …)` API backed by a `text/template`. **No XML library** — all XML is generated from templates (see [ADR-0002](decisions/0002-xml-via-text-template.md)). METS ID minting lives in the mets encoder's `identifier()`/`idStore`.
- **`formats/`** — pluggable, optional format identification. `Identificator` is the interface (`Identify(path) (*sip.Format, error)`; nil identificator means skip); `Register`/`New` is a self-registration registry keyed by name. `formats/siegfried` self-registers on import and shells out to the external `sf` binary, returning errors (not panics) on exec or parse failure.
- **`schemas/`** — all XSDs bundled via `//go:embed`; `Get()` returns them as `map[name][]byte` for copying into each SIP.
- **`archive/`** — `Zip` walks the package directory and writes an uncompressed zip.
- **`cli/`** — the operator frontend: cobra commands, env config (`.env` via godotenv, parsed with caarlos0/env; `CONFIG.md` is generated from the config struct, `go generate ./cli`), and logger construction. All of it unexported: configuration is the CLI's operator contract, not library API.

Dependencies are kept small and boring: cobra (CLI), google/uuid, godotenv + caarlos0/env (config), samber/lo (used only by the mets encoder). Format identification depends on an **external `sf` binary**, not a Go library.

## CLI/library boundary

SIP Creator is a library with a CLI frontend, and the boundary between the two is a design principle the system is converging on. The library is designed to be embeddable in larger systems that automate ingest workflows — systems that hold content as streams (e.g. in object storage) and metadata as structured data, not as operator-prepared directories. The library API is therefore the contract; input formats are frontends that map onto it (the same layering commons-ip uses: a programmatic builder at the core, tools on top).

- **The library owns the domain, not the input.** Its API is the domain model (`sip/`) plus primitives to assemble and emit a package. It never sees a CSV and never assumes a source directory layout.
- **The CLI owns the operator contract.** It reads the input convention (see [input-spec.md](input-spec.md), draft), merges configuration (agents, profile), enforces the input rules, and translates everything into library API calls. All operator-facing error reporting lives here, in plain language.
- **Validation splits in two.** Input-contract errors (bad CSV key, misplaced file, missing folder) are CLI concerns. Domain invariants (at least one representation; a replacement needs the original package identifier; an entity needs a local identifier) are `Validate` methods on the `sip/` domain types — any embedding system must hit the same guardrails without going through the input convention.
- **The library is stream-first.** The file-adding primitive takes an `io.Reader` plus a logical path and optional pre-computed fixity, not a source directory. The CLI feeds it opened disk files; an embedding system feeds it streams from wherever it stores content.
- **Format identification is pluggable and optional** ([ADR-0006](decisions/0006-format-identification-optional.md)): Siegfried wants a real file path, so it cannot be assumed in a stream-first API. Callers may supply format information as data, the same pattern as fixity.

**Known gap:** the assemble/emit split separates input discovery (`assemble.go`) from output generation (`write.go`), and administrative metadata (agents, profile values) is now data (`sip.Spec`/`sip.Agent`, set from the profile definition) — but the build still assumes an operator-prepared source directory rather than streams, and fixity/identifiers cannot yet be supplied by a caller. The remaining work is the embeddability requirements in [TODO.md](TODO.md).

## Input contract

The `basic` profile expects a source directory containing:

- Exactly one **`dc+schema.json`** — descriptive metadata as JSON-LD, using the `dcterms:` (`http://purl.org/dc/terms/`) and `schema:` (`http://schema.org/`) vocabularies. Decoded by `encoders/metadata.Decode`.
- One or more **`representation_N/`** directories, each holding at least one essence file. The directory name must match `representation_([0-9]+)$`.

`tmp/basic/` is the sample input; `basic-uuid/` is sample generated output. Both are local fixtures, not tracked in git.

A convention-based replacement for this contract — folders as the UI, one key–value `metadata.csv`, administrative data from configuration — is drafted in [input-spec.md](input-spec.md). It is not implemented; the contract above is what the code enforces today.

## Validation

The acceptance check is **external**: generated packages are validated with commons-ip, the E-ARK CSIP reference validator. The CSIP rules are *not* reimplemented as Go tests — commons-ip is the reference (see [ADR-0003](decisions/0003-validation-stays-external.md)). Go tests cover what the validator can't see: the `store/` primitives (fixity, truncate-on-rewrite, fail-fast) and the assembler (graph shape, meemoo-layer semantics like `MEEMOO-LOCAL-ID`, the zero-writes guarantee).

The workflow around the tool ([ADR-0005](decisions/0005-dockerized-validation-and-html-reporting.md)): `./build.sh [profile]` is the local CI loop — build, regenerate the sample SIP, validate the zip with a dockerized commons-ip (release jar pinned by version + sha256), and exit non-zero iff the package is not `VALID` (the commons-ip CLI itself always exits 0; the verdict is read from its JSON report). The spec version is pinned per profile family's era — `basic`/meemoo-1.2 against 2.0.4 (whose profile URL meemoo 1.2 mandates), `eark` against 2.2.0 — so a commons-ip default change can't silently move the goalposts. Each run's reports are published to `reports/runs/<timestamp>/`; `docker compose up -d reports` serves a static HTML view of all runs at http://localhost:8080. `scripts/validate.sh` is usable standalone against any package zip or directory.

## Known gaps

These are true of the code today and tracked in [TODO.md](TODO.md):

- **One error path is still a panic.** The build path (`profiles/`, `store/`, `formats/`, the metadata decoder) returns errors, but `archive.Zip` still `panic`s on failure.
- **`mets/@TYPE` is a fixed registry value** (`Photographs – Digital` — legal meemoo-1.2 vocabulary, and apt for the sample fixture) where it should ultimately be operator- or content-selectable per package. Data in `profiles/definition.go`; the remaining work is input/config plumbing, not a fix.
- **The `sip.Identifier` interface and `sip.Event` stub are unused.** Identifiers are bare strings; PREMIS events are not modeled.
- **The `representation_([0-9]+)$` regex is stricter than either spec requires** and silently skips non-matching directories — a representation named `master` is dropped with no error.

The [refactoring plan](plans/refactoring-plan.md) is mid-flight: Phases 0–2 (baseline gate; assemble/emit split, `store/` package, panic-to-error conversion; declarative `Definition` registry, spec-driven METS templates) are done and verified output-equivalent. Phase 3 (a true E-ARK writer) and the companion [format-identification plan](plans/format-identification-optional.md) remain.
