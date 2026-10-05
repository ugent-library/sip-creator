# SIP Creator input specification

Status: **Current** (2026-10-01). This is the input contract the tool enforces. Implemented by the [input-convention plan](archive/input-convention.md) with [ADR-0010](decisions/0010-config-over-self-describing-input.md); the §3 vocabulary by the [descriptive-vocabulary plan](archive/descriptive-vocabulary.md) with [ADR-0011](decisions/0011-closed-descriptive-vocabulary.md); the supplied document of §3 by the [descriptive-model plan](archive/descriptive-model.md) with [ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md).

This document describes how to prepare a folder so that the SIP Creator **CLI** can turn it into an E-ARK submission package. It is written for the people preparing material; the section [Mapping to the SIP](#7-mapping-to-the-sip-informative-for-specialists) at the end is for specialists and explains how each rule lands in the E-ARK CSIP/SIP structure.

The key words MUST, SHOULD and MAY are to be interpreted as in RFC 2119.

## Scope and architecture

This spec describes the CLI's input convention only. The underlying library exposes a programmatic builder API (domain model in `sip/`); this input format is one frontend that maps onto it. Larger systems that automate ingest workflows construct the same model directly from their own stores without using this input format; see the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc. Nothing in this spec is required to use the library.

## What you prepare

One folder = one package. In the simplest case:

```
example-0001/
├── description.csv       ← describes the content (the only file you write)
└── ... your files ...
```

With multiple versions of the content and extras:

```
example-0001/
├── description.csv
├── representations.csv   ← optional: a label and type per representation
├── representations/
│   ├── master/           ← the archival scans, any structure you like
│   └── access/           ← e.g. a PDF version
│       └── description.csv  ← optional: describes just this version (e.g. its license)
├── documentation/        ← optional: scan reports, context material
└── premis/               ← optional: preservation XML received from a vendor
```

The submitting organization does **not** live in the folder: it rarely changes and comes from the tool's configuration. See [What comes from configuration](#6-what-comes-from-configuration-and-the-command-line). Neither does the profile: you pass it to `check` and `create` with `--profile`, and it says which keys `description.csv` takes and whether a finished `dc.xml` or `mods.xml` may stand in for it (§3).

## 1. General rules

- One input folder MUST correspond to one package.
- Six names at the top level are reserved under every profile: `description.csv` (the descriptive rows, see §3), `representations/`, `representations.csv`, `documentation/`, `premis/`, and `siegfried.json` (each optional; `representations.csv` names representations, see §2; `siegfried.json` is the pre-computed characterization report, see §2). Under the two eark profiles a seventh is: the profile's descriptive document, `dc.xml` under `eark` and `mods.xml` under `eark-mods`, which may stand in for `description.csv` (§3). All other folder and file names are free, with any nesting; that includes the rows file's former names (`metadata.csv`, `dcschema.csv`, `dc.csv`) and a document name the profile does not take (`dc.xml` under `basic`, `mods.xml` under `eark`), which are content like any other file.
- Operating-system artifacts (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `._*`) MUST be ignored by the tool: never packaged, never warned about.
- Symbolic links anywhere in the input MUST be an error.
- The tool MUST compare paths after Unicode canonical normalization (NFC), because macOS file names and typed CSV values often differ only in normalization form.
- The tool MUST refuse to build when any MUST rule is violated, and MUST report all violations at once, in plain language, naming the file or folder concerned. SHOULD violations produce warnings.
- The tool MUST offer a check-only mode that validates a folder's structure and metadata against the rules here without building anything. It takes the same profile a build takes, because the profile says which keys the rows may use (§3), and no configuration. Content-level verification (the characterization report's checksum checks in §2 and the PREMIS conformance of received preservation files in §5) happens at build, not at check: those rules apply to whoever supplies the data, however it arrives.

## 2. Content files and representations

A *representation* is one version of the content: the archival master scans are one, a derived PDF is another. Every package has at least one.

