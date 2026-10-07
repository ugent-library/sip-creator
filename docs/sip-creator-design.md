# SIP Creator design

SIP Creator is a Go library and CLI that builds a Submission Information Package (SIP) from a producer's essence files and descriptive metadata. It copies the essence into the package, computing fixity during the copy unless a pre-computed characterization report supplies it, optionally adds format info from that report, and generates the descriptive, preservation and structural XML. Every package is an [E-ARK SIP](https://earksip.dilcis.eu/). Three profiles decide the rest:

| Profile | Package | Descriptive metadata | E-ARK SIP version |
|---|---|---|---|
| `meemoo/basic` | [Meemoo SIP 1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) basic content profile (the stable version; 2.0 and 2.1 are release candidates) | Meemoo's Dublin Core terms and schema.org (`dc+schema.xml`) | 2.0.4 |
| `eark/dc` | plain E-ARK SIP | Simple Dublin Core (`dc.xml`) | 2.2.0 |
| `eark/mods` | plain E-ARK SIP | [MODS 3.7](https://www.loc.gov/standards/mods/) (`mods.xml`) | 2.2.0 |

The BagIt envelope Meemoo's transfer requires is out of scope ([ADR-0008](decisions/0008-bag-layer-out-of-scope.md)): bag the package directory with a reference BagIt implementation.

This document describes the system as it is today. The reasons for key choices are in the [decision records](decisions/), planned changes in [plans/](plans/), open questions in [TODO.md](TODO.md). [README.md](README.md) explains how the docs are organized; the repository's `CLAUDE.md` holds the coding conventions.

> **Status: experimental.** The sample package of each profile validates `VALID` with zero warnings against commons-ip, and every descriptive document in it validates with xmllint against the schema the package ships for it.

## Domain model

The `sip/` package holds the package as a graph of plain structs, built in memory before anything is written. It follows the OAIS and METS vocabulary: one intellectual entity, its representations, and their files. The fields are documented on the types.

- **Package** (`sip.Package`): one SIP, rooted at `dest/<identifier>/`. It holds the entity and the package-level files: the package METS and PREMIS, the schemas, documentation and received PREMIS. An update reuses the identifier of the package it updates, which keeps the original's `mets/@OBJID`.
- **Entity** (`sip.Entity`): the intellectual entity, with its description and its representations. A package has one entity; nested entities are not modeled (an open question in [TODO.md](TODO.md)).
- **Representation** (`sip.Representation`): one version of the content, such as a master or an access copy, with its files, an optional description of this version only, and its own METS and PREMIS. Its name is its directory under `representations/`, taken as the producer gave it; its label is its display name (`mets/@LABEL`). Its METS can type it differently from the package: when the definition sets `EmitRepresentationType`, the representation's type goes into the content typing, where ingest systems read it ([ADR-0013](decisions/0013-representation-type-from-label.md)).
- **File** (`sip.File`): one file in the package: essence, documentation, a schema or a metadata document. Its path is **relative to the METS document that references it**: essence and representation PREMIS relative to the representation; schemas, descriptive documents and representation METS files relative to the package. Its MIME type, which METS must declare, is never a guess: the characterization report's type, the known type of a generated document, or `application/octet-stream`. The assembler sets path and MIME type; the writer fills in checksum, size and creation time as the file is written.
- **Format** (`sip.Format`): the PRONOM format of an essence file, from the characterization report. Nil when there is no report or no match; PREMIS then leaves out `premis:format` ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)).
- **Record status** (`sip.RecordStatus`): `metsHdr/@RECORDSTATUS`, typed because SIP3's six values are a closed list. A status that updates an earlier package (`IsUpdate`) requires that package's identifier. The CSIP content category is an open list and stays a plain string.
- **Event** (`sip.Event`): an empty stub. PREMIS events are not modeled yet.

