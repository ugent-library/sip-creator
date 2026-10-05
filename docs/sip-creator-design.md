# SIP Creator design

SIP Creator is a Go library and CLI that builds a Submission Information Package (SIP) from a producer's essence files and descriptive metadata. It copies the essence into the package, computing fixity during the copy, optionally adds format info from a pre-computed characterization report, and generates the descriptive, preservation and structural XML. Every package is an [E-ARK SIP](https://earksip.dilcis.eu/). Three profiles decide the rest:

| Profile | Package | Descriptive metadata | E-ARK SIP version |
|---|---|---|---|
| `basic` | [Meemoo SIP 1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) basic content profile (the stable version; 2.0 and 2.1 are release candidates) | Meemoo's Dublin Core terms and schema.org (`dc+schema.xml`) | 2.0.4 |
| `eark` | plain E-ARK SIP | Simple Dublin Core (`dc.xml`) | 2.2.0 |
| `eark-mods` | plain E-ARK SIP | [MODS 3.7](https://www.loc.gov/standards/mods/) (`mods.xml`) | 2.2.0 |

The BagIt envelope Meemoo's transfer requires is out of scope ([ADR-0008](decisions/0008-bag-layer-out-of-scope.md)): bag the package directory with a reference BagIt implementation.

This document describes the system as it is today. The reasons for key choices are in the [decision records](decisions/), planned changes in [plans/](plans/), open questions in [TODO.md](TODO.md). [README.md](README.md) explains how the docs are organized; the repository's `CLAUDE.md` holds the coding conventions.

> **Status: experimental.** The sample package of each profile validates `VALID` with zero warnings against commons-ip, and every `mods.xml` in an `eark-mods` package validates against the MODS 3.7 schema with xmllint.

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
      dc+schema.xml            the descriptive document: dc.xml under eark, mods.xml under eark-mods
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
          dc.xml               optional, eark profiles only: a description of this version (mods.xml under eark-mods)
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

The CLI zips the directory **uncompressed** (`zip.Store`) to `dest/<identifier>.zip`, unless `--no-zip` is given. Under the eark profiles the zip is the deliverable; under `basic` the deliverable is the package directory in a BagIt bag, which this tool does not produce.

## Metadata

Each kind of metadata has its own standard and its own generator: the profile writes the descriptive document, the encoders in `encoders/` write PREMIS and METS.

### Descriptive metadata

Four concepts describe the descriptive side ([ADR-0024](decisions/0024-the-metadata-model.md)):

- **Metadata model** (`build.MetadataModel`): what the tool knows about one kind of description: which description type holds it, how it is written as a document, which XSDs that document points at, and how METS types the document (`MDTYPE`, `MDTYPEVERSION`). The type belongs to the model, so a profile cannot pair a model with another model's type. "Model" means this data model only; `sip/` is the domain model.
- **Description** (`sip.Description`): the description of one entity or representation, a value of the model's description type. It validates itself against its standard, and states what a package-level description must contain.
- **Document format**: the root element, namespace and version that make an XML file a document in the model, plus its file name.
- **Term** (`sip.Term`): a key, an optional language and a value. In a description the key is the element's name as the standard spells it (`dcterms:title` for Meemoo, `title` for Simple Dublin Core); in a `description.csv` it is the CSV's key, which the profile's mapper turns into the element. The library uses the standard's names; the CLI owns the CSV's keys.

Each model lives in its profile's package ([ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md), [ADR-0018](decisions/0018-engine-and-profile-packages.md)), and each description type follows its standard ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)): a Dublin Core document is a flat list, so its type is a list of terms; MODS is a tree, so its type is a struct typed by field.

**Meemoo** (`profiles/meemoo`): one closed table of elements, named as Meemoo's specification names them (`dcterms:isPartOf`, `schema:artMedium`), each with its cardinality and `xsi:type` ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)). Validation and the template both read the table, so a new element is one table row, one key in the CLI's mapper and one line in [input-spec](input-spec.md) §3. A language-tagged element needs a Dutch value. The document carries the entity's identifier: the build swaps it in and keeps the producer's as `MEEMOO-LOCAL-ID`.

**Simple Dublin Core** (`profiles/eark`): the fifteen DCMES elements, each optional and repeatable, written unqualified. The document keeps the producer's identifier ([ADR-0012](decisions/0012-eark-keeps-producer-identifier.md)). There is no conversion from Meemoo's model.

**MODS 3.7** (`profiles/earkmods`): one bibliographic record with a local identifier, titles per language, and the library's physical copies as items (call number, optional barcode, optional volume or issue), written as one `location/holdingSimple`. Copies belong on the package-level record, because a copy is never a representation ([ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md)). A further MODS element is a field on the record, a template line, a key in the CLI's mapper and a line in the input spec.

**Supplied documents.** Under the eark profiles a description can also be a finished document supplied as a file (`build.EncodedDescription`, [ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)). The build checks only its root element (`simpledc` without namespace; `mods:mods` in the MODS v3 namespace with `version="3.7"`) and copies it as it is, computing fixity on the way. Schema validity stays with the validators downstream ([ADR-0003](decisions/0003-validation-stays-external.md)). `basic` takes no supplied document, because its document must carry the identifier the build mints.

### Preservation metadata

`premis.xml` at package and representation level ([PREMIS](https://www.loc.gov/standards/premis/) 3.0), written by `encoders/premis`: object identifiers, fixity, format registry entries, and the relationships between entity, representations and files (`represents`, `is represented by`, `includes`, `is included in`). Only `basic` writes PREMIS. The PREMIS documents point `xsi:schemaLocation` at the remote `https://www.loc.gov/standards/premis/premis.xsd`, as Meemoo SIP 1.2 requires, so the package ships no PREMIS schema.

### Structural metadata

`METS.xml` at package and representation level ([METS](https://www.loc.gov/standards/mets/)), written by `encoders/mets`: the file inventory with checksums and sizes, the references to the descriptive and preservation documents, and the structMap. The package METS points at each representation METS with an `mptr`.

## Profiles

A profile is data: a `build.Definition`, exported by the profile's package under `profiles/` and listed by name in the registry (`profiles.Get`, `profiles.Names`). The fields are documented on the type. The one behavior a profile brings is its metadata model; everything else is a value: the descriptive document's file name, whether the package carries PREMIS and representation types, the profile's rules on representations, and the METS values. All profiles share one writer ([ADR-0007](decisions/0007-profile-families-share-one-writer.md)), and the engine imports no profile ([ADR-0018](decisions/0018-engine-and-profile-packages.md)). A further Meemoo content profile (such as material artwork or newspapers) is a definition in a profile package and one registry line.

`Definition.ValidateSource` checks the profile's rules on a source package. Under `basic` they allow exactly one representation (a maximum of one, with the one every package needs) and no description below the package level, as Meemoo SIP 1.2's basic profile requires.

The submitting organization is not part of the profile, because one profile serves every organization that uses it. `Definition.WithSubmitter(name, orID)` returns a copy with the organization as a `CREATOR` agent; under `basic` it requires the organization's Meemoo OR-id as well. The CLI reads both from `SIP_SUBMITTER_*`; a program using the library passes them as arguments.

## Build lifecycle

`sip-creator create --profile <name> <src> <dest>` (`cli/create_cmd.go`):

1. Resolves `--profile` in the registry; an unknown or empty name lists the available profiles. Adds the submitting organization from the environment.
2. Turns the flags into values on the source package: `--status` into `RecordStatus`, `--updates` into `PackageIdentifier`, and `--content-category` (else `SIP_CONTENT_CATEGORY`, else the profile's value) into `ContentCategory`.
3. Reads the input folder into a `build.SourcePackage` with `input.Read`, reporting every violation at once (see `cli/input/` under [Code organization](#code-organization)).
4. Calls `Build(source)` on a builder from `build.New`.
5. Zips the package directory, unless `--no-zip` is given.

`check --profile <name> <src>` (`cli/check_cmd.go`) runs step 3 and then `Definition.ValidateSource`, with no configuration. It does not build, so the checks on file contents (received PREMIS, the characterization report's checksums) run only in `create`.

`build.New` takes the profile, the destination and a logger, and refuses a profile without a metadata model. `Builder.Build` takes one `build.SourcePackage` per package. It runs a check, then two separate phases: assemble the complete graph in memory, then write it. Errors are returned, never panicked, and a failure before the write phase leaves nothing on disk.

**Check.** First the profile's rules: every description is of the model's type, a supplied document has the right root element, and the profile's rules on representations hold. Then the source package's own rules: names, attribute text, each description against its standard, and the required elements of the package-level description.

**Phase 1: assemble** (`build/assemble.go`) builds the complete graph without writing anything: the package and its METS values (with the source package's record status and content category over the profile's), the entity and its description, every file node with its path and MIME type, the schemas the documents point at (a schema the bundle does not hold stops the build), and which PREMIS documents exist. When a characterization report is supplied, each essence file takes its format from it, and the report is strict: a missing entry, an sf error, a missing checksum or an MD5 that doesn't match the source file stops the build; an entry without a match leaves the format empty. Documentation files need no entry, but their checksum is checked when one exists. Received PREMIS files must be well-formed `premis:premis` documents. After assembly the writer only writes; it creates no nodes.

**Phase 2: write** (`build/write.go`, through `store/`) writes the graph in one fixed order, set in `write()`. The order follows from what each document records: a representation's METS comes after every file it lists (essence, PREMIS, documentation, its descriptive document), and the package METS comes last, after everything else. Fixity is computed while each file is copied or rendered, so it describes the bytes in the package, and the writer fills it into the graph for the METS documents written after it.

## Code organization

`sip/` is the shared domain: the assembler builds its graph, and the writer and encoders read it.

- **`sip/`**: the domain model as plain data: the package graph, the `Description` interface each profile's description type implements, the record status vocabulary and the identifier format. It does no I/O, no validation and imports no profile, so templates, the engine and every profile can share it. Its fields are set by the assembler, which keeps the graph's invariants.
- **`build/`**: the library's face and the engine. Everything a program using the library works with lives here: the definition of a profile, the metadata model interface, the source package and its validation, the builder. The engine runs the check and the two phases described under [Build lifecycle](#build-lifecycle) and imports no profile.
- **`profiles/`**: the registry the CLI uses for `--profile`, and one package per profile (`meemoo` for `basic`, `eark`, `earkmods` for `eark-mods`). A profile package holds everything that needs its concrete description type: the type and its rules, the template, the XSDs its document points at, and the exported definition.
- **`encoders/`**: the METS and PREMIS documents, rendered from the graph with `text/template` ([ADR-0002](decisions/0002-xml-via-text-template.md)). Every profile difference arrives as data. METS elements that describe a graph node carry that node's identifier, so METS and PREMIS agree on which file is which; other METS IDs are minted per render. `encoders/xmldoc` reads the root element of a well-formed document and is the tool's only XML reader.
- **`store/`**: writes files into the package directory, by package-relative path. A copy computes its checksum and size as it streams; a rendered document is written only when the template succeeds; every write replaces what was there.
- **`schemas/`**: every XSD the profiles need, embedded in the binary. A package ships only those its documents point at.
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
- **Characterization is data** ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)): a program sets `SourcePackage.Characterization`; the CLI fills it from `siegfried.json`. The same strict checks apply to both.
- **Other institutions bring their own profile** ([ADR-0022](decisions/0022-reference-implementation-bring-your-own-profile.md)): a description type implementing `sip.Description`, a model implementing `build.MetadataModel`, and a `build.Definition` passed to `build.New`. The registry lists only what `--profile` accepts. Where a profile leaves a value to the program that builds the description, it offers typed constants to pick from.
- **Two choices in the eark profiles follow RODA, not E-ARK** ([ADR-0013](decisions/0013-representation-type-from-label.md)): they write no PREMIS, because RODA drops package PREMIS that describes no agents or events, and they put the representation type in the content typing, because RODA reads it there. They stay until a consumer needs otherwise.

## Input

[input-spec.md](input-spec.md) defines the input folder and every rule for it. In short: one folder is one package, with one description per level (`description.csv` in the profile's keys, or under the eark profiles a finished `dc.xml` or `mods.xml`), content either flat or under `representations/<name>/`, and optional `representations.csv`, `documentation/`, `premis/` and `siegfried.json`. Administrative values come from configuration ([ADR-0010](decisions/0010-config-over-self-describing-input.md)).

## Validation

Generated packages are validated externally with commons-ip, the E-ARK CSIP reference validator; CSIP rules are not reimplemented as Go tests ([ADR-0003](decisions/0003-validation-stays-external.md), [ADR-0005](decisions/0005-dockerized-validation-and-html-reporting.md)). Each profile is validated against its own E-ARK SIP version: `basic` against 2.0.4, whose profile URL Meemoo 1.2 requires, and the eark profiles against 2.2.0. commons-ip does not validate the descriptive documents the METS points at, so every `mods.xml` is also validated with xmllint against the bundled MODS 3.7 schema. Go tests cover what the validator cannot see: internal contracts and failure paths, such as fixity in the store, assembly writing nothing to disk, the refusals of supplied documents, the input reader's violations, and each profile's template agreeing with its XSD list. [CONTRIBUTING.md](../CONTRIBUTING.md) has the commands.

## Known gaps

Tracked in [TODO.md](TODO.md):

- **PREMIS events are not modeled.** `sip.Event` is an empty stub.
- **Essence arrives as file paths, not streams, and fixity cannot be supplied pre-computed.** A program using the library can build a `SourcePackage` without the input folder, but its essence must be files on disk.
- **Not all administrative values are implemented.** [ADR-0010](decisions/0010-config-over-self-describing-input.md) assigns the archival creator, contact persons and a submission agreement reference to configuration; the tool reads and writes none of them yet.
