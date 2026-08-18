# Plan: retarget the basic profile at meemoo SIP 1.2 (stable)

*Status: **implemented** (2026-07-17). The one pending operator input — UGent's
real OR-id — was resolved 2026-08-18: the placeholder is gone, and the submitter
(name + OR-id) is deployment configuration (`SIP_SUBMITTER_*`,
`Definition.WithSubmitter`); a meemoo build refuses to run without it.
Drafted 2026-07-17.
Companion decision: [ADR-0008](../decisions/0008-bag-layer-out-of-scope.md)
(the bag layer is out of scope).*

## Execution record (2026-07-17)

All changes A–F landed as planned. Findings and resolutions of the open items:

- **`schemas/descriptive_basic.xsd` is byte-identical to meemoo's published
  1.2 XSD after the two-line namespace change** — both copies evidently derive
  from the same source, settling the provenance question in practice.
- **The SIP2 mystery resolved as a spec-version mismatch**: commons-ip's base
  `SipMetsValidator` (used for spec 2.0.4) compares `mets/@PROFILE` against
  exactly the unversioned URL meemoo 1.2 mandates; only the 2.1.0/2.2.0
  subclasses demand version-pinned URLs. `validate.sh` gained `-s`, and
  `build.sh` pins the spec version per profile era (basic → 2.0.4,
  eark → 2.2.0) — still pinned, per ADR-0003's spirit, now era-correct.
- **Result: the basic profile validates `VALID` for the first time**
  (errors=0, 131 passed, one SHOULD warning: CSIPSTR16, fixture has no
  documentation dir). The eark gate stays green. Both `./build.sh` gates
  exit 0.
- **Required descriptive fields** (spec, basic profile pages): `dcterms:title`,
  `dcterms:identifier`, `dcterms:description`, `dcterms:created`, all 1..1.
  The encoder emits all four (identifier always, as the entity UUID; the rest
  from input) — documented as an input requirement in README.
- **Re-baselined deliberately**: `tmp/baseline/` refreshed (tree,
  failed-checks, README note), self-tested (fresh build diffs clean).

### Desk-check record: meemoo 1.2 package MUSTs → output

| spec requirement | satisfied by |
|---|---|
| `mets/@PROFILE` = unversioned E-ARK SIP URL | registry `ProfileURL`, both METS documents |
| `csip:CONTENTINFORMATIONTYPE="OTHER"` + `OTHERCONTENTINFORMATIONTYPE` = 1.2 profile URI | registry values |
| `mets/@TYPE` from the content-category vocabulary | `Photographs – Digital` (legal; operator-selectable is a TODO) |
| software agent (CREATOR/OTHER/SOFTWARE) + SOFTWARE VERSION note | registry agent, `NoteType` field |
| submitting org agent (CREATOR/ORGANIZATION) + IDENTIFICATIONCODE note | registry agent — **placeholder OR-id pending** |
| descriptive `dc+schema.xml` at `metadata/descriptive/`, 1.2 namespace, XSD-valid | template + XSD byte-identical to meemoo's |
| required descriptive elements (title/identifier/description/created) | encoder emits all four given conforming input (README documents the input duty) |
| package `metadata/preservation/premis.xml` with the IE and relationships | unchanged, conforms |
| representation premis + file fixity (MD5), size, originalName | unchanged, conforms |
| root dir = `mets/@OBJID` | `uuid-<uuid>` naming, unchanged |
| BagIt envelope for transfer | **out of scope** (ADR-0008); operator bags the package directory |

## Context — why 1.2, and why now

The basic profile has targeted **meemoo SIP 2.0, which is a release
candidate**. The stable specification — the one meemoo's production ingest
accepts today — is **1.2**. UGent needs SIPs meemoo can actually ingest, so
`basic` retargets **in place**: the 2.0 values are preserved in git history and
in the parked 2.x-migration TODO item, resurrectable as a versioned family when
2.x stabilizes (the promotion trigger recorded in
[ADR-0007](../decisions/0007-profile-families-share-one-writer.md)).

### Sources, in order of authority

1. **The published specification** (developer.meemoo.be, 1.2) — the contract a
   content partner writes against. Every requirement below cites it.
2. **meemoo's sipin validator** (viaacode/sipin-sip-validator, `resources/1.2/basic/`)
   — meemoo's *internal implementation*, which happens to be open source. Used
   as informative intelligence only (the same standing RODA's ingest code had
   for the eark profile): it predicts pipeline behavior, it is **not** a
   requirement source. Where implementation and spec disagree, that is a
   question for meemoo, never a silently adopted requirement. Any future
   validation harness built on those assets claims "matches meemoo's current
   implementation", explicitly not "conforms to the spec" — recorded here so
   the shapes don't get re-elevated later.

### Scope boundary: the bag

meemoo 1.2 transfers SIPs as BagIt bags. Bagging is the operator's
envelope step with a reference BagIt implementation — **out of scope** for
this tool ([ADR-0008](../decisions/0008-bag-layer-out-of-scope.md)). This plan
concerns only the package inside the bag's `data/`, which is CSIP-shaped —
meaning no writer fork and no template family: the deltas below are data and
one small model addition, exactly ADR-0007's happy path.

## The delta (current output → 1.2 package, spec-cited)

Measured 2026-07-17 against the 1.2 structure pages. Our output is already
~90% a 1.2 package: file-level fixity (MD5), size, originalName, all four
PREMIS relationships, package-level `premis.xml` with the intellectual entity,
and the software agent with its SOFTWARE VERSION note all conform as-is.