Identifiers are plain strings of the form `uuid-<uuid>`, minted with the standard library `uuid` package. The prefix keeps them valid as an `xsd:ID`, which may not start with a digit. SIP Creator claims no authority over them; they identify things within the package only ([ADR-0001](decisions/0001-package-builder-not-archive.md)).

## Package layout

The package directory, at `dest/<identifier>/`:

```
uuid-<uuid>/
  METS.xml                     package METS
  metadata/
    descriptive/
      dc+schema.xml            the descriptive document: dc.xml under eark/dc, mods.xml under eark/mods
    preservation/
      premis.xml               package PREMIS (basic only)
      ...                      received PREMIS from the input's premis/ folder
  representations/
    <name>/                    one directory per representation
      METS.xml                 representation METS
      data/
        ...                    the essence
      metadata/
        descriptive/
          dc.xml               optional, eark profiles only: a description of this version (mods.xml under eark/mods)
        preservation/
          premis.xml           representation PREMIS (basic only)
          ...                  received PREMIS
      documentation/
        ...                    optional
  schemas/
    *.xsd                      the XSDs the package's documents point at
  documentation/
    ...                        optional, from the input's documentation/ folder
```

A descriptive document the producer supplied as a file lands at the same path as a generated one, copied as it is.

The CLI zips the directory **uncompressed** (`zip.Store`) to `dest/<identifier>.zip`, unless `--no-zip` is given. Every file copied into the package keeps the modification time of its source, as `cp -p` does, because it is the one date the producer's file system holds and it cannot be recovered later; a generated document has the build time. Each zip entry carries the modification time of its file in the package directory, as both MS-DOS date fields and an Info-ZIP extended timestamp, so the producer's dates survive the zip. METS `CREATED` is a different date: when the file was written into the package, which is what E-ARK CSIP defines it as. The zip is written to a temporary file next to it and renamed when complete, so `<identifier>.zip` only ever names a whole zip; a zip that already exists is refused, never replaced ([ADR-0031](decisions/0031-final-names-hold-complete-output.md)). An update reuses the earlier package's identifier, so `create` checks for its zip before building; the refusal then leaves nothing behind. Under the eark profiles the zip is the deliverable; under `meemoo/basic` the deliverable is the package directory in a BagIt bag, which this tool does not produce.

## Metadata

Each kind of metadata has its own standard and its own generator: the profile writes the descriptive document, the encoders in `encoders/` write PREMIS and METS.

### Descriptive metadata

Four concepts describe the descriptive side ([ADR-0024](decisions/0024-the-metadata-model.md)):

- **Metadata model** (`build.MetadataModel`): what the tool knows about one kind of description: which description type holds it, how it is written as a document, which XSDs that document points at, with their contents (`build.Schema`), and the name and version of its format, which METS records for the document: a name the METS 1.12 `MDTYPE` vocabulary lists is the `MDTYPE`, any other name is `MDTYPE` `OTHER` with the name in `OTHERMDTYPE` (`mets.MDType`), and the version is `MDTYPEVERSION`. The type belongs to the model, so a profile cannot pair a model with another model's type. "Model" means this data model only; `sip/` is the domain model.
- **Description** (`sip.Description`): the description of one entity or representation, a value of the model's description type. It validates itself against its standard, and states what a package-level description must contain.
- **Document format**: the root element, namespace and version that make an XML file a document in the model, plus its file name.
- **Term** (`sip.Term`): a key, an optional language and a value. In a description the key is the element's name as the standard spells it (`dcterms:title` for Meemoo, `title` for Simple Dublin Core); in a `description.csv` it is the CSV's key, which the profile's mapper turns into the element. The library uses the standard's names; the CLI owns the CSV's keys.

Each model lives in its profile's package ([ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md), [ADR-0018](decisions/0018-engine-and-profile-packages.md)), and each description type follows its standard ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)): a Dublin Core document is a flat list, so its type is a list of terms; MODS is a tree, so its type is a struct typed by field.

