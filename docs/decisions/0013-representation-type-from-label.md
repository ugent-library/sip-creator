# 0013 — The eark profile types each representation METS by its label

Status: Accepted (2026-09-03)

## Context

RODA shows a Type on every representation of an AIP. At ingest,
`EARKSIP2ToAIPPluginUtils.getType(sr)` sets it from what commons-ip parsed as
the representation's content type, which commons-ip reads from the
representation METS root's `csip:CONTENTINFORMATIONTYPE`, preferring
`csip:OTHERCONTENTINFORMATIONTYPE` whenever it is set. The `OBJID` and `LABEL`
attributes, the only place our packages carried the producer's representation
name (`archival`, `preservation`, `access`), are never consulted for Type.

The eark profile used to stamp its one profile-level declaration onto the
package METS and every representation METS alike, so every representation
declared the fixed `CONTENTINFORMATIONTYPE="MIXED"` and RODA showed the same
Type for all of them ("Other", after its vocabulary lookup fell through).

Two upstream issues frame the fix:

- [keeps/roda#3139](https://github.com/keeps/roda/issues/3139): exactly this
  problem. Fixed in RODA v5.7.0 (commit `dafa809`, April 2024) so that a
  representation METS declaring `CONTENTINFORMATIONTYPE="OTHER"` plus free
  text in `OTHERCONTENTINFORMATIONTYPE` shows that free text as the Type. The
  fix ships with a regression-test SIP, so the channel is supported, not
  accidental.
- [keeps/commons-ip#372](https://github.com/keeps/commons-ip/issues/372) (open
  as of 2026-09): commons-ip's parser fills its `contentType` from
  `CONTENTINFORMATIONTYPE`/`OTHERCONTENTINFORMATIONTYPE` instead of
  `TYPE`/`OTHERTYPE`, and never fills `contentInformationType` at all. RODA's
  Type today works *because of* that crossed mapping. If commons-ip fixes the
  parse as the issue proposes, RODA would start reading the representation
  METS `TYPE`/`csip:OTHERTYPE` instead.

Which field becomes the Type is decided entirely in commons-ip and RODA; a
package builder can only choose what to emit.

## Decision

`Definition.RepresentationTypeFromLabel`, set on the eark profile: each
representation METS declares its producer label in **both** pairs of
attributes:

- `TYPE="Other"` with `csip:OTHERTYPE="<label>"`, and
- `csip:CONTENTINFORMATIONTYPE="OTHER"` with
  `csip:OTHERCONTENTINFORMATIONTYPE="<label>"`.

The second pair is what RODA v5.7.0+ reads today; the first pair is what it
would read after a commons-ip fix for #372. Emitting both costs two attributes
and removes the dependency on which side of the crossed mapping the installed
versions sit. Both values are legal: "Other" is in the CSIP content-category
vocabulary, and CSIP requires the OTHER... companion attribute exactly when
the base attribute is Other/OTHER.

The package METS keeps the profile declaration unchanged (`TYPE="Mixed"`,
`CONTENTINFORMATIONTYPE="MIXED"`), so the AIP-level type is unaffected. The
basic (meemoo) profile is unchanged: meemoo 1.2 fixes the representation
METS's content typing to `OTHER` plus the profile URI.

## Alternatives rejected

- **Emit only the CONTENTINFORMATIONTYPE pair** (what RODA reads today):
  works on RODA v5.7.0+, but silently stops working if a commons-ip release
  fixes #372 and RODA picks it up. The dual declaration is two attributes.
- **Keep `MIXED` and rename representations to carry the type in `OBJID`**:
  RODA never reads `OBJID` or `LABEL` for Type; no naming scheme helps.

## Consequences

- On RODA v5.7.0 and later the representation Type column shows the
  producer's label verbatim (stored casing: `archival`, `preservation`,
  `access`). On older RODA (5.6.x and earlier, before the roda#3139 fix)
  nothing we emit changes the Type.
- CSIP intends `CONTENTINFORMATIONTYPE` for content-format vocabularies
  (ERMS, SIARD, ...); `OTHER` plus a free-text name bends it into a name
  carrier. It is spec-conformant, and it is the only field current RODA maps
  to representation Type.
- The package and representation METS now declare different `TYPE` values in
  the eark profile. CSIP allows that; a validator comparing them would be
  wrong to.
- Labels are free text from the producer, used verbatim. If a controlled
  representation-type vocabulary is ever wanted in RODA, label validation
  would have to be added on our side; nothing constrains it today.
- If commons-ip fixes #372, re-check which pair RODA reads; the dual
  declaration is designed to survive that change without a release on our
  side.
