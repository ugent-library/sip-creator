# SIP Creator design

SIP Creator is a Go library and CLI that assembles a producer's essence files and descriptive metadata into a standards-conformant Submission Information Package (SIP). It walks an input directory, copies the essence into the package layout (computing fixity natively during the copy), optionally enriches files with format info from a pre-computed characterization report (a `siegfried.json` sidecar), and generates the descriptive, preservation, and structural XML that describe the content and tie the package together. The output targets are a [meemoo SIP v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) `basic` package (the stable meemoo spec; 2.0/2.1 are release candidates) layered on the [E-ARK](https://earksip.dilcis.eu/) profile, and a plain E-ARK SIP (`eark` profile) for RODA-class repositories. The BagIt envelope meemoo's transfer requires is deliberately out of scope ([ADR-0008](decisions/0008-bag-layer-out-of-scope.md)): bag the package directory with a reference BagIt implementation.

This document describes **what the system is today**: the domain model, the package layout it produces, and the build lifecycle. It reflects the code as it stands, including its known gaps and rough edges; where the code is mid-evolution, that is called out rather than smoothed over. The *why* behind key choices lives in the [decision records](decisions/); planned changes live in [plans/](plans/); the backlog lives in [TODO.md](TODO.md). See [README.md](README.md) for how the docs are organized, and the repo-root `CLAUDE.md` for coding conventions.

> **Status: experimental, work in progress.** Two profiles are registered: `basic` (meemoo SIP 1.2) and `eark` (plain E-ARK SIP). Both sample packages validate **VALID** against commons-ip, each against the E-ARK spec version of its era (2.0.4 and 2.2.0 respectively).

## Domain model

The domain lives in the `sip/` package as a graph of plain structs, built entirely in memory before anything is written to disk. It mirrors the OAIS/METS vocabulary: an intellectual entity, its representations, and the files within them.

- **Package** (`sip.Package`): the unit of work, one SIP. Carries its `Identifier` and `Location` (the on-disk package directory), a single root `Entity`, and the generated top-level files (package `MetsFile`, package `PremisFile`, `SchemaFiles`). Created by `NewPackage(baseDir)`, which mints the identifier and sets `Location = baseDir/uuid-<uuid>`.
- **Entity** (`sip.Entity`): the intellectual entity being preserved (the "what this is"). Holds its `Identifier`, a map of `AdditionalIdentifiers` (e.g. `MEEMOO-LOCAL-ID`), its `Representations`, its `Description` (the decoded descriptive metadata, a `metadata.Terms`, awaiting serialization by the writer), its `DescriptionFile` (the descriptive-metadata file node), and a slice of sub-`Entities`. The sub-entity slice exists in the model but the `basic` profile only ever builds a single root entity with no children.
- **Representation** (`sip.Representation`): a typed grouping of the essence (e.g. a master vs. a derivative). Holds its `Label` (the source directory name, e.g. `representation_1`), `Identifier`, its `Files`, its own representation-level `MetsFile` and `PremisFile`, and a back-reference to its `Entity`.
- **File** (`sip.File`): one file in the package, whether essence or generated metadata. Records `Identifier`, `Name`, `Checksum` (MD5), `Size`, `Created` (RFC3339Nano), `Path`, `Source` (the absolute input path an essence file is copied from; empty for generated metadata), `Mime`, and, for essence, a `Format`. `Path` is the href **relative to the METS document that references the file**: essence and representation PREMIS are representation-relative, schema/descriptive files and representation METS entries are package-relative. `Mime` is what METS `@MIMETYPE` declares (a CSIP MUST) and is never empty by write time and never a guess: the characterization report's assertion for essence/documentation, a type true by construction for generated XML (`text/xml`) and bundled XSDs (`application/xml`), or `application/octet-stream` as the admitted unknown. The assembler declares `Path` and `Mime` up front; the writer back-fills `Checksum`/`Size`/`Created` as each file lands on disk.
- **Format** / **FormatRegistry** (`sip.Format`, `sip.FormatRegistry`): the characterization result attached to an essence file: the PRONOM registry key and name from the Siegfried report, plus a `Role` of `specification`. Nil when no report is supplied or the report recorded no match ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)); the PREMIS template omits `premis:format` accordingly.
- **Identifier validation** (`sip.ValidateIdentifier`): there is no identifier type; every struct above carries its identifier as a plain `string` in the `uuid-<uuid>` form, and `ValidateIdentifier` checks that form (the prefix keeps a UUID valid as an xsd:ID, which may not start with a digit).
- **Event** (`sip.Event`): an empty stub. PREMIS events are not modeled yet.