- **Simple case:** if there is no `representations/` folder, everything in the package folder (apart from the reserved names) is the content of a single representation, named after the input folder itself.
- **Multiple versions:** if `representations/` exists, each folder directly inside it is one representation, named by its folder name. All content MUST then live inside `representations/`; content files elsewhere at the top level are an error (except inside `documentation/` and `premis/`). Under `basic` the folder MUST hold exactly one representation: Meemoo SIP 1.2's basic profile says "The IE MUST be represented by exactly one representation." The eark profiles set no limit.
- Representation names (the folder names, or the input folder's own name in the simple case) MUST match `A–Z a–z 0–9 . _ -`. The name is used as-is inside the final package: it becomes the representation's directory name under `representations/` and, unless `representations.csv` says otherwise, its human-readable name and type in the generated metadata. Neither E-ARK CSIP nor the Meemoo specification dictates a naming scheme; CSIP requires only that the names be unique, which folder names are by construction.
- Inside a representation folder, three names are reserved: `description.csv`, `documentation/` and `premis/` (all optional, see §3–5), plus the profile's document name under the eark profiles (§3). Everything else is content, with free naming and nesting.
- Files are packaged in a stable, tool-determined order (alphabetical by path). This order carries no meaning: neither E-ARK CSIP nor the Meemoo specification assigns semantics to file order. If a human-readable sequence matters to you, zero-pad your numbering (`0001.tiff`, `0002.tiff`, …); explicit ordering is a deferred feature (see §8, the manifest).
- The tool computes checksums and sizes itself; you never supply those. File formats come from an optional pre-computed characterization report (`siegfried.json` at the top level, generated from the input root with `sf -hash md5 -json`) that the tool verifies against the files before trusting; you never hand-author format info ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). The report may be generated on another operating system than the build runs on: the tool reads backslashes in its paths as folder separators, so a file whose name itself contains a backslash cannot be matched to its report entry: a content file is then refused, a documentation file goes without format info.

### `representations.csv`: labels and types (optional)