**Meemoo** (`profiles/meemoo`): one closed table of elements, named as Meemoo's specification names them (`dcterms:isPartOf`, `schema:artMedium`), each with its cardinality and `xsi:type` ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)). Validation and the template both read the table, so a new element is one table row, one key in the CLI's mapper and one line in [input-spec](input-spec.md) §3. A language-tagged element needs a Dutch value. The document carries the entity's identifier: the build swaps it in and keeps the producer's as `MEEMOO-LOCAL-ID`.

**Simple Dublin Core** (`profiles/earkdc`): the fifteen DCMES elements, each optional and repeatable, written unqualified inside a `simpledc` root, the form RODA reads and commons-ip's own sample packages use. The package ships its own `simpledc.xsd` for it: DCMI's 2008 `simpledc.xsd` and `dc.xsd` joined into one schema with the namespace removed, built from DCMI's originals under CC BY 4.0 (the file's header records how). DCMI's file of the same name expects the elements in the DCMES namespace and would reject this document. The document keeps the producer's identifier ([ADR-0012](decisions/0012-eark-keeps-producer-identifier.md)). There is no conversion from Meemoo's model.

**MODS 3.7** (`profiles/earkmods`): one bibliographic record with a local identifier, titles per language, and the library's physical copies as items (call number, optional barcode, optional volume or issue), written as one `location/holdingSimple`. Copies belong on the package-level record, because a copy is never a representation ([ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md)). A further MODS element is a field on the record, a template line, a key in the CLI's mapper and a line in the input spec.

**Supplied documents.** Under the eark profiles a description can also be a finished document supplied as a file (`build.EncodedDescription`, [ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)). The build checks only its root element (`simpledc` without namespace; `mods:mods` in the MODS v3 namespace with `version="3.7"`) and copies it as it is, computing fixity on the way. Schema validity stays with the validators downstream ([ADR-0003](decisions/0003-validation-stays-external.md)). `meemoo/basic` takes no supplied document, because its document must carry the identifier the build mints. The input reader reports a `dc+schema.xml` in a basic folder instead of packaging it as content.

### Preservation metadata