**All identifiers take the form `uuid-<uuid>`** and are minted locally with the standard library `uuid` package. SIP Creator asserts no authority over these identifiers; they are meaningful only within the package (see [ADR-0001](decisions/0001-package-builder-not-archive.md)).

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
  documentation/
    ...                                 optional: copied from the input's documentation/ dir
```

The whole tree is then zipped **uncompressed** (`zip.Store`) to `dest/uuid-<uuid>.zip`, the deliverable meemoo ingests.

### The three metadata standards

Each metadata file corresponds to one preservation standard, generated by its own encoder package:

- **Descriptive**: `metadata/descriptive/dc+schema.xml`, Dublin Core terms + schema.org in the meemoo `1.2/basic` namespace (basic profile), or a simple-DC document (eark profile). Encoder: `encoders/metadata`, one document define per shape. The metadata model is **one closed vocabulary table** ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md), `vocabulary.go`): a row per key carrying the CSV key, the emitted element, meemoo's required and cardinality columns (no / yes / per-language), the `xsi:type`, and the Simple DC parent. Decoding (`ResolveKey`), validation (`terms_validation.go`: term validity, cardinality, requiredness, required language; the last three are enforced per profile family via `Definition` data), and both templates (EDTF typing; the eark dumb-down) all read the table; nothing else in the code knows an element name. A new element is a table row plus an input-spec §3 line.
- **Preservation**: `premis.xml` at both package and representation level ([PREMIS](https://www.loc.gov/standards/premis/) 3.0): the objectIdentifiers, fixity, format registry, and the structural relationships (`represents`, `is represented by`, `includes`, `is included in`) between entity, representation, and files. Encoder: `encoders/premis`.
- **Structural**: `METS.xml` at both package and representation level ([METS](https://www.loc.gov/standards/mets/)): the manifest tying descriptive and preservation metadata, the file inventory (with checksums/sizes), and the structMap together. The package METS points at each representation METS via `mptr`. Encoder: `encoders/mets`.

## Build lifecycle

Entry point: `main.go` → `cli/cli.go` → `cli/create_cmd.go`. The command is:

```
sip-creator create [src] [dest] --profile basic
```

`create_cmd.go` resolves `--profile` against the profile registry (`profiles.Get`); an unknown or empty value returns an error listing the available profiles (`profiles.Names`). It resolves the run flags (`--status`/`--updates` per SIP3, `--content-category` over `SIP_CONTENT_CATEGORY` over the registry value), reads and validates the input folder via `input.Read` (all violations reported at once), maps it onto a `profiles.Input` (`Package.BuilderInput()`), and calls `Build(def, in)`. On success it hands the assembled `*sip.Package` to `archive.Zip`. The sibling `check` command runs `input.Read` alone: structure and metadata validation with no configuration and no build.

A profile is **data, not code**: a `profiles.Definition` resolved from the in-package registry, carrying the output family, the emitted descriptive filename, emission flags, the descriptive conformance rules (required elements, required language, cardinality enforcement), and the `sip.MetsDeclaration` METS values: profile URL, content typing, descriptive typing, record status, agents. One agent is deliberately *not* registry data: the submitting ORGANIZATION is operator identity, supplied per deployment. `Definition.WithSubmitter(name, orID)` returns a completed copy, and meemoo-family profiles refuse to build without the OR-id (the CLI reads both from `SIP_SUBMITTER_*`; embedding systems pass them as arguments). Each definition declares its **output family** (`FamilyMeemoo`, `FamilyEARK`): families share the one canonical writer and select *encodings*, today exactly one: the descriptive encoder (dc+schema vs simple DC). See [ADR-0007](decisions/0007-profile-families-share-one-writer.md), including the recorded fork triggers. One engine reads it: `Builder.Build(def, in)` takes the profile definition plus a per-build `profiles.Input`, the caller's source material as data (descriptive terms, representations as labeled source-file lists, documentation, received PREMIS, an optional characterization report and package identifier). The build runs in **two strictly separated phases**: validate the input and assemble the complete package graph in memory, then emit it to disk. Errors are returned, not panicked, and failures before the write phase leave no partial package directory behind. Adding a meemoo content profile (material-artwork, newspaper, …) is one registry entry, not a new build path.

**Phase 1: assemble** (`profiles/assemble.go`, zero writes; `Input.Validate` plus the profile's descriptive conformance rules run first):

1. Mint the package (`sip.NewPackage`; a caller-supplied identifier is reused, which is how an update keeps the original `mets/@OBJID`) and the root `Entity`.
2. Take the descriptive terms from the input: lift the producer's `dcterms:identifier` onto the entity as `MEEMOO-LOCAL-ID` (per `EmitLocalIdentifier`), swap the entity's UUID in as `dcterms:identifier` (per `SwapObjectIdentifier`; the eark profile skips the swap and keeps the producer's identifier, [ADR-0012](decisions/0012-eark-keeps-producer-identifier.md)), and park the terms on the entity (`Entity.Description`) for the writer to serialize. The descriptive `File` node is declared with the profile's `DescriptiveName`.
3. Declare one schema `File` node per bundled XSD, in sorted order (deterministic METS emission).
4. Turn each supplied representation into a graph node (package-side names are `representation_N` in supplied order; the producer's label rides along as the METS `@LABEL`) and, when a characterization report is present, enrich each essence file from the report's record for the **source file** ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)): no report is skipped; a present report is strict (missing essence entry, per-entry sf error, missing checksum, or an MD5 mismatch against the source bytes aborts assembly); a recorded no-match leaves that file's `Format` nil. Documentation files (package and representation level) need no entry but are checksum-verified when one exists. Received PREMIS files are checked to be well-formed `premis:premis` documents and declared as nodes, never parsed.
5. Declare the generated-metadata `File` nodes: per representation a `premis.xml` (only when the definition's `EmitRepresentationPremis` says so) and its `METS.xml`, plus the package `premis.xml` (per `EmitPackagePremis`) and the package `METS.xml`. Assembly leaves the graph complete: the writer creates no nodes, it only emits what the graph declares and back-fills fixity.

**Phase 2: write** (`profiles/write.go`, backed by the `store/` package), in a dependency order encoded exactly once, top to bottom in `write()`. The order is a hard constraint, not style: later files embed the fixity of earlier ones, so reordering emits METS documents whose checksum references are wrong (see "Ordering dependencies" below):

6. Skeleton directories.
7. Schema files and package documentation copies (package METS references their fixity).
8. Per representation: directories, then essence copies. Fixity is computed during the streamed copy and back-filled onto the graph nodes, so it describes the bytes actually in the package.
9. Package descriptive XML (serialized from `Entity.Description`).
10. Per representation: its descriptive XML (when the representation carries terms), documentation copies, received PREMIS copies, the generated `premis.xml`, then `METS.xml`; the representation METS embeds the fixity of all of them. Whether a generated PREMIS node exists was decided at assembly by the emission flags, and the METS templates render their references conditionally, so a profile without PREMIS stays valid.
11. Package `premis.xml` (the intellectual entity) and the package-level received PREMIS copies.
12. Package `METS.xml`, **strictly last**, because it references every representation METS, all preservation and descriptive documents, and every schema and documentation file by checksum.

The assembled package is returned to the CLI, which zips it.

### Ordering dependencies

The dependency order is still not enforced by types, but it now lives in exactly one place (the numbered steps of `write()`) instead of being re-hand-sequenced per profile:

- Characterization enrichment (when a report is present) happens at assembly, **before** any PREMIS is rendered (PREMIS records the format registry).
- Representation PREMIS is written **before** its representation METS (which embeds its fixity).
- Everything is written **before** the package METS; it is always **last**.

## Code organization

Layered by responsibility. The `sip/` graph is the shared domain; everything else reads or writes it.

- **`sip/`**: the domain model (above). Plain structs and constructors, no IO.
- **`profiles/`**: profile-driven package building, split into the assemble and emit phases. `definition.go` holds `Definition` (a profile as data) and the registry (`Get`/`Names`); `builder.go` holds the `Builder` engine and `Build(def)`; `assemble.go` is the pure input-to-graph phase; `write.go` is the canonical emission order. Covered by `assemble_test.go` (graph shape, walk edge cases, the guarantee that assembly writes nothing to disk) and `definition_test.go` (registry, definition-driven behavior).
- **`store/`**: dumb filesystem primitives rooted at the package directory (`MkdirAll`, `CopyFile`, `WriteMetadata`). Callers deal only in package-relative paths; writes truncate (safe re-runs); `CopyFile` computes MD5/size in the same streamed pass as the copy; `WriteMetadata` renders to memory first, so a failed template leaves no partial file.
- **`encoders/`**: one package per metadata standard (`metadata`, `mets`, `premis`), each a thin `Encode*(io.Writer, …)` API backed by a `text/template`. **No XML library**: all XML is generated from templates (see [ADR-0002](decisions/0002-xml-via-text-template.md)). METS elements that describe a graph node (`file`, `dmdSec`, `digiprovMD`, the per-representation `fileGrp`) carry that node's identifier, so METS and PREMIS agree on file identity; element IDs with no graph counterpart (`fileSec`, `structMap` divs, the data/documentation/schema `fileGrp`s) are minted per render by the mets encoder's `identifier()`. The `metadata` package carries the descriptive vocabulary table and the `Terms` model with its validation ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)).
- **`characterization/`**: decodes pre-computed characterization reports into per-file records (`DecodeSiegfried(io.Reader) (Report, error)`; `Report` maps input-relative slash paths to `Record`s carrying format, mime, MD5, and per-file tool errors). Decoding carries facts without judging them; strictness policy lives in the assembler, which knows which entries it needs ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). A future FITS/DROID sidecar is one new decode function.
- **`schemas/`**: all XSDs bundled via `//go:embed`; `Get()` returns them as `map[name][]byte` for copying into each SIP.
- **`archive/`**: `Zip` walks the package directory and writes an uncompressed zip.
- **`cli/`**: the operator frontend. Cobra commands, env config (`.env` via godotenv, parsed with caarlos0/env; `CONFIG.md` is generated from the config struct, `go generate ./cli`), and logger construction. All of it unexported: configuration is the CLI's operator contract, not library API.

