# 0038 — An update under the UGent profiles states UPDATE

Status: Accepted (2026-10-13)

## Context

E-ARK SIP 2.2.0 (SIP3) gives the package status in `metsHdr/@RECORDSTATUS` as one of `NEW`, `SUPPLEMENT`, `REPLACEMENT`, `TEST`, `VERSION`, `DELETE` and `OTHER`. The tool wrote only those values, without `OTHER`.

commons-ip, which RODA parses packages with, reads the status against a list from 2016–2017, before E-ARK SIP 2.0 defined a vocabulary: `NEW`, `UPDATE`, `REPLACEMENT` and `TEST`. It reads any other value as `NEW`. RODA v5.7.1 runs `NEW`, updates the AIP on `UPDATE`, and rejects every other status. The test on 2026-10-13 confirmed it ([runbook](../development-roda-ingest.md#6-outcomes)): a package with `REPLACEMENT` failed with "Unknown IP Status", and the same package with `UPDATE` updated the AIP. No status the tool could write updated an AIP in RODA. The mismatch is reported as [keeps/commons-ip#389](https://github.com/keeps/commons-ip/issues/389). commons-ip's maintainers recommend `UPDATE` for an update in RODA (keeps/commons-ip#346).

UGent needs to update packages in its RODA, through the library as well as the CLI.

## Decision

- **The tool knows `UPDATE` as a record status, next to the SIP3 values.** `sip.RecordStatusUpdate` is an update, so it needs the earlier package's identifier, like `REPLACEMENT`.
- **A profile can narrow the record statuses a package may state,** in `Definition.RecordStatuses`. `Definition.ValidateSource` refuses a status outside the list. A package without a status always passes, because the E-ARK SIP specification reads it as `NEW`.
- **The UGent profiles allow `NEW` and `UPDATE`,** the two statuses RODA acts on. Refusing the others in the tool keeps an operator from a package that RODA rejects or ingests as a second AIP.
- **`meemoo/basic` sets no list,** so it keeps every status the tool knows.
- **The tool leaves out `OTHER`.** METS has no attribute that says which other status is meant, so an archive cannot act on it.

## Alternatives rejected

- **Keep the SIP3 values and wait for RODA to read them.** The fix would be in commons-ip and RODA, and nothing in their trackers plans it. UGent could not update a package until then.
- **The operator types an SIP3 value and the UGent profiles write `UPDATE` instead.** The METS would state a value the operator did not ask for, and nothing in the command shows it.
- **`UPDATE` for every profile.** Meemoo's ingest is not RODA, and nothing says it reads `UPDATE`.

## Consequences

- An update under the UGent profiles leaves the E-ARK vocabulary. commons-ip's validation reports the package VALID, because its SIP3 check does not apply the vocabulary (keeps/commons-ip#389).
- When RODA reads the SIP3 values, the UGent profiles can move back to them. The change is the list in `profiles/ugent` and this ADR.
- The CLI's `--status update` and the library's `sip.RecordStatusUpdate` are the way to update an AIP in UGent's RODA.