`premis.xml` at package and representation level ([PREMIS](https://www.loc.gov/standards/premis/) 3.0), written by `encoders/premis`: object identifiers, fixity, format registry entries, and the relationships between entity, representations and files (`represents`, `is represented by`, `includes`, `is included in`). Only `meemoo/basic` writes PREMIS. The PREMIS documents point `xsi:schemaLocation` at the remote `https://www.loc.gov/standards/premis/premis.xsd`, as Meemoo SIP 1.2 requires, so the package ships no PREMIS schema.

### Structural metadata

`METS.xml` at package and representation level ([METS](https://www.loc.gov/standards/mets/)), written by `encoders/mets`: the file inventory with checksums and sizes, the references to the descriptive and preservation documents, and the structMap. The package METS points at each representation METS with an `mptr`.

## Profiles

A profile is data: a `build.Definition`, exported by the profile's package under `profiles/` and listed by name in the registry (`profiles.Get`, `profiles.Names`). The fields are documented on the type. The one behavior a profile brings is its metadata model; everything else is a value: the descriptive document's file name, whether the package carries PREMIS and representation types, the profile's rules on representations, and the METS values. All profiles share one writer ([ADR-0007](decisions/0007-profile-families-share-one-writer.md)), and the engine imports no profile ([ADR-0018](decisions/0018-engine-and-profile-packages.md)). A further Meemoo content profile (such as material artwork or newspapers) is a definition in a profile package and one registry line.

`Definition.ValidateSource` checks the profile's rules on a source package. Under `meemoo/basic` they allow exactly one representation (a maximum of one, with the one every package needs) and no description below the package level, as Meemoo SIP 1.2's basic profile requires.

The software agent is not part of the profile: E-ARK CSIP requires every package to name the software that built it, so the engine adds a `CREATOR` agent of type `SOFTWARE`, named SIP Creator, first in every package METS. Its version is the module version Go stamps into the binary: the tag of a tagged `go install`, otherwise a pseudo-version carrying the commit, with `+dirty` when the working tree had uncommitted changes.

The submitting organization is not part of the profile, because one profile serves every organization that uses it. `Definition.WithSubmitter(name, orID)` returns a copy with the organization as a `CREATOR` agent; under `meemoo/basic` it requires the organization's Meemoo OR-id as well. The CLI reads both from `SIP_SUBMITTER_*`; a program using the library passes them as arguments.

## Build lifecycle

`sip-creator create --profile <name> <src> <dest>` (`cli/create_cmd.go`):

1. Resolves `--profile` in the registry; an unknown or empty name lists the available profiles. Adds the submitting organization from the environment.
2. Turns the flags into values on the source package: `--status` into `RecordStatus`, `--updates` into `PackageIdentifier`, and `--content-category` (else `SIP_CONTENT_CATEGORY`, else the profile's value) into `ContentCategory`.
3. Reads the input folder into a `build.SourcePackage` with `input.Read`, reporting every violation at once (see `cli/input/` under [Code organization](#code-organization)).
4. Calls `Build(source)` on a builder from `build.New`.
5. Zips the package directory, unless `--no-zip` is given.

`check --profile <name> <src>` (`cli/check_cmd.go`) runs step 3 and, when the folder was read without violations, `Definition.ValidateSource`, with no configuration. It prints a report on stdout (`cli/check_report.go`): every problem, then a summary of the source package as far as the reader got (where the package description comes from, counts of representations, essence, documentation and PREMIS files, and whether a format report was supplied), then the verdict. It exits with 1 when there are problems and with 2 when it could not check the folder. It does not build. It confirms that each received PREMIS file is well-formed XML and that a supplied characterization report has an entry for every content file. Whether a received file's root is a `premis:premis` element is checked only by `create`.

`build.New` takes the profile, the destination and an optional logger, and refuses a profile without a metadata model. `Builder.Build` takes one `build.SourcePackage` per package. It runs a check, then two separate phases: assemble the complete graph in memory, then write it. Errors are returned, never panicked, and a failed build leaves nothing on disk.

**Check.** First the profile's rules: every description is of the model's type, a supplied document has the right root element, and the profile's rules on representations hold. Then the source package's own rules: names, attribute text, each description against its standard, and the required elements of the package-level description.

**Phase 1: assemble** (`build/assemble.go`) builds the complete graph without writing anything: the package and its METS values (with the source package's record status and content category over the profile's), the entity and its description, every file node with its path and MIME type, the schemas the documents point at (a schema the bundle does not hold stops the build), and which PREMIS documents exist. When a characterization report is supplied, each essence file takes its format, MIME type and checksum from it: a missing entry, an sf error or a missing checksum stops the build, and an entry without a match leaves the format empty. The report's MD5 is taken as given, without reading the file to check it; keeping the report true to the files is the operator's responsibility ([ADR-0032](decisions/0032-the-report-checksum-is-taken-as-given.md)). Documentation files need no entry; one that has an entry takes its checksum from it too. Received PREMIS files must be well-formed `premis:premis` documents. After assembly the writer only writes; it creates no nodes.

**Phase 2: write** (`build/write.go`, through `store/`) writes the graph in one fixed order, set in `write()`. The order follows from what each document records: a representation's METS comes after every file it lists (essence, PREMIS, documentation, its descriptive document), and the package METS comes last, after everything else. Fixity is computed while each file is copied or rendered, so it describes the bytes in the package, and the writer fills it into the graph for the METS documents written after it. The exception is a file whose checksum a characterization report supplies: the copy then computes none and the report's MD5 stands ([ADR-0032](decisions/0032-the-report-checksum-is-taken-as-given.md)). The package is written into `dest/.<identifier>.tmp` and renamed to `dest/<identifier>` when complete; a package directory that already exists is refused, never written into, and a failed write removes the temporary directory ([ADR-0031](decisions/0031-final-names-hold-complete-output.md)).

## Code organization

`sip/` is the shared domain: the assembler builds its graph, and the writer and encoders read it.

- **`sip/`**: the domain model as plain data: the package graph, the `Description` interface each profile's description type implements, the record status vocabulary and the identifier format. It does no I/O, no validation and imports no profile, so templates, the engine and every profile can share it. Its fields are set by the assembler, which keeps the graph's invariants.
- **`build/`**: the library's face and the engine. Everything a program using the library works with lives here: the definition of a profile, the metadata model interface, the source package and its validation, the builder. The engine runs the check and the two phases described under [Build lifecycle](#build-lifecycle) and imports no profile.
- **`profiles/`**: the registry the CLI uses for `--profile`, and one package per profile (`meemoo` for `meemoo/basic`, `earkdc` for `eark/dc`, `earkmods` for `eark/mods`). A profile package holds everything that needs its concrete description type: the type and its rules, the template, the list of XSDs its document points at, and the exported definition. The in-tree profiles take their XSDs from the bundle in `schemas/` (`build.BundledSchemas`).
- **`encoders/`**: the METS and PREMIS documents, rendered from the graph with `text/template` ([ADR-0002](decisions/0002-xml-via-text-template.md)). Every profile difference arrives as data. The templates escape every value they read from the graph and percent-encode every `xlink:href`, so file names, labels and agent names may hold any character XML can carry; `build.ValidateXMLText` refuses the rest before anything is written ([ADR-0028](decisions/0028-encoders-escape-every-value.md)). METS elements that describe a graph node carry that node's identifier, so METS and PREMIS agree on which file is which; other METS IDs are minted per render. `encoders/xmldoc` reads the root element of a well-formed document and is the tool's only XML reader.
- **`store/`**: writes files into the package directory, by package-relative path. A copy computes its checksum as it streams, unless the caller already holds one, and reads its size from the written file; a rendered document is written only when the template succeeds; every write replaces what was there.
- **`schemas/`**: the XSDs the METS documents and the in-tree profiles need, embedded in the binary. A package ships only those its documents point at. A profile outside this module supplies its own XSDs; the build checks each name and refuses empty contents or two different schemas under one name.
- **`characterization/`**: decodes a Siegfried report into per-file records. It records what the report says without judging it; the assembler decides what is required ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). Another report format would be one more decoder.
- **`archive/`**: zips a package directory without compression.
- **`cli/`**: the `create` and `check` commands and the environment configuration. All unexported: configuration belongs to the CLI, not to the library.
- **`cli/input/`**: reads an input folder into a `build.SourcePackage`, in two steps ([ADR-0027](decisions/0027-input-reader-walks-then-decodes.md)): a walk that judges only names, kinds and places, then decoders that read the files the walk found and report each finding with its file and line. It imports no profile: it takes a mapper for `description.csv` and a description of the document a profile accepts ([ADR-0023](decisions/0023-cli-input-one-package.md)).
- **`cli/input/mapping/`**: one mapper per profile, from the rows of `description.csv` to the profile's description. It is the only package under `cli/input` that imports the profiles, and it does not import `cli/input`, so the reader's tests can use the real mappers.

Dependencies: cobra (CLI), godotenv and caarlos0/env (configuration), golang.org/x/text (Unicode normalization of input paths). UUIDs come from the standard library. No characterization tool runs or is a dependency.

## CLI/library boundary

The library is meant to be embedded in systems that automate ingest workflows, which hold content in storage and metadata as structured data, not in prepared folders. Its API is the contract; the input folder is one way to fill it.

- **The library owns the domain, not the input.** It never sees a CSV or assumes a folder layout. A program builds a `build.SourcePackage` from its own data.
- **The CLI owns the input folder.** It reads the folder ([input-spec.md](input-spec.md)) and the configuration, reports violations in plain language, and calls the library.
- **Validation has two layers.** The CLI reports input-folder violations (an unknown CSV key, a misplaced file) with the file and line. The library validates the data itself before every write, so a program that never uses the folder meets the same rules; the CLI calls the same description checks on a rows file, so both say the same thing.
- **Characterization is data** ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)): a program sets `SourcePackage.Characterization`; the CLI fills it from `siegfried.json`. The same checks apply to both, and both take the report's checksums as given ([ADR-0032](decisions/0032-the-report-checksum-is-taken-as-given.md)).
- **A program can keep a profile of its own** ([ADR-0033](decisions/0033-ugent-first-profiles-of-your-own.md)): the library serves UGent's profiles first, and the extension point the in-tree profiles use is open to a program that embeds it: a description type implementing `sip.Description`, a model implementing `build.MetadataModel` that supplies its own XSDs, and a `build.Definition` passed to `build.New`. `Example_ownProfile` in `build/example_test.go` builds a package that way. The registry lists only what `--profile` accepts. Where a profile leaves a value to the program that builds the description, it offers typed constants to pick from.
- **Two choices in the eark profiles follow RODA, not E-ARK** ([ADR-0013](decisions/0013-representation-type-from-label.md)): they write no PREMIS, because RODA drops package PREMIS that describes no agents or events, and they put the representation type in the content typing, because RODA reads it there. They stay until a consumer needs otherwise.

## Input

[input-spec.md](input-spec.md) defines the input folder and every rule for it. In short: one folder is one package, with one description per level (`description.csv` in the profile's keys, or under the eark profiles a finished `dc.xml` or `mods.xml`), content either flat or under `representations/<name>/`, and optional `representations.csv`, `documentation/`, `premis/` and `siegfried.json`. Administrative values come from configuration ([ADR-0010](decisions/0010-config-over-self-describing-input.md)).

## Validation

Generated packages are validated externally with commons-ip, the E-ARK CSIP reference validator; CSIP rules are not reimplemented as Go tests ([ADR-0003](decisions/0003-validation-stays-external.md), [ADR-0005](decisions/0005-dockerized-validation-and-html-reporting.md)). Each profile is validated against its own E-ARK SIP version: `meemoo/basic` against 2.0.4, whose profile URL Meemoo 1.2 requires, and the eark profiles against 2.2.0. commons-ip does not validate the descriptive documents the METS points at, so every `mods.xml`, `dc.xml` and `dc+schema.xml` is also validated with xmllint against the schema the package ships for it. Meemoo's `descriptive_basic.xsd` checks element names only: Meemoo SIP 1.2's cardinality and required elements are the library's checks, and EDTF date syntax is checked by neither ([TODO.md](TODO.md)). Go tests cover what the validator cannot see: internal contracts and failure paths, such as fixity in the store, every reference in a built package naming a file with its size and checksum, assembly writing nothing to disk, the refusals of supplied documents, the input reader's violations, and each profile's template agreeing with its XSD list. [CONTRIBUTING.md](../CONTRIBUTING.md) has the commands.

## Known gaps

Tracked in [TODO.md](TODO.md):

- **PREMIS events are not modeled.** `sip.Event` is an empty stub.
- **Essence arrives as file paths, not streams, and fixity cannot be supplied pre-computed.** A program using the library can build a `SourcePackage` without the input folder, but its essence must be files on disk.
- **On Windows, the final rename of the package or the zip can fail** while a virus scanner holds a just-written file open; the build reports an error and a rerun works.
- **Not all administrative values are implemented.** [ADR-0010](decisions/0010-config-over-self-describing-input.md) assigns the archival creator, contact persons and a submission agreement reference to configuration; the tool reads and writes none of them yet.
