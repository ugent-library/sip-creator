# SIP Creator input specification

Status: **Draft** — not implemented. Current code still expects the `dc+schema.json` / `representation_N/` layout described in [sip-creator-design.md](sip-creator-design.md).

This document describes how to prepare a folder so that the SIP Creator **CLI** can turn it into an E-ARK submission package. It is written for the people preparing material; the section [Mapping to the SIP](#7-mapping-to-the-sip-informative-for-specialists) at the end is for specialists and explains how each rule lands in the E-ARK CSIP/SIP structure.

The key words MUST, SHOULD and MAY are to be interpreted as in RFC 2119.

## Scope and architecture

This spec describes the CLI's input convention only. The underlying library exposes a programmatic builder API (domain model in `sip/`); this input format is one frontend that maps onto it. Larger systems that automate ingest workflows construct the same model directly from their own stores without using this input format — see the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc. Nothing in this spec is required to use the library.

## What you prepare

One folder = one package. In the simplest case:

```
fotoalbum-gent-1913/
├── metadata.csv          ← describes the content (the only file you write)
└── ... your files ...
```

With multiple versions of the content and extras:

```
fotoalbum-gent-1913/
├── metadata.csv
├── representations/
│   ├── master/           ← the archival scans, any structure you like
│   └── access/           ← e.g. a PDF version
│       └── metadata.csv  ← optional: describes just this version (e.g. its license)
├── documentation/        ← optional: scan reports, context material
└── premis/               ← optional: preservation XML received from a vendor
```

Institution details (who submits, who archives, contact person, agreement number) do **not** live in the folder — they rarely change and come from the tool's configuration. See [What comes from configuration](#6-what-comes-from-configuration-and-the-command-line).

## 1. General rules

- One input folder MUST correspond to one package.
- Five names at the top level are reserved: `metadata.csv` (required), `representations/`, `documentation/`, `premis/`, and `siegfried.json` (each optional; `siegfried.json` is the pre-computed characterization report, see §2). All other folder and file names are free, with any nesting.
- Operating-system artifacts (`.DS_Store`, `Thumbs.db`, `desktop.ini`, `._*`) MUST be ignored by the tool: never packaged, never warned about.
- Symbolic links anywhere in the input MUST be an error.
- The tool MUST compare paths after Unicode canonical normalization (NFC) — macOS file names and typed CSV values often differ only in normalization form.
- The tool MUST refuse to build when any MUST rule is violated, and MUST report all violations at once, in plain language, naming the file or folder concerned. SHOULD violations produce warnings.
- The tool MUST offer a check-only mode that validates a folder against every rule here without building anything.

## 2. Content files and representations

A *representation* is one version of the content: the archival master scans are one, a derived PDF is another. Every package has at least one.

- **Simple case:** if there is no `representations/` folder, everything in the package folder (apart from the reserved names) is the content of a single representation.
- **Multiple versions:** if `representations/` exists, each folder directly inside it is one representation, named by its folder name. All content MUST then live inside `representations/` — content files elsewhere at the top level are an error (except inside `documentation/` and `premis/`).
- Representation folder names MUST match `A–Z a–z 0–9 . _ -`. They are labels: the tool decides the folder naming inside the final package (e.g. meemoo's `representation_1`) and keeps your label as the human-readable name.
- Inside a representation folder, three names are reserved: `metadata.csv`, `documentation/` and `premis/` (all optional, see §3–5). Everything else is content, with free naming and nesting.
- Files are packaged in a stable, tool-determined order (alphabetical by path). This order carries no meaning: neither E-ARK CSIP nor the meemoo specification assigns semantics to file order. If a human-readable sequence matters to you, zero-pad your numbering (`0001.tiff`, `0002.tiff`, …); explicit ordering is a deferred feature (see §8, the manifest).
- The tool computes checksums and sizes itself; you never supply those. File formats come from an optional pre-computed characterization report (`siegfried.json` at the top level, generated from the input root with `sf -hash md5 -json`) that the tool verifies against the files before trusting — you never hand-author format info ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)).

## 3. `metadata.csv` — describing the content

A two-column CSV (`key,value`) describing what the package contains. This is the only file an operator writes.

- MUST be UTF-8 with a `key,value` header row. The tool MUST accept a UTF-8 BOM and CRLF line endings (spreadsheet tools produce both) and RFC 4180 quoting.
- `identifier` and `title` MUST be present and non-empty. The identifier is your local catalog or inventory number; it travels with the package as its local identifier.
- Repeat a key for multiple values (two `creator` lines for two creators).
- Add a language tag in square brackets where the language matters: `title[nl]`, `description[en]`.
- Unknown keys MUST be an error — a typo must not silently drop metadata.

