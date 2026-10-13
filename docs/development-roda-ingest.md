# Runbook: ingesting a UGent SIP into RODA

*The field-acceptance procedure for the [ugent/basic](profiles/ugent-basic.md) and
[ugent/bibliographic](profiles/ugent-bibliographic.md) profiles, first written for
`eark/dc` in the [eark-writer plan](archive/eark-writer.md):
produce a package, pre-flight it, ingest it into UGent's RODA instance, and
verify what arrived. Written 2026-07-17 from a desk-check of RODA's ingest
code (`EARKSIP2ToAIPPlugin` / `EARKSIP2ToAIPPluginUtils`, commons-ip2 2.11.2,
the same version our validator pins); the UI steps describe the expected flow
and should be checked against the instance's RODA version on first use.*

## 1. Produce the package

```sh
./bin/sip-creator create --profile ugent/basic <source-dir> <dest-dir>
```

The **zip is the deliverable** (`<dest-dir>/uuid-<uuid>.zip`): RODA ingests
zips directly; no bagging for RODA (bags are Meemoo's envelope, [ADR-0008](decisions/0008-bag-layer-out-of-scope.md)).
Include a `documentation/` folder in the source: it satisfies CSIPSTR16 and
maps into the AIP.

## 2. Pre-flight

```sh
./scripts/validate.sh -s 2.2.0 <dest-dir>/uuid-*.zip
```

Expect `VALID (errors=0 warnings=0)`. Do not submit anything that isn't. RODA
parses the package with commons-ip too, so a validator failure predicts an
ingest failure. RODA v5.7.1 bundles commons-ip 2.8.0, not the 2.11.2 the
validator pins. To see what RODA's version says, run the validator with
`CSIP_CMD`, as CONTRIBUTING.md describes.

## 3. Ingest

In RODA: **Ingest → Transferred Resources** → upload the zip, then create an
ingest job over it selecting the **E-ARK SIP 2** format (the
`EARKSIP2ToAIPPlugin` path). Leave "technical metadata validation" off (the
package carries none) and "create submission" per local policy.

RODA does **not** run the full specification validator on ingest: it parses
the package into its model and accepts or rejects on the parse-level report
(METS readable, referenced files present, checksums matching). Failures surface in
the job report per SIP.

## 4. Verify what arrived

The desk-check of RODA's mapping code, as a checklist. Each row is what the
code does with what our package contains. The rows on identifiers and updates
were read from RODA's `master` at commit `668f61b2` (2026-10-07):

| our package | RODA mapping | verify in the UI |
|---|---|---|
| package METS `csip:CONTENTINFORMATIONTYPE="OTHER"` with `csip:OTHERCONTENTINFORMATIONTYPE` `ugent/basic` or `ugent/bibliographic` | becomes the **AIP type** | AIP shows type `ugent/basic` or `ugent/bibliographic`, or the label RODA's AIP type vocabulary gives it, which is RODA configuration and outside this tool |
| package METS `OBJID` (`uuid-<uuid>`, minted by the tool or given with `--updates`) | kept on the AIP as one of its **SIP IDs** (`ingestSIPIds`). It does not become the AIP id | the AIP's details list the package's `OBJID` as a SIP ID |
| (nothing in our package) | RODA gives the AIP a **random id of its own** at ingest (`createAIP` with no AIP id) | the AIP id differs from the package's `OBJID` |
| `RECORDSTATUS="UPDATE"` with the earlier package's `OBJID` (`--status`, `--updates`) | finds the AIP to update: first the one active or under-appraisal AIP whose SIP IDs hold the `OBJID`, then, if not exactly one matched, the AIP whose own id equals the `OBJID`. If neither search finds exactly one AIP, the update fails | the update changes the AIP the first package created, and its SIP IDs still hold that `OBJID` |
| rep METS `OBJID` (the representation's folder name: `preservation`, `archival` or `access`) | becomes the **representation id** | one representation per folder, with that name |
| rep METS main `div` without `TYPE` | commons-ip takes a representation's status from the `TYPE` of the main `div` in its METS, and reads a missing `TYPE` as `OTHER`. RODA sets a representation's original flag only when that status is `ORIGINAL` | every representation is **not original** (`original: false`). Observed 2026-10-12 and 2026-10-13, see section 6 |
| rep METS `csip:OTHERCONTENTINFORMATIONTYPE`, the folder name capitalized ([ADR-0013](decisions/0013-representation-type-from-label.md)) | becomes the **representation type** (RODA v5.7.0 and later) | each representation shows its type, `Preservation`, `Archival` or `Access`, in the Type column |
| essence in `data/` with METS checksums | files created; fixity verified at parse | files present, sizes right, no checksum complaints in the job report |
| `dc.xml` (`ugent/basic`), `MDTYPE="DC"` + `MDTYPEVERSION="SimpleDC20021212"`, simple-DC shape | recognized descriptive metadata (`dc_SimpleDC20021212`) | title/description **rendered and indexed** (searchable), form-editable; if RODA shows raw XML instead, the typing didn't match |
| `mods.xml` (`ugent/bibliographic`), `MDTYPE="MODS"` + `MDTYPEVERSION="3.7"` | descriptive metadata of type MODS 3.7 | rendered and indexed only if a MODS crosswalk is configured in RODA, which is outside this tool; otherwise shown as XML. With `mods_3.7.xslt` configured, the search index shows the title and the level `item` |
| `documentation/` files of the package | mapped into AIP documentation | visible under the AIP's documentation |
| `documentation/` files of a representation, listed in the Documentation division of the representation's METS | commons-ip 2.8.0 and 2.11.2 attach them to the package, not to the representation, so RODA stores them as **AIP documentation** | visible under the AIP's documentation, not under the representation. A representation file with the same name as a package file would be created twice, which is not tested |
| `schemas/` XSDs | mapped into AIP schemas | present (RODA ignores content) |
| received PREMIS from the input folder's `premis/`, as a `digiprovMD` with `MDTYPE="PREMIS"`. The UGent profiles write no PREMIS of their own | kept as other technical metadata, not parsed into RODA's PREMIS | each file under `metadata/preservation/otherTechMd/` of the AIP or the representation, byte for byte, next to RODA's own ingest events |

## 5. Record the outcome

Whatever happens, write it down under section 6: the date, the profile, the RODA
version, the job report's verdict, and any row of the table in section 4 that did
not hold. A failed row is a finding about either our package or this desk-check,
and both are what this runbook is for. The first clean ingest is the field
acceptance of the [eark-writer plan](archive/eark-writer.md), which shipped on
2026-07-17 without it.

## 6. Outcomes

### 2026-10-12: the examples of ugent/basic and ugent/bibliographic

- **RODA:** v5.7.1, in the image `ugentlib/roda:v5.7.1-1`, with commons-ip 2.8.0. Run
  locally, not on UGent's production instance.
- **Packages:** `examples/ugent/basic` (`uuid-420bd31b-…`) and
  `examples/ugent/bibliographic` (`uuid-8cc88898-…`), built with `build.sh` at commit
  `0e4f1c2`. Both reported `VALID (errors=0 warnings=0)` under commons-ip 2.11.2.
- **Verdict:** job `f45bfa3a-…`, Default ingest workflow: both packages **SUCCESS**, 8 of
  8 steps, every step passed. A first job (`be9cc2be-…`) ended in partial success for
  both, because RODA's own Siegfried service answered 404 to its File Format Detector
  step. With that service running, the same packages pass.
- **Rows of section 4 that held,** read from the API and RODA's storage:
  - AIP type `ugent/basic` and `ugent/bibliographic`.
  - The package `OBJID` is the AIP's SIP ID. The AIP has an id of its own.
  - Representations `archival` and `access`, with types `Archival` and `Access`.
  - Essence files present, with no checksum complaints.
  - `dc.xml` as DC `SimpleDC20021212` and `mods.xml` as MODS `3.7`, at package level
    and on `archival`. The Dublin Core title, "Example photograph", became the AIP's
    label. RODA's Simple DC crosswalk indexes the title and a level, `item` by
    default. This RODA has no MODS crosswalk, so the search index shows the MODS
    AIP without title and level, as the row on `mods.xml` predicts. Adding one is
    RODA configuration.
  - Documentation and schemas present.
- **Rows that did not hold:**
  - Every representation is **not original**. The table's earlier claim of status
    ORIGINAL was wrong: our representation METS has no `TYPE` on its main `div`.
    [TODO.md](TODO.md) holds the open question.
  - Received PREMIS is kept, not dropped. The table's earlier row described a package
    without PREMIS.
- **Not checked:** an update (`--status`, `--updates`), and whether the Dublin Core
  record renders and is form-editable in the UI.

The raw API answers, as JSON, are kept locally in `tmp/roda-ingest-2026-10-12/`.

### 2026-10-13: the examples again, in a new RODA setup

- **RODA:** v5.7.1, in the image `roda:v5.7.1-3`, with commons-ip 2.8.0 and a MODS
  crosswalk (`mods_3.7.xslt`). Its storage shards AIP folders by id
  (`storage/aip/7f/17b/<id>`).
- **Packages:** `ugent/basic` as on 2026-10-12 (`uuid-420bd31b-…`), and a new build of
  `ugent/bibliographic` (`uuid-97b28a46-…`).
- **Verdict:** job `59b21ab5-…`: both packages **SUCCESS**, 8 of 8 steps.
- **Every file of each zip is in the AIP, byte for byte,** compared by MD5. The three
  METS files are kept inside the submitted zip under `submission/`, not on their own.
- **Search index:** `ugent/basic` has title "Example photograph", level `image`, the
  description and the date from `dc.xml`. `ugent/bibliographic` has title "Example
  book" and level `item`. It has no description or dates, because the MODS record does
  not hold them yet ([TODO](TODO.md), the catalogue fields not yet mapped).
- **Rows that did not hold:**
  - Every representation is still **not original**.
  - The representation's documentation file (`capture-notes.txt` of `archival`) is AIP
    documentation, not representation documentation. The cause is in commons-ip, not
    in the package: section 4 has the row.

The raw API answers are kept locally in `tmp/roda-ingest-2026-10-13/`.