Dependencies are kept small and boring: cobra (CLI), godotenv + caarlos0/env (config), golang.org/x/text (Unicode path normalization in the input walker); UUIDs come from the Go standard library. No characterization tool is executed and none lives in `go.mod`; format info arrives as a pre-computed report ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)).

## CLI/library boundary

SIP Creator is a library with a CLI frontend, and the boundary between the two is a design principle the system is converging on. The library is designed to be embeddable in larger systems that automate ingest workflows: systems that hold content as streams (e.g. in object storage) and metadata as structured data, not as operator-prepared directories. The library API is therefore the contract; input formats are frontends that map onto it (the same layering commons-ip uses: a programmatic builder at the core, tools on top).

- **The library owns the domain, not the input.** Its API is the domain model (`sip/`) plus primitives to assemble and emit a package. It never sees a CSV and never assumes a source directory layout.
- **The CLI owns the operator contract.** It reads the input convention (see [input-spec.md](input-spec.md)), merges configuration (agents, profile), enforces the input rules, and translates everything into library API calls. All operator-facing error reporting lives here, in plain language.
- **Validation splits in two.** Input-contract errors (bad CSV key, misplaced file, missing folder) are CLI concerns, collected as violations phrased for the operator. Domain invariants are fail-fast `Validate` methods next to the data they guard: `Input.Validate` (graph rules) and `metadata.Terms` (term validity, cardinality). Any embedding system therefore hits the same guardrails without going through the input convention; profile conformance (required elements, required language, whether cardinality is enforced) is `Definition` data checked in `Build`.
- **The library is converging on streaming input.** The target file-adding primitive takes an `io.Reader` plus a logical path and optional pre-computed fixity, not a source directory: the CLI would feed it opened disk files, an embedding system streams from wherever it stores content. Today essence still arrives as filesystem paths (the known gap below).
- **Characterization is data, not an interface** ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)): library callers supply a `characterization.Report` directly (`profiles.Input.Characterization`); the CLI's `siegfried.json` sidecar file is one transport of the same data. The same strictness (MD5 binding) applies to both.