A folder name makes a good machine name but not always a good display name or type. An optional `representations.csv` at the top level, next to `description.csv`, gives each representation a label (its human-readable name in the generated metadata) and a type (what an ingest system such as RODA shows as the representation's kind):

```csv
folder,label,type
master,Master scan (TIFF),archival
access,Access copy (PDF),access
```

- MUST be UTF-8 with a header row naming its columns. The accepted columns, in any order, are `folder` (required), `label`, and `type` (each optional; header matching is case-insensitive, so a spreadsheet's `Folder` works too). An unknown or repeated column name MUST be an error: a typo must not silently drop a column. A UTF-8 BOM, CRLF line endings, and RFC 4180 quoting are accepted, as for `description.csv`.
- `folder` names a folder directly under `representations/` by its bare name (`master`, not a path). Every row MUST match an existing folder, no two rows may name the same folder, and every folder MUST have a row. A folder without a row is an error, never an exclusion: silently dropping content from the package is the one thing this file must never cause. To leave material out, move it out of the input folder.
- An empty `label` cell means the folder name; an empty `type` cell means the label. A file listing only folder names changes nothing about the output.
- `label` and `type` values are emitted into the package's XML verbatim, so the characters `< > & "` MUST be an error.
- The rows' order is the order the representations take in the package.
- The file requires a `representations/` folder: in the simple flat case there is one representation named after the input folder, and a `representations.csv` MUST be an error.
- The type reaches the output only in profiles that declare representation types (the `eark` and `eark-mods` profiles; [ADR-0013](decisions/0013-representation-type-from-label.md)). Meemoo profiles fix their representation typing to the Meemoo profile URI, so `type` has no effect there; `label` is emitted for every profile.

## 3. Descriptive metadata: `description.csv`, or a supplied document

A two-column CSV (`key,value`) describing what the package contains. Usually this is the only file an operator writes; under the eark profiles a finished document may stand in for it (see [Supplying a finished document](#supplying-a-finished-document-eark-and-eark-mods) below). The profile passed to `check` and `create` (`--profile`) says which keys its rows may use and which document, if any, may replace it:

| profile | `description.csv` keys | document that may stand in for it |
|---|---|---|
| `basic` | Meemoo's dc+schema keys: the key table below | none |
| `eark` | Simple Dublin Core | `dc.xml`, a `simpledc` document |
| `eark-mods` | the MODS keys `identifier` and `title` | `mods.xml`, a `mods:mods` document declaring MODS 3.7 |

Under `basic` the file takes the key table below. Under `eark` it takes the fifteen Simple Dublin Core elements as keys: `title`, `creator`, `subject`, `description`, `publisher`, `contributor`, `date`, `type`, `format`, `identifier`, `source`, `language`, `relation`, `coverage`, `rights`. Every one is optional and repeatable, `identifier` and `title` MUST be present at the top level, and a language tag is accepted but not written into the document. Under `eark-mods` it takes two keys: `identifier` (once, no language tag: the record's local identifier, emitted as a `mods:identifier` of type `local`) and `title` (once per language, emitted as a `mods:titleInfo/mods:title` with `xml:lang`). The rows carry flat terms about the record only; its physical copies, and any structure richer than these keys, travel in a supplied `mods.xml` ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)). The key tables are separate: a Meemoo key under `eark` is an unknown key, not a silently dropped one ([ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md)). Until 2026-09-28 the file was named after its keys (`dcschema.csv`, `dc.csv`); the profile decides now ([ADR-0016](decisions/0016-descriptive-input-rows-or-supplied-document.md)).

- At the top level, exactly one of `description.csv` and the profile's document MUST be present (under `basic`, `description.csv`). Both, or neither, is an error.
- MUST be UTF-8 with a `key,value` header row. The tool MUST accept a UTF-8 BOM and CRLF line endings (spreadsheet tools produce both) and RFC 4180 quoting.
- Under `basic`, `identifier`, `title`, `description` and `created` MUST be present and non-empty: Meemoo's basic content profile requires all four. Under `eark` and `eark-mods`, `identifier` and `title` MUST be present. The identifier is your local catalog or inventory number; it travels with the package as its local identifier. `check` reports a missing one.
- Repeat a key for multiple values (two `creator` lines for two creators), but only for keys the table lists as repeatable. Keys listed as *per-language* may repeat only with distinct language tags (`title[nl]` plus `title[en]` is fine; two `title[nl]` rows are not). A second row for a single-valued key, or a repeated language on a per-language key, MUST be an error.
- Add a language tag in square brackets where the language matters: `title[nl]`, `description[en]`. Under `basic`, wherever a language-tagged key is used, a Dutch entry (`[nl]`) MUST be among the rows (Meemoo's rule); other languages are welcome alongside, but Dutch must be present. `check` reports a missing Dutch entry.
- Unknown keys MUST be an error: a typo must not silently drop metadata. Keys are matched case-insensitively (`Title` reads as `title`). The table below lists every key under `basic`; it follows the flat-expressible elements of Meemoo's basic content profile.

Supported keys, with the element each one states (findings about a whole file, such as a missing required value or a repeat, name the element; the specialist mapping is in [§7](#7-mapping-to-the-sip-informative-for-specialists)):

| key | element | meaning | repeatable |
|---|---|---|---|
| `identifier` | `dcterms:identifier` | local catalog/inventory number (required) | no |
| `title` | `dcterms:title` | title of the work (required) | per-language |
| `description` | `dcterms:description` | free-text description (required) | per-language |
| `created` | `dcterms:created` | creation date of the original (year or ISO date) (required) | no |
| `alternative` | `dcterms:alternative` | alternative title | yes |
| `abstract` | `dcterms:abstract` | summary or abstract | per-language |
| `creator` | `dcterms:creator` | maker of the work (photographer, author, artist) | yes |
| `contributor` | `dcterms:contributor` | other contributors | yes |
| `publisher` | `dcterms:publisher` | publisher | yes |
| `issued` | `dcterms:issued` | date of issue or publication | no |
| `available` | `dcterms:available` | date the material became available | no |
| `subject` | `dcterms:subject` | subject keyword | yes |
| `spatial` | `dcterms:spatial` | place depicted or covered | yes |
| `temporal` | `dcterms:temporal` | period covered | yes |
| `extent` | `dcterms:extent` | extent (e.g. "48 foto's") | no |
| `language` | `dcterms:language` | language of the content | yes |
| `type` | `dcterms:type` | kind of work | yes |
| `ispartof` | `dcterms:isPartOf` | collection or series this belongs to | yes |
| `license` | `dcterms:license` | license on the content | yes |
| `rights` | `dcterms:rights` | rights statement | per-language |
| `rightsholder` | `dcterms:rightsHolder` | rights holder | no |
| `artmedium` | `schema:artMedium` | material or medium of an artwork | yes |
| `artform` | `schema:artform` | form of an artwork | yes |

Example:

```csv
key,value
identifier,example-0001
title[nl],Voorbeeldfoto
title[en],Example photograph
description[nl],Een voorbeeldpakket met één afbeelding.
created,2026-01-15
creator,Example Studio
subject[nl],voorbeelden
subject[nl],fotografie
extent[nl],1 foto
rights[nl],publiek domein
```

### Describing one representation

Under the eark profiles a representation MAY carry its own rows file (at `representations/<name>/description.csv`) when something is true of that version only, typically a license or rights statement that differs between the master and an access copy. Same format and rules as the package-level file, with two differences:

- `identifier` and `title` are NOT required: the package-level description covers the work's identity. A `title` MAY still be given as a human-readable name for the version (e.g. "PDF-versie").
- It describes the representation, not the work: keys like `created` or `creator` here refer to the making of this version.

Under the eark profiles the profile's document MAY stand in for the rows here too (`representations/<name>/dc.xml` or `mods.xml`), one or the other, never both. In the simple case without a `representations/` folder there is no place for either file, by design; the simple case stays simple.

Under `basic` a representation MUST NOT carry a description: Meemoo SIP 1.2's basic profile says "There MUST NOT be any descriptive metadata at the representation level." A `description.csv` in a representation folder is reported by `check` and refused by `create`.

### Supplying a finished document (`eark` and `eark-mods`)

A record that already exists as a document, or one richer than the rows can say (a MODS record with its physical copies, names with roles), travels as a file instead of rows: `dc.xml` under `eark`, `mods.xml` under `eark-mods`, at the top level or inside a representation folder. The tool copies it into the package as it is, computes its checksum on the way, and references it from the METS like a generated document ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)).

- At each level the document stands in for `description.csv`: one or the other, never both. At the top level one of the two MUST be present.
- The file MUST be well-formed XML whose root element is the profile's standard: `simpledc` without namespace under `eark`, the shape the tool itself writes (a Dublin Core export in another wrapper, such as `oai_dc:dc`, must be rewrapped); `mods:mods` in the MODS v3 namespace (`http://www.loc.gov/mods/v3`) declaring `version="3.7"` under `eark-mods`, the version the package's METS declares. `check` reports a file that fails either.
- Nothing else in the document is checked: not its validity against the schema, not whether it states an identifier or a title ([ADR-0003](decisions/0003-validation-stays-external.md)). Schema validity is the producer's, and the validators downstream check it. The required-keys rule above applies to rows only.
- Name the file as the package names it: `dc.xml`, `mods.xml`. A file carrying another standard's document name (`mods.xml` under `eark`) is content, not a document: the profile's METS would otherwise declare a type the file does not have.
- The `basic` profile takes no document: Meemoo's document must carry the package identifier the tool mints, and the tool does not edit XML. Under `basic`, a `dc.xml` or `mods.xml` is content like any other file.

## 4. Documentation

Context material that is not itself the preserved content: scan reports, correspondence, finding aids.

- Files under `documentation/` at the top level document the whole package.
- Files under `representations/<name>/documentation/` document that representation.
- Free naming and nesting inside.

## 5. Received preservation files (`premis/`)

Digitization vendors and lab equipment sometimes deliver preservation metadata as PREMIS XML (events such as "scanned on this device, on this date"). You never write these files yourself. If you received them, put them in:

- `premis/` at the top level (about the whole package), or
- `representations/<name>/premis/` (about one representation).

Rules:

- Files here MUST be well-formed XML whose root is a `premis:premis` element in the PREMIS 3 namespace, and SHOULD be schema-valid PREMIS 3.0. The tool checks the former at build time and leaves schema validation to the external validators, like all content validation. They are included in the package as received: not parsed, edited, or merged.
- Because these files cannot know the identifiers the tool generates, they SHOULD identify their subject using local identifiers built from your `identifier` and the representation name (e.g. `example-0001-master`), so a future reader can correlate them with the generated preservation metadata.

## 6. What comes from configuration and the command line

These values span many packages or belong to the run, so they do not live in the package folder. The environment variables are listed in [CONFIG.md](../CONFIG.md).

| value | source |
|---|---|
| submitting organization: name | `SIP_SUBMITTER_NAME`, required by `create` under every profile |
| submitting organization: Meemoo OR-id | `SIP_SUBMITTER_OR_ID`, required by `create` under `basic` |
| content category (e.g. `Photographs – Digital`) | `--content-category`, else `SIP_CONTENT_CATEGORY`, else the profile's value |
| profile | `--profile`, required by `check` and `create` |
| record status | `--status` |
| identifier of the package this one updates | `--updates` |

Creating vs. updating:

- Without `--status` the package carries no record status, which the E-ARK SIP specification reads as new.
- To submit a package that supplements or replaces an earlier one, the operator passes the kind of update and the original package identifier (e.g. `--status replacement --updates <original-package-id>`). The tool then reuses the original identifier as the package identifier, so the archive can match the update to the package it already holds.

Because the submitting organization comes from configuration, an input folder alone does not determine the package. The generated METS records the values used, so audit the output, not the input.

## 7. Mapping to the SIP (informative, for specialists)

| input | E-ARK SIP location |
|---|---|
| representation folders (or the flat single-representation case) | `representations/<name>/data/`, METS fileSec + structMap |
| `representations.csv` `label` / `type` | representation METS `mets/@LABEL`; in the eark profiles the type lands in `TYPE="Other"`+`csip:OTHERTYPE` and `CONTENTINFORMATIONTYPE="OTHER"`+`csip:OTHERCONTENTINFORMATIONTYPE` ([ADR-0013](decisions/0013-representation-type-from-label.md)) |
| file order (stable, no semantics) | document order within the representation structMap; METS `ORDER` attributes are the real sequencing mechanism, deferred with the manifest (§8) |
| `documentation/` (package and representation level) | `documentation/` folders, conformant per CSIPSTR16; METS fileSec `USE="DOCUMENTATION"` |
| `description.csv` keys under `basic` | the Meemoo table's elements, mapped as `dcterms:*` (`identifier`→`dcterms:identifier`, `rightsholder`→`dcterms:rightsHolder`, `ispartof`→`dcterms:isPartOf`, the rest 1:1) and `schema:*` (`artmedium`→`schema:artMedium`, `artform`→`schema:artform`), in `metadata/descriptive/dc+schema.xml`, METS dmdSec |
| `description.csv` keys under `eark` | the unqualified Simple Dublin Core element of the same name (`title`→`<title>`), in `metadata/descriptive/dc.xml`, METS dmdSec |
| `description.csv` keys under `eark-mods` | `identifier`→`mods:identifier type="local"`, `title[lang]`→`mods:titleInfo xml:lang/mods:title`, in `metadata/descriptive/mods.xml`, METS dmdSec `MDTYPE="MODS" MDTYPEVERSION="3.7"` |
| `dc.xml` / `mods.xml` (a supplied document) | copied as it is to `metadata/descriptive/` (or `representations/<name>/metadata/descriptive/`), checksum computed on the copy, METS dmdSec typed by the profile (`DC` / `MODS`) exactly as a generated document |
| `representations/<name>/description.csv` (or `dc.xml` / `mods.xml`) | `representations/<name>/metadata/descriptive/*.xml`, dmdSec of that representation's METS (CSIPSTR12/13) |
| `[lang]` suffixes | `xml:lang` attributes |
| `SIP_SUBMITTER_NAME`, `SIP_SUBMITTER_OR_ID` | METS `metsHdr/agent ROLE="CREATOR" TYPE="ORGANIZATION"` with the name; under `basic` the OR-id as its `note NOTETYPE="IDENTIFICATIONCODE"` |
| content category | METS `@TYPE` (CSIP vocabulary) |
| `--status` | METS `metsHdr/@RECORDSTATUS` (SIP3 vocabulary: NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE); omitted without the flag |
| `--updates <id>` | package identifier `mets/@OBJID` reuses the original package's identifier (the E-ARK SIP spec defines no separate prior-AIP pointer) |
| `premis/` files | copied under `metadata/preservation/` (package or representation level), referenced from METS amdSec/digiprovMD |
| computed checksums, sizes; formats from `siegfried.json` | METS fileSec + generated PREMIS fixity/format |

## 8. Deferred to a later version

Recorded so they are chosen against, not forgotten:

- An optional explicit manifest (per-file roles, exclusions, custom ordering, per-file labels) for curator-style workflows.
- Prefixed vocabulary keys (`dcterms:*`, `schema:*`) beyond the table (withdrawn 2026-08-20, [ADR-0011](decisions/0011-closed-descriptive-vocabulary.md): every Meemoo-legal flat element has a plain key, and an open vocabulary lets an operator build packages Meemoo rejects).
- Operator-supplied descriptive XML under `basic`, and with it structured schema.org values (`schema:creator` with roles, `schema:isPartOf` variants) that `key,value` cannot express: Meemoo's document carries the package identifier the tool mints, so a supplied one would need the tool to edit XML. Under the eark profiles a supplied `dc.xml` or `mods.xml` has been current since 2026-10-01 (§3, [ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)).
- Describing multiple intellectual entities / hierarchies in one package.
- Accepting a BagIt bag as input (fixity from `manifest-sha256.txt`).
- Per-package overrides of configured administrative values.
- The archival creator (`metsHdr/agent ROLE="ARCHIVIST"`), contact persons, and a submission agreement reference (`altRecordID TYPE="SUBMISSIONAGREEMENT"`, SIP5). [ADR-0010](decisions/0010-config-over-self-describing-input.md) places them in configuration; the tool does not read or write them yet.
