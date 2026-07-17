# Runbook: ingesting an eark SIP into RODA

*The field-acceptance procedure for the [eark profile](plans/eark-writer.md):
produce a package, pre-flight it, ingest it into UGent's RODA instance, and
verify what arrived. Written 2026-07-17 from a desk-check of RODA's ingest
code (`EARKSIP2ToAIPPlugin` / `EARKSIP2ToAIPPluginUtils`, commons-ip2 2.11.2 —
the same version our validator pins); the UI steps describe the expected flow
and should be checked against the instance's RODA version on first use.*

## 1. Produce the package

```sh
./bin/sip-creator create --profile eark <source-dir> <dest-dir>
```

The **zip is the deliverable** (`<dest-dir>/uuid-<uuid>.zip`) — RODA ingests
zips directly; no bagging for RODA (bags are meemoo's envelope, [ADR-0008](decisions/0008-bag-layer-out-of-scope.md)).
Include a `documentation/` directory in the source: it satisfies CSIPSTR16 and
maps into the AIP.

## 2. Pre-flight

```sh
./scripts/validate.sh -s 2.2.0 <dest-dir>/uuid-*.zip
```

Expect `VALID (errors=0 warnings=0)`. Do not submit anything that isn't —
the validator shares its implementation (commons-ip 2.11.2) with RODA's
parser, so a validator failure predicts an ingest failure.

## 3. Ingest

In RODA: **Ingest → Transferred Resources** → upload the zip, then create an
ingest job over it selecting the **E-ARK SIP 2** format (the
`EARKSIP2ToAIPPlugin` path). Leave "technical metadata validation" off (the
package carries none) and "create submission" per local policy.

RODA does **not** run the full specification validator on ingest: it parses
the package into its model and gates on the parse-level report (METS
readable, referenced files present, checksums matching). Failures surface in
the job report per SIP.

## 4. Verify what arrived

The desk-check of RODA's mapping code, as a checklist — each row is what the
code does with what our package contains:

| our package | RODA mapping | verify in the UI |
|---|---|---|
| `csip:CONTENTINFORMATIONTYPE="MIXED"` | becomes the **AIP type** | AIP shows type MIXED |
| rep METS `OBJID="representation_1"` | becomes the **representation id**, status ORIGINAL | one original representation named `representation_1` |
| essence in `data/` with METS checksums | files created; fixity verified at parse | files present, sizes right, no checksum complaints in the job report |
| `dc+schema.xml`, `MDTYPE="DC"` + `MDTYPEVERSION="SimpleDC20021212"`, simple-DC shape | recognized descriptive metadata (`dc_SimpleDC20021212`) | title/description **rendered and indexed** (searchable), form-editable — if RODA shows raw XML instead, the typing didn't match |
| `documentation/` files | mapped into AIP documentation | visible under the AIP's documentation |
| `schemas/` XSDs | mapped into AIP schemas | present (RODA ignores content) |
| **no PREMIS** (premis-less v1) | nothing to map — and note: package-level PREMIS that isn't an agent/event would be *silently dropped* by RODA anyway | AIP has **no** preservation metadata from the SIP; RODA's own ingest events appear instead — that is expected, not a defect |

## 5. Record the outcome

Whatever happens, write it down in the [eark plan](plans/eark-writer.md)'s
execution record: RODA version, job report verdict, any row of the table that
didn't hold. A failed row is a finding about either our package or this
desk-check — both are worth exactly this feedback loop. On a clean first
ingest, the eark plan's field-acceptance tier is done and the plan can retire
per the [docs lifecycle](README.md#lifecycle-what-happens-when-a-plan-ships).
