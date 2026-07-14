# SIP Creator — design

SIP Creator is a Go library and CLI that assembles a producer's essence files and descriptive metadata into a standards-conformant Submission Information Package (SIP). It walks an input directory, copies the essence into the package layout, runs format identification over each file, and generates the descriptive, preservation, and structural XML that describe and bind the whole. The current output target is a [meemoo SIP v2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/) `basic` package layered on the [E-ARK CSIP](https://earkcsip.dilcis.eu/) profile.

This document describes **what the system is today** — the domain model, the package layout it produces, and the build lifecycle. It reflects the code as it stands, including its known gaps and rough edges; where the code is mid-evolution, that is called out rather than smoothed over. The *why* behind key choices lives in the [decision records](decisions/); planned changes live in [plans/](plans/); the backlog lives in [TODO.md](TODO.md). See [README.md](README.md) for how the docs are organized, and the repo-root `CLAUDE.md` for coding conventions.

> **Status: experimental, work in progress.** Only the `basic` profile is wired into the CLI. The sample package this tool generates is known to be **INVALID** against commons-ip today (see [Known gaps](#known-gaps)); making it valid is active work.

## Domain model

The domain lives in the `sip/` package as a graph of plain structs, built entirely in memory before (and during) writing to disk. It mirrors the OAIS/METS vocabulary: an intellectual entity, its representations, and the files within them.

- **Package** (`sip.Package`) — the unit of work: one SIP. Carries its `Identifier` and `Location` (the on-disk package directory), a single root `Entity`, and the generated top-level files (package `MetsFile`, package `PremisFile`, `SchemaFiles`). Created by `NewPackage(baseDir)`, which mints the identifier and sets `Location = baseDir/uuid-<uuid>`.
- **Entity** (`sip.Entity`) — the intellectual entity being preserved (the "what this is"). Holds its `Identifier`, a map of `AdditionalIdentifiers` (e.g. `MEEMOO-LOCAL-ID`), its `Representations`, its `DescriptionFile` (the descriptive-metadata file), and a slice of sub-`Entities`. The sub-entity slice exists in the model but the `basic` profile only ever builds a single root entity with no children.
- **Representation** (`sip.Representation`) — a typed grouping of the essence (e.g. a master vs. a derivative). Holds its `Label` (the source directory name, e.g. `representation_1`), `Identifier`, its `Files`, its own representation-level `MetsFile` and `PremisFile`, and a back-reference to its `Entity`.
- **File** (`sip.File`) — one file in the package, whether essence or generated metadata. Records `Identifier`, `Name`, `Checksum` (MD5), `Size`, `Created` (RFC3339Nano), `Path`, and — for essence — a `Format`. `Path` is the href **relative to the METS document that references the file**, which is why it is computed differently for essence (representation-relative) than for package-level metadata (package-relative).
- **Format** / **FormatRegistry** (`sip.Format`, `sip.FormatRegistry`) — the format-identification result attached to an essence file: the PRONOM registry key and name from Siegfried, plus a `Role` of `specification`.
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

- **Descriptive** — `metadata/descriptive/dc+schema.xml`, Dublin Core terms + schema.org, in the meemoo `2.0/basic` namespace. Encoder: `encoders/metadata`.
- **Preservation** — `premis.xml` at both package and representation level ([PREMIS](https://www.loc.gov/standards/premis/) 3.0): the objectIdentifiers, fixity, format registry, and the structural relationships (`represents`, `is represented by`, `includes`, `is included in`) between entity, representation, and files. Encoder: `encoders/premis`.
- **Structural** — `METS.xml` at both package and representation level ([METS](https://www.loc.gov/standards/mets/)): the manifest tying descriptive and preservation metadata, the file inventory (with checksums/sizes), and the structMap together. The package METS points at each representation METS via `mptr`. Encoder: `encoders/mets`.

## Build lifecycle

Entry point: `main.go` → `cli/cli.go` → `cli/create_cmd.go`. The command is:

```
sip-creator create [src] [dest] --profile basic
```

`create_cmd.go` builds a format identificator from config (`formats.New`), constructs a `profiles.Profile`, and switches on `--profile`. Only `"basic"` is handled; any other value (including empty) returns an error. On success it hands the assembled `*sip.Package` to `archive.Zip`.

`Profile.Basic()` (`profiles/basic.go`) runs the build in two conceptual steps. **Model-building and disk IO are interleaved** — a metadata `File` node is only born after its bytes have been written and stat-ed, because the METS manifest needs each file's real checksum and size:

**Step 1 — assemble the package and copy input:**

1. `createPackage()` — mint the package, create the skeleton directories, and write the `schemas/` XSDs (each schema copy produces a `File` node with its checksum/size).
2. Create the root `Entity`.
3. `createDescriptiveFile()` — decode `src/dc+schema.json`, swap the entity's UUID in as `dcterms:identifier`, lift the source `dcterms:identifier` out to the entity as `MEEMOO-LOCAL-ID`, then encode `dc+schema.xml`.
4. `eachDirectory()` — walk `src` for directories matching `representation_([0-9]+)$`; for each, `eachEssenceFile()` copies every file into `representations/<label>/data/` and runs format identification on the copy.

**Step 2 — generate the derived metadata**, in a dependency order that is load-bearing (each METS references the checksum/size of files written before it):

5. Per representation: representation `premis.xml`, then representation `METS.xml`.
6. Package `premis.xml` (the intellectual entity).
7. Package `METS.xml` — **strictly last**, because it references every representation METS, the package PREMIS, and every schema file by checksum.

The assembled package is returned to the CLI, which zips it.

### Ordering dependencies

These are implicit in the call order of `Basic()` and are not enforced by types — reordering silently produces broken references:

- Format identification must run **before** any PREMIS (PREMIS records the format registry).
- Representation PREMIS + METS must be written **before** the package METS (it references their fixity).
- Package PREMIS must be written **before** the package METS.
- The package METS is always **last**.

## Code organization

Layered by responsibility. The `sip/` graph is the shared domain; everything else reads or writes it.

- **`sip/`** — the domain model (above). Plain structs and constructors, no IO.
- **`profiles/`** — the build engine. `profile.go` holds the `Profile` type and every filesystem/assembly helper (`createPackage`, `createDescriptiveFile`, `eachDirectory`, `eachEssenceFile`, `generate*`, plus the low-level `createMetadataFile`/`copy`/`createDir`). `basic.go` is the one wired profile: it orchestrates the helpers into the lifecycle above. `roda.go` holds a second, **unreachable** profile (see [Known gaps](#known-gaps)).
- **`encoders/`** — one package per metadata standard (`metadata`, `mets`, `premis`), each a thin `Encode*(io.Writer, …)` API backed by a `text/template`. **No XML library** — all XML is generated from templates (see [ADR-0002](decisions/0002-xml-via-text-template.md)). METS ID minting lives in the mets encoder's `identifier()`/`idStore`.
- **`formats/`** — pluggable format identification. `Identificator` is the interface (`Process(path) *sip.File`); `Register`/`New` is a self-registration registry keyed by name. `formats/siegfried` self-registers on import and shells out to the external `sf` binary.
- **`schemas/`** — all XSDs bundled via `//go:embed`; `Get()` returns them as `map[name][]byte` for copying into each SIP.
- **`archive/`** — `Zip` walks the package directory and writes an uncompressed zip.
- **`services/`** — config: `.env` via godotenv, parsed with caarlos0/env. `CONFIG.md` is generated from the config struct (`go generate ./services`).

Dependencies are kept small and boring: cobra (CLI), google/uuid, godotenv + caarlos0/env (config), samber/lo (used only by the mets encoder). Format identification depends on an **external `sf` binary**, not a Go library.

## Input contract

The `basic` profile expects a source directory containing:

- Exactly one **`dc+schema.json`** — descriptive metadata as JSON-LD, using the `dcterms:` (`http://purl.org/dc/terms/`) and `schema:` (`http://schema.org/`) vocabularies. Decoded by `encoders/metadata.Decode`.
- One or more **`representation_N/`** directories, each holding at least one essence file. The directory name must match `representation_([0-9]+)$`.

`tmp/basic/` is the sample input; `basic-uuid/` is sample generated output. Both are local fixtures, not tracked in git.

## Validation

The acceptance check is **external**: generated packages are validated with commons-ip, the E-ARK CSIP reference validator. The CSIP rules are *not* reimplemented as Go tests — commons-ip is the reference (see [ADR-0003](decisions/0003-validation-stays-external.md)). There are currently no `*_test.go` files.

The workflow around the tool ([ADR-0005](decisions/0005-dockerized-validation-and-html-reporting.md)): `./build.sh` is the local CI loop — build, regenerate the sample SIP, validate the zip with a dockerized commons-ip (release jar pinned by version + sha256, spec version pinned to 2.2.0), and exit non-zero iff the package is not `VALID` (the commons-ip CLI itself always exits 0; the verdict is read from its JSON report). Each run's reports are published to `reports/runs/<timestamp>/`; `docker compose up -d reports` serves a static HTML view of all runs at http://localhost:8080. `scripts/validate.sh` is usable standalone against any package zip or directory.

## Known gaps

These are true of the code today and tracked in [TODO.md](TODO.md):

- **The sample package is INVALID against commons-ip.** Two known failures: the descriptive `<metadata>` element has no resolvable schema declaration, and `dcterms:created`'s `edtf:EDTF-level1` type does not resolve. RODA additionally reports a mislocated representation PREMIS path.
- **Errors are panics.** The profile helpers, the metadata decoder, and `archive.Zip` `panic` on IO/parse failure rather than returning errors. A partial package directory can be left behind on failure.
- **`createMetadataFile` and `copy` open files with `O_APPEND`**, so re-running into an existing package directory *concatenates* onto existing files rather than truncating.
- **`roda.go` is dead code.** It is ~90% identical to `Basic()`, is not reachable from the CLI switch, and omits representation PREMIS (the source of the RODA "Preservation metadata file not found" error). The agreed direction is a genuine E-ARK writer, not this copy — see [ADR-0004](decisions/0004-eark-base-meemoo-specialization.md) and the [refactoring plan](plans/refactoring-plan.md).
- **Meemoo literals are baked into shared templates** (`TYPE="Photographs – Digital"`, the `2.0/basic` profile URL, the UGent agent block), which blocks a clean CSIP-base / meemoo-specialization split.
- **The `sip.Identifier` interface and `sip.Event` stub are unused.** Identifiers are bare strings; PREMIS events are not modeled.
- **The `representation_([0-9]+)$` regex is stricter than either spec requires** and silently skips non-matching directories — a representation named `master` is dropped with no error.

The [refactoring plan](plans/refactoring-plan.md) addresses the assemble/emit split, the declarative profile registry, the panic-to-error conversion, and the `O_APPEND` and `roda.go` issues; it is accepted but not yet started.
