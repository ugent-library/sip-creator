# 0037 — The preservation representation is the original

Status: Accepted (2026-10-13)

## Context

The first ingests of the UGent examples into RODA v5.7.1, on 2026-10-12 and 2026-10-13, stored every representation as not original ([runbook](../development-roda-ingest.md#6-outcomes)). commons-ip, which RODA parses packages with, takes a representation's status from the `TYPE` of the main division of the structMap in the representation's METS. It knows two statuses, `ORIGINAL` and `OTHER`, and reads a missing `TYPE` as `OTHER`. RODA sets a representation's original flag only when the status is `ORIGINAL`. The representation METS of SIP Creator wrote no `TYPE` there.

METS leaves `div/@TYPE` free, and neither CSIP nor E-ARK SIP defines a representation status. The value is a convention of commons-ip and RODA.

The UGent profile pages define the three representations (§4 of [ugent-basic](../profiles/ugent-basic.md) and [ugent-bibliographic](../profiles/ugent-bibliographic.md)). `preservation` is the copy as delivered: the born-digital files as deposited, or the files of a digitisation as the scanning produced them. `archival` is a conversion to a format fit for long-term archiving, and `access` the copy that will be disseminated.

The profile definition already mapped each representation name to the type its METS declares, in `Definition.RepresentationTypes`.

## Decision

- **Under the UGent profiles the `preservation` representation is the original.** Its METS writes `TYPE="ORIGINAL"` on the main division of its structMap. `archival` and `access` write no `TYPE` there.
- **The status follows from the name, as the type does.** A producer cannot set it, in `representations.csv` or through the library.
- **A profile states what it knows about a representation name in one place.** `Definition.Representations` maps each name to a `RepresentationDefinition`, which holds the type and the status. It replaces `Definition.RepresentationTypes`. The graph holds the status in `sip.Representation.Status`, and the METS template writes it.
- **The UGent examples hold their scans in `preservation`,** as the profile pages define it.

## Alternatives rejected

- **A separate field on `Definition` for the statuses.** It adds a field for one value in two profiles, next to a map of the same names.
- **A function a profile registers to change the graph after assembly.** The profile would no longer be readable as data, and a function that can change any node can leave one without an identifier, a path or a media type. One case does not justify the mechanism.
- **A `status` column in `representations.csv`.** It would let a producer declare `access` the original, against the profile pages. `representations.csv` lost its `type` column on 2026-10-12 for the same reason.
- **`ORIGINAL` on `archival`.** The archival copy is derived from the delivered one when the delivered one is not in an archival format, so it is not the original.

## Consequences

- RODA shows the `preservation` representation of a UGent package as original. A package without a `preservation` representation has no original in RODA.
- The output of the UGent profiles changes, and their reference copies with it. `meemoo/basic` has no representation definitions, so its output stays the same.
- The next fact a profile states about a representation name goes into `RepresentationDefinition`, not onto `Definition`.