| # | 1.2 requirement (spec §package structure) | current output | change class |
|---|---|---|---|
| 1 | `mets/@PROFILE` MUST be `https://earksip.dilcis.eu/profile/E-ARK-SIP.xml` | CSIP profile URL | registry data |
| 2 | `csip:OTHERCONTENTINFORMATIONTYPE` MUST be the profile URI, `https://data.hetarchief.be/id/sip/1.2/basic` | `…/2.0/basic` | registry data |
| 3 | descriptive metadata namespace + XSD are the 1.2 profile namespace | `…/2.0/basic` | template literal + 2-line XSD edit |
| 4 | submitting ORGANIZATION agent (ROLE=CREATOR, TYPE=ORGANIZATION) MUST carry a `csip:NOTETYPE="IDENTIFICATIONCODE"` note — the meemoo OR-id | no note; a stray `OTHERROLE="OTHERROLE"` | **the one real gap**: model field + template + data |
| 5 | `mets/@TYPE` from the spec's fixed content-category vocabulary | `"Photographs – Digital"` | none — the value long marked "known-wrong" is legal 1.2 vocabulary; the TODO item softens to "make operator-selectable" |

## Changes

- **A. `sip/spec.go`** — `Agent` gains `NoteType string`; the METS template's
  hardcoded `NOTETYPE="SOFTWARE VERSION"` becomes `{{ .NoteType }}` (Phase 2
  left it hardcoded while only one note type existed; #4 introduces the second).
- **B. `profiles/definition.go`, basic entry** — `ProfileURL` per #1;
  `OtherContentInformationType` per #2; the UGent agent drops `OTHERROLE` and
  gains `Note: "<OR-id>", NoteType: "IDENTIFICATIONCODE"`; the software agent
  gets explicit `NoteType: "SOFTWARE VERSION"`. The eark entry's agents gain
  their `NoteType` values (no output change there). **Needed from the
  operator: UGent's OR-id** — until supplied, a loudly-fake placeholder.
- **C. `encoders/metadata/encoder.go`** — the `dc+schema` define's namespace
  and schemaLocation move to the 1.2 profile namespace.
- **D. `schemas/descriptive_basic.xsd`** — the two namespace declarations move
  to 1.2. Provenance check while implementing: source the XSD from where the
  spec publishes it (the profile URI should resolve), not from meemoo's
  validator repo.
- **E. Tests** — registry-value assertions, agent-note rendering (both note
  types), updates to anything pinned to 2.0 strings.
- **F. `--no-zip`** (or `--output folder`) — invocation-level flag skipping
  `archive.Zip`, commons-ip's Zip/Folder `WriteStrategy` precedent: for the
  meemoo flow the package *directory* is the deliverable-to-be-bagged and the
  zip is noise. CLI concern, not `Definition` data.

### Re-baselining (deliberate, the first since Phase 0)

This is the first intentional output change to `basic` since the baseline gate
was built. Sequence: land the change, verify (below), then consciously refresh
`tmp/baseline/` (reference tree, `failed-checks.txt`, environment notes) and
record the refresh in the baseline README. The gate protects 1.2-shaped output
from then on.

## Verification — three tiers, honestly labeled

1. **Executable**: `./build.sh` — commons-ip on the E-ARK/CSIP layers, pinned
   and dockerized as always. The SIP2 outcome under the meemoo-mandated
   unversioned PROFILE URL is documented whichever way it goes (open item
   below); the meemoo spec wins within the meemoo layer (CLAUDE.md).
2. **Desk-check**: the meemoo layer, requirement by requirement against the
   **specification documentation** — a checklist in this plan when executed
   (each spec MUST → how the output satisfies it). Explicitly a desk-check,
   not a validation. No ephemeral SHACL runs: an ad-hoc reconstruction of
   meemoo's transform pipeline would validate our facsimile of their
   internals, not the spec — a false green is worse than an honest desk-check.
3. **Field**: real submission through meemoo's pipeline when UGent is ready —
   the only true meemoo-layer verdict until a *properly built* harness (pinned,
   containerized, per [ADR-0005](../decisions/0005-dockerized-validation-and-html-reporting.md)
   discipline) is un-parked, with the authority caveat from "Sources" above.

## Open items to settle while implementing

- **Required descriptive fields for 1.2 basic** — from the spec's basic-profile
  pages (not the shapes); confirm the `tmp/basic` fixture carries every MUST.
- **commons-ip SIP2 vs the unversioned PROFILE URL** — empirical; possibly
  validate `basic` with the commons-ip spec version matching 1.2's E-ARK era
  rather than 2.2.0 (the validate.sh pin is per
  [ADR-0003](../decisions/0003-validation-stays-external.md) — changing it for
  one profile is a deliberate, documented choice, not a drive-by).
- **XSD publication location** (change D provenance).

## Docs (same change as the code)

`CLAUDE.md`, `README.md`, `docs/sip-creator-design.md`: the basic profile
targets **meemoo SIP 1.2 (stable)**; 2.0/2.1 are release candidates (the
2.x-migration TODO item reframes accordingly); the meemoo delivery workflow
documents the external bagging step (bag the *directory*, not our zip);
`--no-zip` documented. TODO housekeeping: the `mets/@TYPE` item softens per
delta #5.

## Out of scope, recorded

The bag layer (ADR-0008); the 2.x families until stabilization; batches
(`schema:isPartOf`/`haSip:Batch` — optional in 1.2, no UGent use case yet);
PREMIS events/agents for digitization provenance (optional in 1.2; owned by
the parked events TODO); the meemoo validation harness (parked, with the
authority caveat).
