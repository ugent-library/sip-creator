# Profile `ugent/basic`

*Content type `ugent/basic`, defined by UGent Library. Written 2026-10-07 with the
[ugent-profiles plan](../archive/ugent-profiles.md). The decision is
[ADR-0034](../decisions/0034-a-profile-is-a-content-type.md).*

The key words MUST, SHOULD and MAY are to be interpreted as in RFC 2119. Each MUST says
who holds it: **checked** (`check` and `create` refuse a package that breaks it, and so
does a program that builds through the library), **written** (the tool writes the value
itself, so no input can break it), or **operator** (the tool cannot see it; the person
preparing the package is responsible). A SHOULD is advice the tool does not check.

## 1. Scope

`ugent/basic` packages digital-born or digitised resources that UGent Library has not
necessarily accessioned or catalogued but that still need to be preserved in its RODA
instance, such as an internal database dump or a deposited set of files. A short Simple Dublin Core record describes what the package holds.

A resource the library has catalogued as its holdings is packaged under
[`ugent/bibliographic`](ugent-bibliographic.md) instead.

Meemoo SIP 1.2 also has a profile called basic. It means something else there: one media
file in one representation, described in Meemoo's `dc+schema.xml`. A package for Meemoo
uses `meemoo/basic`, never this profile.

## 2. Declaration

A package conforms to E-ARK CSIP and the E-ARK SIP 2.2.0 specification, with this
profile on top.

| METS attribute (package METS) | value |
|---|---|
| `mets/@PROFILE` | `https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml` |
| `mets/@TYPE` | `Mixed`, unless the operator gives another value from the CSIP content category vocabulary with `--content-category` |
| `mets/@csip:CONTENTINFORMATIONTYPE` | `OTHER` |
| `mets/@csip:OTHERCONTENTINFORMATIONTYPE` | `ugent/basic` |

- The package METS MUST declare `ugent/basic` as its content information type. **Written.**
- A representation METS declares its representation's type instead (§4,
  [ADR-0013](../decisions/0013-representation-type-from-label.md)).

The value carries no version. When a change to these rules would refuse a package that
was ingested earlier, the value gains one.

## 3. Descriptive metadata

- A package MUST carry a description at package level, in Simple Dublin Core, as
  `metadata/descriptive/dc.xml`. **Checked.**
- The description MUST state an identifier and a title. **Checked** when the description
  comes from `description.csv` or from a program through the library; **operator** when
  the operator supplies a finished `dc.xml`, because the tool checks only its root
  element.
- The identifier is the producer's own: an inventory number, a deposit number, the name
  of a database. The package's `mets/@OBJID` is a UUID the tool mints; the description
  keeps the producer's identifier
  ([ADR-0012](../decisions/0012-eark-keeps-producer-identifier.md)).
- A supplied `dc.xml` MUST have a `simpledc` root without namespace. **Checked.** Its
  elements SHOULD carry no namespace either, and it SHOULD be valid against the
  `simpledc.xsd` the package ships.
- A representation MAY carry its own description, in the same standard, for what is true
  of that representation only, such as a license.

The keys `description.csv` takes are in the [input specification](../input-spec.md) §3.

## 4. Representations

A package holds zero or more representations. Each is one copy of the intellectual
entity, named by its kind:

- `preservation` is the copy as delivered: the raw data, the born-digital file as
  deposited, the files of a digitisation as the scanning produced them. Anything that
  comes from outside, such as born-digital material from an estate, is a preservation
  copy.
- `archival` is a copy in a format suitable for long-term archiving, derived from the
  delivered copy when that is not in one: a conversion to PDF/A, for instance.
- `access` is the copy that will be disseminated.

Rules:

- A representation's name MUST be one of `preservation`, `archival` and `access`,
  exactly, in lowercase. **Checked.**
- A package MUST NOT hold two representations of the same name. **Checked**, because a
  name is a folder name.
- A representation's type MUST equal its name. A `representations.csv` row MAY leave
  `type` empty, and MUST NOT give another value. **Checked.** The label is free.
- A package MAY hold no representation at all: a description of an intellectual entity
  whose content is not, or not yet, in the archive. This holds for a new package and for
  an update (§8). **Checked**: the tool accepts it.
- A representation MUST hold at least one file. **Checked.**
- Inside a representation, folders and file names are free.

The name becomes the representation's directory under `representations/` in the package,
its METS `OBJID`, and its type in `csip:OTHERCONTENTINFORMATIONTYPE` on the representation
METS.

## 5. Files

- Any file format is allowed in any representation. This profile states no format list.
- Every file in the package carries an MD5 checksum and its size in the METS. **Checked**:
  the tool computes them, or takes the MD5 from a Siegfried report
  ([ADR-0032](../decisions/0032-the-report-checksum-is-taken-as-given.md)).
- A file's format (its PRONOM identifier) is recorded only when the operator supplies a
  Siegfried report.

The input specification §1 and §2 hold the rules for file names and for the report.

## 6. Preservation metadata

- The tool generates no PREMIS for this profile. Without agents or events a generated
  PREMIS document would only repeat the fixity the METS already declares.
- A package MAY carry PREMIS documents the operator received, such as a digitisation
  vendor's events, at package or representation level. Each MUST be well-formed XML with
  a `premis:premis` root in the PREMIS 3 namespace. **Checked.** It SHOULD be valid
  PREMIS 3.0. The tool copies it as received.

## 7. Documentation

- A package MAY carry documentation at package level and per representation: scan
  reports, correspondence, a database's data dictionary.
- E-ARK CSIP recommends a documentation folder (CSIPSTR16); the package is valid without
  one.

## 8. Updates

- A package that updates an earlier one MUST carry the record status of the update and
  the earlier package's identifier, which becomes its `mets/@OBJID`. **Checked**, given
  with `--status` and `--updates`.
- An update MAY hold no representations, to correct the description of a package
  ingested earlier.

## 9. What RODA does with the package

From a reading of RODA's ingest code with commons-ip 2.11.2
([runbook](../development-roda-ingest.md)); the first ingest under this profile confirms
each row.

| part of the package | in RODA |
|---|---|
| `csip:OTHERCONTENTINFORMATIONTYPE` `ugent/basic` | the AIP type, `ugent/basic`, or the label RODA's AIP type vocabulary gives it, which is RODA configuration |
| a representation's name | the representation's id, and its type through the representation METS |
| `dc.xml` | descriptive metadata of type `dc_SimpleDC20021212`: rendered, indexed and editable without further configuration |
| essence and its checksums | the AIP's files; RODA verifies the checksums when it parses the package |
| documentation | the AIP's documentation |
| `schemas/` | the AIP's schemas |
| supplied PREMIS | at package level, agents and events are kept and anything else is dropped; at representation level, not yet checked |
| no generated PREMIS | none expected; RODA records its own ingest events |
