# SIP Creator input specification

This document describes how to prepare a folder so that the SIP Creator CLI can turn it into an E-ARK submission package. It is the contract the `check` and `create` commands enforce, written for the people preparing material. [§7](#7-mapping-to-the-sip-informative-for-specialists) shows specialists where each part of the folder lands in the package. A program using the library builds the same package from its own data, without a folder (see the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary)).

The key words MUST, SHOULD and MAY are to be interpreted as in RFC 2119. The tool enforces every MUST; a SHOULD is advice it does not check.

## What you prepare

One folder is one package. In the simplest case:

```
example-0001/
├── description.csv       ← describes the content (the only file you write)
└── ... your files ...
```

With several versions of the content and extras:

```
example-0001/
├── description.csv
├── representations.csv   ← optional: a label and type per representation
├── siegfried.json        ← optional: a format report
├── representations/
│   ├── master/           ← the archival scans, any structure you like
│   └── access/           ← e.g. a PDF version
│       └── description.csv  ← optional: describes just this version (e.g. its license)
├── documentation/        ← optional: scan reports, context material
└── premis/               ← optional: preservation XML received from a vendor
```

Two things do not live in the folder. The submitting organization comes from the tool's configuration (§6). The profile is passed to `check` and `create` with `--profile`, and decides which keys `description.csv` takes and whether a finished `dc.xml` or `mods.xml` may stand in for it (§3).

This document holds the folder rules. The rules on the package itself that differ per profile, such as which representations it may hold, are written per profile: [`ugent/basic`](profiles/ugent-basic.md), [`ugent/bibliographic`](profiles/ugent-bibliographic.md), and for `meemoo/basic` the [Meemoo SIP 1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) basic content profile.

## 1. General rules

- One input folder MUST correspond to one package.
- These names are reserved at the top level:
  - under every profile: `description.csv` (§3), `representations/`, `representations.csv` and `siegfried.json` (§2), `documentation/` (§4) and `premis/` (§5);
  - under `ugent/basic`: `dc.xml`; under `ugent/bibliographic`: `mods.xml` (§3).

  All other names are free, with any nesting, and are content. That includes `metadata.csv`, `dcschema.csv` and `dc.csv`, and a document name the profile does not take, such as `dc.xml` under `meemoo/basic` or `mods.xml` under `ugent/basic`.
- Operating-system files (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `._*`) MUST be ignored: never packaged, never reported.
- A symbolic link anywhere in the input MUST be an error.
- A file or folder name MAY hold any character XML can carry, `&`, `%`, `+` and spaces included. A name that is not valid UTF-8, or that holds a control character other than tab, line feed and carriage return, MUST be an error: the package's METS and PREMIS documents cannot carry it, and escaping would change the name.
- The tool MUST compare paths after Unicode normalization (NFC), because macOS file names and typed CSV values often differ only in normalization form.
- The tool MUST refuse to build when a MUST rule is broken, and MUST report every violation at once, in plain language, naming the file or folder concerned.
- The tool MUST offer a check-only mode, `check`, that validates a folder against these rules without building. It takes the same `--profile` as a build and reads no configuration. It reports every problem, then counts what the folder holds. The check that judges a document's kind, the root element of received PREMIS (§5), runs only when building.

## 2. Content files and representations

A *representation* is one version of the content: the archival master scans are one, a derived PDF is another. Every package has at least one.

- **Simple case:** without a `representations/` folder, everything in the package folder apart from the reserved names is the content of one representation, named after the input folder.
- **Several versions:** with a `representations/` folder, each folder directly inside it is one representation, named after that folder. All content MUST then be inside `representations/`; a content file elsewhere at the top level is an error. Under `meemoo/basic` there MUST be exactly one representation, as Meemoo SIP 1.2's basic profile requires ("The IE MUST be represented by exactly one representation."). The UGent profiles set no limit.
- A representation's name MUST consist of `A–Z a–z 0–9 . _ -` only. It becomes the representation's directory under `representations/` in the package, and, unless `representations.csv` says otherwise, its label and type. Neither E-ARK CSIP nor the Meemoo specification prescribes names; CSIP requires only that they are unique, which folder names are.
- Inside a representation folder, `description.csv`, `documentation/` and `premis/` are reserved (§3–5), and under the UGent profiles the profile's document name (§3). Everything else is content, with free names and nesting.
- Files are packaged in alphabetical order by path. The order carries no meaning in E-ARK CSIP or the Meemoo specification. If a reading order matters, zero-pad your numbering (`0001.tiff`, `0002.tiff`); explicit ordering is deferred (§8).
- The tool computes checksums and sizes itself; you never supply them.
- Every file keeps its modification time in the package and in the zip. Copy files into the input folder in a way that keeps it (`cp -p`, `rsync -t`, or a download that sets it), or the package carries the time of that copy instead.

### `representations.csv`: labels and types (optional)

A folder name makes a good machine name but not always a good display name. `representations.csv`, next to `description.csv`, gives each representation a label (its display name in the package) and a type (what an ingest system such as RODA shows as its kind):

```csv
folder,label,type
master,Master scan (TIFF),archival
access,Access copy (PDF),access
```

- The file MUST be UTF-8 with a header row. The columns are `folder` (required), `label` and `type` (optional), in any order; header names are matched case-insensitively, so a spreadsheet's `Folder` works. An unknown or repeated column MUST be an error. A UTF-8 BOM, CRLF line endings and RFC 4180 quoting are accepted.
- The file MUST have at least one row, and it requires a `representations/` folder: in the simple case it MUST be an error.
- `folder` names a folder directly under `representations/` by its name alone (`master`, not a path). Every row MUST match an existing folder, no two rows may name the same folder, and every folder MUST have a row. A folder without a row is an error, never an exclusion, so no content can silently drop out of the package. To leave material out, move it out of the input folder.
- An empty `label` means the folder name; an empty `type` means the label.
- `label` and `type` may hold any text, `&` and quotes included: the tool escapes them when it writes the package's XML. A control character other than tab, line feed and carriage return MUST be an error.
- The rows' order is the representations' order in the package.
- The label is used under every profile. The type is used only under the UGent profiles; under `meemoo/basic` it has no effect (§7).

### `siegfried.json`: format report (optional)

The tool adds each content file's format (a PRONOM identifier) from a Siegfried report at the top level. It never runs Siegfried itself and never takes format information from you directly. Generate the report from the input folder, capturing it before writing so `sf` does not read its own half-written output:

```sh
cd ./your-input && report="$(sf -hash md5 -json .)" && printf '%s\n' "$report" > siegfried.json
```

- Without the report, the package carries no format information.
- With it, the build MUST stop when the report is malformed or made without `-hash md5`, when a content file has no entry, or when Siegfried reported an error for a content file.
- The report's MD5 is the checksum the package declares for each file the report covers. The tool takes it as given and does not read the file to check it ([ADR-0032](decisions/0032-the-report-checksum-is-taken-as-given.md)). The report SHOULD therefore be generated right before the build, from the folder as it will be built: a file changed after the report gets a checksum that does not match its bytes, and an archive that checks fixity rejects the package.
- `check` reports a malformed report and every content file without an entry.
- An entry without a match leaves that file without a format.
- Documentation files need no entry; when they have one, its MD5 is the file's checksum, as for content.
- The report may come from another operating system: backslashes in its paths are read as folder separators. A file whose name contains a backslash therefore cannot be matched: a content file is then an error, a documentation file goes without a format.

## 3. Descriptive metadata: `description.csv`, or a supplied document

`description.csv` describes what the package contains. Usually it is the only file you write. Under the UGent profiles a finished document may take its place (see [Supplying a finished document](#supplying-a-finished-document-ugentbasic-and-ugentbibliographic)). At the top level, exactly one of `description.csv` and the profile's document MUST be present.

### The file

- A two-column CSV with a `key,value` header row, in UTF-8. A UTF-8 BOM, CRLF line endings (spreadsheet tools produce both) and RFC 4180 quoting MUST be accepted.
- Keys are matched case-insensitively (`Title` reads as `title`). An unknown key MUST be an error, so a typo cannot silently drop metadata.
- A value MAY hold any text, `&`, `<` and quotes included. A control character other than tab, line feed and carriage return MUST be an error, because XML cannot carry it.
- Add a language tag in square brackets where the language matters: `title[nl]`, `description[en]`.
- Repeat a key for more values (two `creator` rows for two creators) where the profile allows it, as listed below. A key that may repeat *per language* takes one row per language tag: `title[nl]` and `title[en]` is fine, two `title[nl]` rows are not. A repeat the profile does not allow MUST be an error.
- The required keys of the profile MUST be present and non-empty at the top level. `identifier` is your local catalog or inventory number; it travels with the package as its local identifier.

### Keys per profile

| profile | keys | required | document that may stand in |
|---|---|---|---|
| `meemoo/basic` | Meemoo's keys, in the table below | `identifier`, `title`, `description`, `created` | none |
| `ugent/basic` | the fifteen Simple Dublin Core elements | `identifier`, `title` | `dc.xml` |
| `ugent/bibliographic` | `identifier`, `title` | `identifier`, `title` | `mods.xml` |

Each profile has its own keys: a Meemoo key under `ugent/basic` is an unknown key, and an error.

**`meemoo/basic`** follows the elements of Meemoo's basic content profile that fit in a key and a value. Wherever a language-tagged key is used, a Dutch entry (`[nl]`) MUST be among its rows, as Meemoo requires; other languages may be added.

| key | element | meaning | repeatable |
|---|---|---|---|
| `identifier` | `dcterms:identifier` | local catalog or inventory number (required) | no |
| `title` | `dcterms:title` | title of the work (required) | per language |
| `description` | `dcterms:description` | free-text description (required) | per language |
| `created` | `dcterms:created` | creation date of the original, a year or an ISO date (required) | no |
| `alternative` | `dcterms:alternative` | alternative title | yes |
| `abstract` | `dcterms:abstract` | summary or abstract | per language |
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
| `rights` | `dcterms:rights` | rights statement | per language |
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

**`ugent/basic`** takes the fifteen Simple Dublin Core elements: `title`, `creator`, `subject`, `description`, `publisher`, `contributor`, `date`, `type`, `format`, `identifier`, `source`, `language`, `relation`, `coverage`, `rights`. Every key is repeatable. A language tag is accepted but not written into the document.

**`ugent/bibliographic`** takes two keys: `identifier`, once and without a language tag, and `title`, once per language. Anything richer, such as the library's physical copies of the work, needs a supplied `mods.xml`.

### Describing one representation

Under the UGent profiles a representation MAY have its own `description.csv`, at `representations/<name>/description.csv`, for what is true of that version only, such as a license that differs between the master and an access copy. The rules are those of the top-level file, with two differences:

- No key is required: the package-level description covers the work's identity. A `title` MAY still name the version (e.g. "PDF-versie").
- It describes the representation, not the work: `created` or `creator` here refer to the making of this version.

The profile's document MAY take its place (`representations/<name>/dc.xml` or `mods.xml`), never both. In the simple case without `representations/` there is no place for either.

Under `meemoo/basic` a representation MUST NOT have a description, as Meemoo SIP 1.2's basic profile requires ("There MUST NOT be any descriptive metadata at the representation level.").

### Supplying a finished document (`ugent/basic` and `ugent/bibliographic`)

A record that already exists as a document, or one richer than the rows can hold (a MODS record with its physical copies, names with roles), can be supplied as a file: `dc.xml` under `ugent/basic`, `mods.xml` under `ugent/bibliographic`, at the top level or in a representation folder. The tool copies it into the package as it is and references it from the METS like a generated document.

- At each level the document takes the place of `description.csv`: one or the other, never both.
- The file MUST be well-formed XML with the profile's root element:
  - under `ugent/basic`: `simpledc` without namespace, the shape the tool itself writes. A Dublin Core export in another wrapper, such as `oai_dc:dc`, must be rewrapped. The elements inside SHOULD carry no namespace either (`<title>`, not `<dc:title>`): that is the form RODA reads, and the only one the schema the package ships (`simpledc.xsd`) accepts.
  - under `ugent/bibliographic`: `mods:mods` in the MODS v3 namespace (`http://www.loc.gov/mods/v3`) with `version="3.7"`, the version the package's METS declares.
- Nothing else in the document is checked: not its validity against the schema, and not whether it has an identifier or a title. Schema validity is the producer's responsibility; the validators downstream check it.
- The file name must be the profile's: a `mods.xml` under `ugent/basic` is content, not a document.
- `meemoo/basic` takes no document, because Meemoo's document must carry the package identifier the tool mints and the tool does not edit XML. A `dc+schema.xml`, at the top level or in a representation folder, MUST NOT be present under `meemoo/basic`; `check` and `create` report it.

## 4. Documentation

Context material that is not itself the preserved content: scan reports, correspondence, finding aids.

- Files under `documentation/` at the top level document the whole package.
- Files under `representations/<name>/documentation/` document that representation.
- Free naming and nesting inside.

Documentation is optional. E-ARK CSIP recommends it (CSIPSTR16, a SHOULD), and commons-ip warns when a representation has no `documentation/` folder; the package still validates.

## 5. Received preservation files (`premis/`)

Digitization vendors and lab equipment sometimes deliver preservation metadata as PREMIS XML, such as events like "scanned on this device, on this date". You never write these files yourself. If you received them, put them in:

- `premis/` at the top level, about the whole package, or
- `representations/<name>/premis/`, about one representation.

Rules:

- Each file MUST be well-formed XML whose root is a `premis:premis` element in the PREMIS 3 namespace, and SHOULD be valid PREMIS 3.0. The files are copied into the package as received: not parsed, edited or merged.
- `premis.xml` is reserved for the document the tool generates and MUST NOT be used as a file name here.
- `check` confirms that each file is well-formed XML. The `premis:premis` root is confirmed by `create`, which refuses the folder otherwise.
- The files cannot know the identifiers the tool generates, so they SHOULD identify their subject with local identifiers built from your `identifier` and the representation name (e.g. `example-0001-master`), so a future reader can match them with the generated preservation metadata.

## 6. What comes from configuration and the command line

These values span many packages or belong to the run, so they do not live in the folder. The environment variables are listed in [CONFIG.md](../CONFIG.md).

| value | source |
|---|---|
| submitting organization: name | `SIP_SUBMITTER_NAME`, required by `create` under every profile |
| submitting organization: Meemoo OR-id | `SIP_SUBMITTER_OR_ID`, required by `create` under `meemoo/basic` |
| content category (e.g. `Photographs – Digital`) | `--content-category`, else `SIP_CONTENT_CATEGORY`, else the profile's value |
| profile | `--profile`, required by `check` and `create` |
| record status | `--status` |
| identifier of the package this one updates | `--updates` |

- Without `--status` the package carries no record status, which the E-ARK SIP specification reads as new.
- To submit a package that supplements or replaces an earlier one, pass the kind of update and the original package identifier (e.g. `--status replacement --updates <original-package-id>`). The tool reuses the original identifier as the package identifier, so the archive can match the update to the package it holds.

Because the submitting organization comes from configuration, the folder alone does not determine the package. The generated METS records the values used: audit the output, not the input.

## 7. Mapping to the SIP (informative, for specialists)

| input | E-ARK SIP location |
|---|---|
| representation folders (or the simple case) | `representations/<name>/data/`, METS fileSec and structMap |
| `representations.csv` `label` / `type` | representation METS `mets/@LABEL`; under the UGent profiles the type in `TYPE="Other"` + `csip:OTHERTYPE` and `CONTENTINFORMATIONTYPE="OTHER"` + `csip:OTHERCONTENTINFORMATIONTYPE` ([ADR-0013](decisions/0013-representation-type-from-label.md)); under `meemoo/basic` the content typing is fixed to Meemoo's profile URI |
| file order (no meaning) | document order in the representation's structMap; METS `ORDER` attributes, the real sequencing mechanism, are deferred with the manifest (§8) |
| `documentation/` (package and representation) | `documentation/` directories (CSIPSTR16), METS fileSec `USE="Documentation"` |
| `description.csv` under `meemoo/basic` | the table's elements, `dcterms:*` and `schema:*` (e.g. `ispartof` → `dcterms:isPartOf`, `artmedium` → `schema:artMedium`), in `metadata/descriptive/dc+schema.xml`, METS dmdSec ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)) |
| `description.csv` under `ugent/basic` | the unqualified Simple Dublin Core element of the same name (`title` → `<title>`), in `metadata/descriptive/dc.xml`, METS dmdSec |
| `description.csv` under `ugent/bibliographic` | `identifier` → `mods:identifier type="local"`, `title[lang]` → `mods:titleInfo xml:lang/mods:title`, in `metadata/descriptive/mods.xml`, METS dmdSec `MDTYPE="MODS" MDTYPEVERSION="3.7"` ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)) |
| `dc.xml` / `mods.xml` (supplied) | copied as it is to `metadata/descriptive/` (or the representation's), checksum computed on the copy, METS dmdSec typed as for a generated document ([ADR-0021](decisions/0021-descriptive-model-follows-its-standard.md)) |
| a representation's description | `representations/<name>/metadata/descriptive/`, dmdSec of that representation's METS (CSIPSTR12, CSIPSTR13) |
| `[lang]` suffixes | `xml:lang` attributes |
| `SIP_SUBMITTER_NAME`, `SIP_SUBMITTER_OR_ID` | METS `metsHdr/agent ROLE="CREATOR" TYPE="ORGANIZATION"` with the name; under `meemoo/basic` the OR-id as its `note NOTETYPE="IDENTIFICATIONCODE"` |
| content category | METS `@TYPE` (CSIP vocabulary) |
| `--status` | METS `metsHdr/@RECORDSTATUS` (SIP3 vocabulary: NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE); omitted without the flag |
| `--updates <id>` | `mets/@OBJID` reuses the original package's identifier; the E-ARK SIP specification defines no separate pointer to the earlier package |
| `premis/` files | `metadata/preservation/` (package or representation), referenced from METS amdSec/digiprovMD |
| checksums and sizes; formats from `siegfried.json` | METS fileSec, and under `meemoo/basic` PREMIS fixity and format ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)) |

## 8. Not supported yet

- An optional manifest listing files with roles, exclusions, custom order and labels, for curator workflows.
- A supplied descriptive document under `meemoo/basic`, and with it structured schema.org values (`schema:creator` with roles, `schema:isPartOf` variants) that `key,value` cannot express. Meemoo's document carries the package identifier the tool mints, so a supplied one would need the tool to edit XML.
- Several intellectual entities, or a hierarchy of them, in one package.
- A BagIt bag as input, with fixity taken from its manifest.
- The archival creator (`metsHdr/agent ROLE="ARCHIVIST"`), contact persons, and a submission agreement reference (`altRecordID TYPE="SUBMISSIONAGREEMENT"`, SIP5). [ADR-0010](decisions/0010-config-over-self-describing-input.md) places them in configuration; the tool does not read or write them yet.

Not planned: keys with a prefix (`dcterms:*`, `schema:*`) beyond the table ([ADR-0011](decisions/0011-closed-descriptive-vocabulary.md)). Every element Meemoo allows in flat form has a plain key, and an open vocabulary would let an operator build packages Meemoo rejects.