**Known gap:** the builder takes its input as data (`profiles.Input`: descriptive terms, representations as source paths, documentation, received PREMIS, a characterization report, an optional caller-supplied package identifier), so embedding systems construct its input without the folder convention. But essence still arrives as filesystem paths, not streams, and fixity cannot yet be supplied pre-computed. The remaining work (streams and pre-computed fixity) is tracked in [TODO.md](TODO.md).

## Input contract

The CLI's input contract is the convention-based folder defined in [input-spec.md](input-spec.md), the authoritative, operator-facing statement of every rule. In brief: one folder is one package, carrying one `metadata.csv` (descriptive metadata as key–value Dublin Core terms, vocabulary per [ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)), content either flat or under `representations/<label>/`, and optional `documentation/`, `premis/` (received preservation XML, passed through unparsed) and a `siegfried.json` characterization sidecar ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). Administrative values come from configuration, not the folder ([ADR-0010](decisions/0010-config-over-self-describing-input.md)).

`cli/input` enforces the contract, collecting every MUST violation into one report phrased for the operator; the `check` command runs that validation standalone. The same rules that hold for the *graph* rather than the folder (descriptive validity and requiredness, received-PREMIS conformance, characterization checksum verification) are enforced again in the builder for every producer: the folder is one transport onto `profiles.Input`, not the API.