Supported keys (plain names; the specialist mapping is in [§7](#7-mapping-to-the-sip-informative-for-specialists)):

| key | meaning |
|---|---|
| `identifier` | local catalog/inventory number (required) |
| `title` | title of the work (required) |
| `description` | free-text description |
| `created` | creation date of the original (year or ISO date) |
| `creator` | maker of the work (photographer, author, artist) |
| `contributor` | other contributors |
| `publisher` | publisher |
| `subject` | subject keyword |
| `spatial` | place depicted or covered |
| `extent` | extent (e.g. "48 foto's") |
| `language` | language of the content |
| `type` | kind of work |
| `ispartof` | collection or series this belongs to |
| `license` | license on the content |
| `rights` | rights statement |
| `rightsholder` | rights holder |

Prefixed keys from the supported vocabularies (`dcterms:*`, `schema:*`) MAY be used for anything the plain names don't cover; they map one-to-one onto the generated Dublin Core.

Example:

```csv
key,value
identifier,BIB.FA.2026.001
title[nl],Fotoalbum Gent 1913
description[nl],Album met 48 zwart-witfoto's van de Gentse binnenstad
created,1913
creator,Edmond Sacré
subject[nl],stadsgezichten
subject[nl],wereldtentoonstellingen
spatial[nl],Gent
extent[nl],48 foto's
rights[nl],publiek domein
```

### Describing one representation

A representation MAY carry its own `metadata.csv` (at `representations/<name>/metadata.csv`) when something is true of that version only — typically a license or rights statement that differs between the master and an access copy. Same format and rules as the package-level file, with two differences:

- `identifier` and `title` are NOT required — the package-level description covers the work's identity. A `title` MAY still be given as a human-readable name for the version (e.g. "PDF-versie").
- It describes the representation, not the work: keys like `created` or `creator` here refer to the making of this version.

In the simple case without a `representations/` folder there is no place for this file — by design; the simple case stays simple.

## 4. Documentation

Context material that is not itself the preserved content: scan reports, correspondence, finding aids.

- Files under `documentation/` at the top level document the whole package.
- Files under `representations/<name>/documentation/` document that representation.
- Free naming and nesting inside.

## 5. Received preservation files (`premis/`)

Digitization vendors and lab equipment sometimes deliver preservation metadata as PREMIS XML (events such as "scanned on this device, on this date"). You never write these files yourself — if you received them, put them in:

- `premis/` at the top level (about the whole package), or
- `representations/<name>/premis/` (about one representation).

Rules:

- Files here MUST be valid PREMIS 3.0 XML. They are included in the package as received — not parsed, edited, or merged.
- Because these files cannot know the identifiers the tool generates, they SHOULD identify their subject using local identifiers built from your `identifier` and the representation name (e.g. `BIB.FA.2026.001-master`), so a future reader can correlate them with the generated preservation metadata.

## 6. What comes from configuration and the command line

These values span many packages and rarely change, so they live in the tool's configuration rather than in each package folder:

| value | source |
|---|---|
| submitting organization (name, identifier) | configuration |
| archival creator organization | configuration |
| contact person(s) (name, email) | configuration |
| submission agreement reference | configuration |
| target profile (e.g. meemoo basic) | configuration, overridable per run |
| content category (e.g. photographs) | configuration, overridable per run |

Creating vs. updating:

- By default every package is **new**.
- To submit a package that supplements or replaces an earlier one, the operator passes the original package identifier and the kind of update on the command line (e.g. `--status replacement --updates <original-package-id>`). The tool then reuses the original identifier as the package identifier — that is how a conformant archive matches the update to the existing holdings.

Deliberate trade-off: because organization details come from configuration, an input folder alone does not fully determine the package. The generated package itself records which values were used; audit the output, not the input. If this tool ever serves multiple submitting organizations from one installation, this decision must be revisited.

## 7. Mapping to the SIP (informative, for specialists)

| input | E-ARK SIP location |
|---|---|
| representation folders (or the flat single-representation case) | `representations/<name>/data/`, METS fileSec + structMap |
| file order (stable, no semantics) | document order within the representation structMap — METS `ORDER` attributes are the real sequencing mechanism, deferred with the manifest (§8) |
| `documentation/` (package and representation level) | `documentation/` folders, conformant per CSIPSTR16; METS fileSec `USE="DOCUMENTATION"` |
| `metadata.csv` plain keys | `dcterms:*` elements (`identifier`→`dcterms:identifier`, `rightsholder`→`dcterms:rightsHolder`, `ispartof`→`dcterms:isPartOf`, the rest 1:1) in `metadata/descriptive/*.xml`, METS dmdSec |
| `representations/<name>/metadata.csv` | `representations/<name>/metadata/descriptive/*.xml`, dmdSec of that representation's METS (CSIPSTR12/13) |
| `[lang]` suffixes | `xml:lang` attributes |
| configuration: organizations, contacts | METS `metsHdr/agent` (`ROLE=CREATOR TYPE=ORGANIZATION` submitter; `ROLE=ARCHIVIST TYPE=ORGANIZATION` archival creator; individuals as contact agents) |
| configuration: submission agreement | METS `altRecordID TYPE="SUBMISSIONAGREEMENT"` (SIP5) |
| configuration: content category | METS `@TYPE` (CSIP vocabulary) |
| `--status` | METS `metsHdr/@RECORDSTATUS` (SIP3 vocabulary: NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE; default NEW) |
| `--updates <id>` | package identifier `mets/@OBJID` reuses the original package's identifier — the E-ARK SIP spec defines no separate prior-AIP pointer |
| `premis/` files | copied under `metadata/preservation/` (package or representation level), referenced from METS amdSec/digiprovMD |
| computed checksums, sizes; formats from `siegfried.json` | METS fileSec + generated PREMIS fixity/format |

## 8. Deferred to a later version

Recorded so they are chosen against, not forgotten:

- An optional explicit manifest (per-file roles, exclusions, custom ordering, per-file labels) for curator-style workflows.
- Operator-supplied descriptive XML (MODS/EAD-shaped material) with the validation rules it requires.
- Describing multiple intellectual entities / hierarchies in one package.
- Accepting a BagIt bag as input (fixity from `manifest-sha256.txt`).
- Per-package overrides of configured administrative values.