`tmp/basic/` and `tmp/eark/` are sample inputs; `basic-uuid/` is sample generated output. All are local fixtures, not tracked in git.

## Validation

The acceptance check is **external**: generated packages are validated with commons-ip, the E-ARK CSIP reference validator. The CSIP rules are *not* reimplemented as Go tests; commons-ip is the reference (see [ADR-0003](decisions/0003-validation-stays-external.md)). Go tests cover what the validator can't see: the `store/` primitives (fixity, truncation on rewrite, fail-fast), the assembler (graph shape, meemoo-layer semantics like `MEEMOO-LOCAL-ID`, the guarantee that assembly writes nothing to disk), the input reader (`cli/input`: folder walk, CSV decoding, violation collection), and the descriptive terms model (`encoders/metadata`).

The workflow around the tool ([ADR-0005](decisions/0005-dockerized-validation-and-html-reporting.md)): `./build.sh [profile]` is the local CI loop: build, regenerate the sample SIP, validate the zip with a dockerized commons-ip (release jar pinned by version + sha256), and exit non-zero iff the package is not `VALID` (the commons-ip CLI itself always exits 0; the verdict is read from its JSON report). The spec version is pinned per profile family's era: `basic`/meemoo-1.2 against 2.0.4 (whose profile URL meemoo 1.2 requires), `eark` against 2.2.0. A commons-ip default change therefore can't silently move the goalposts. Each run's reports are published to `reports/runs/<timestamp>/`; `docker compose up -d reports` serves a static HTML view of all runs at http://localhost:8080. `scripts/validate.sh` is usable standalone against any package zip or directory.

## Known gaps

These are true of the code today and tracked in [TODO.md](TODO.md):

- **The `sip.Event` stub is unused.** PREMIS events are not modeled; the stub anchors the events design question in [TODO.md](TODO.md). (The once-unused `sip.Identifier` interface was deleted 2026-08-20; identifiers are plain `uuid-<uuid>` strings, validated by `sip.ValidateIdentifier`.)
- **Essence arrives as filesystem paths, not streams**, and fixity cannot be supplied pre-computed; this is the remaining embeddability work (see the CLI/library boundary above).

Resolved 2026-08-20 with the [input-convention plan](archive/input-convention.md): `mets/@TYPE` is operator-selectable (`--content-category`, `SIP_CONTENT_CATEGORY`), and the old `representation_N` input regex is gone: operator folders under `representations/` are free-form labels, and the profile names the package-side directories.

The [refactoring plan](archive/refactoring-plan.md) and its companions have shipped: the assemble/emit split, `store/` package, and declarative `Definition` registry (Phases 0–2), the decision to make enrichment optional ([format-identification plan](archive/format-identification-optional.md), since superseded in part), and the eark profile ([eark-writer plan](plans/eark-writer.md), awaiting field acceptance on RODA). The [characterization-sidecar plan](archive/characterization-sidecar.md) has landed in full: Step A (sidecar input) and Step B (METS `@MIMETYPE` emitted from resolved per-file mime; the long-standing hardcoded-`text/xml` lie is gone).
