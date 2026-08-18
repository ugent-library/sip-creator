# Plan: the eark profile — a plain E-ARK SIP

*Status: **implemented through E5** (2026-07-17) — awaiting field acceptance: the first real ingest into UGent's RODA instance, per the [RODA ingest runbook](../development-roda-ingest.md). Drafted 2026-07-16, reworked 2026-07-17 after design review (see "Duplication lessons").

## Execution record (2026-07-17)

E2–E4 landed in one day; E5's desk-check became the [runbook](../development-roda-ingest.md). Findings and deviations:

- **The eark package validates `VALID (errors=0 warnings=0)`** — this project's first green verdict — and the E4 loop converged in a single iteration after two findings:
  - **commons-ip's SIP2 quirk**: the 2.2.0 validator compares `mets/@PROFILE` against the *version-pinned* `…/E-ARK-SIP-v2-2-0.xml` while its error message prints the unversioned URL. The registry carries the pinned URL with a comment.
  - **A real zip bug flushed out**: `archive.Zip` wrote no directory entries, so *empty* directories vanished from zips — invisible for basic (all dirs non-empty), fatal for the premis-less eark package (its rep `metadata/` evaporated, failing CSIPSTR13/CSIP107). Fixed with proper trailing-slash entries.
- **Package-level documentation satisfies CSIPSTR16** — the E1 open question about per-representation documentation folders resolved empirically: not needed.
- **`build-eark.sh` never shipped**: review folded it into a parameterized `./build.sh [profile]` (same three-variable difference, same duplication instinct as the writers). Each profile validates against its era's spec version.
- **Phase 2's premis-less template guards were exercised for real** for the first time, as predicted.
- E6's docs landed incrementally with adjacent passes (README, design doc, CLAUDE.md all describe the eark profile, families, and green gates).* Expands the [refactoring plan](../archive/refactoring-plan.md)'s Phase 3 outline; enacts the direction of [ADR-0004](../decisions/0004-eark-base-meemoo-specialization.md). The family mechanism is [ADR-0007](../decisions/0007-profile-families-share-one-writer.md).*

## Context — what an "eark" profile is

The meemoo profiles and the eark profile serve **different consumers**. Meemoo SIPs go
to hetarchief's pipeline (SHACL validation keyed on `OTHERCONTENTINFORMATIONTYPE`);
plain E-ARK SIPs go to RODA-class CSIP repositories. UGent will submit real E-ARK
SIPs to a RODA instance it manages. Scope for v1, deliberately novice-shaped:
**essence + descriptive metadata + documentation folder** — no submission agreement
fields, no PREMIS (see Delta B), no archival-creator/preservation agents.

Acceptance is two-bar, and the bars are not the same thing:

- **commons-ip's full validator** (our dockerized rig, 2.11.2) checks the
  specification exhaustively — the eark package must report **VALID**, which would be
  this project's first.
- **RODA's ingest** never runs that validator: `EARKSIP2ToAIPPlugin` calls
  `new EARKSIP().parse(...)` and gates on the *parse-level* report (METS readable,
  files present, checksums match), then maps the commons-ip model into an AIP
  (`EARKSIP2ToAIPPluginUtils`). RODA embeds the **same commons-ip 2.11.2** our
  validator pins, so the rig is a faithful proxy; the desk-check against the mapping
  code covers the rest. No local RODA instance (deliberate: non-trivial to run); the
  field test is the first real ingest on UGent's own instance.

### The profile hierarchy (conceptual model)

```
CSIP  (information-package base)
└── E-ARK SIP  (submission specialization)
    ├── "plain"          ← the eark profile: stops here, adds nothing
    └── meemoo SIP 2.x   (Flemish heritage specialization)
        ├── basic        ← the existing profile
        └── material-artwork, newspaper, ...   (future)
```

Each level owns specific data:

| level | owns | examples |
|---|---|---|
| root (shared) | graph, assembler, writer, store, input contract | `sip/`, `assemble.go`, `write.go` |
| **family** | dialect-intrinsic choices | descriptive *encoding*, `PROFILE` URL, dmdSec typing |
| **leaf/profile** | the content shape | name, content-information type, emission flags, descriptive source |
| *(neither)* | the agents — organization config, not profile data | "Universiteitsbibliotheek Gent" |

The agents row is a known mis-homing: they live on `Definition.Mets` today (the
right move when Phase 2 lifted them out of templates) but belong to a future
organization-config axis — already tracked by the library-embeddability item in
[TODO.md](../TODO.md).

### Duplication lessons (why there is no `write_eark.go` and no eark template family)

Two designs were considered and rejected during review because they re-created the
`Basic()`/`Roda()` duplication this codebase already paid to remove:

1. **Sibling writers** (`write_meemoo.go` / `write_eark.go`) would carry two copies
   of the canonical emission order — the load-bearing invariant Phase 1 encoded
   exactly once. Auditing the actual writer showed the family-varying surface is
   three encoder call sites plus one data-conditional step; forking an 80-line
   orchestrator to vary that is the Basic/Roda mistake with better branding.
2. **A sibling eark METS template family** guarded against structural differences
   that, at this plan's scope, do not exist: PROFILE/TYPE/content types and agents
   are already data (`sip.Spec`, Phase 2); PREMIS blocks are already conditionally
   guarded; documentation becomes graph-conditional; dmdSec typing is one
   attribute, expressible as data. Both families are physically CSIP packages —
   that is the premise of the hierarchy above.

Therefore: **one writer, shared data-driven templates, families select encodings**
([ADR-0007](../decisions/0007-profile-families-share-one-writer.md)). Descriptive
metadata is the one genuine per-family *document* difference, and the codebase
already holds the precedent: `encoders/metadata` contains two document defines
(`"dc+schema"` in use, `"dc"` unused) in one package — an encoder package per
standard, multiple document defines within it. No `templates/` package: a template
is the implementation detail of its encoder (ADR-0002), and grouping by material
instead of purpose is taxonomy, not architecture.

**Fork triggers, recorded so nobody re-wins this argument from scratch:** fork a
template define when a family needs *structure* that data cannot express cleanly;
fork a writer only when a family stops being CSIP-shaped (different emission
sequence or physical layout). Both require evidence (a failing check, a spec
requirement), not anticipation.

## The two deltas (measured 2026-07-16)

**Delta A — current output → validator-VALID.** Of 156 checks the current basic
package fails exactly two; everything else passes or is conditional-and-vacuous:

| check | level | eark profile answer |
|---|---|---|
| SIP2 | MUST | `mets/@PROFILE = "https://earksip.dilcis.eu/profile/E-ARK-SIP.xml"` (root + rep METS) — already data |
| CSIPSTR16 | SHOULD | a `documentation/` folder (validator checks representation folders too) |

Notable non-issues: the existing metsHdr agents already satisfy the submitting-agent
MUSTs (SIP15–18: `ROLE="CREATOR" TYPE="ORGANIZATION"` + name); the eark entry keeps
the UGent values and drops the `OTHERROLE="OTHERROLE"` oddity. Emitting
documentation activates the currently-skipped documentation MUSTs (CSIP94: the
structMap documentation division) — the shared templates must satisfy them when
documentation is present.

**Delta B — what RODA's ingest mapping consumes:**

1. **Package-level PREMIS is silently discarded** unless it is a premis *agent* or
   *event* — our intellectual-entity object matches neither branch and vanishes.
   Representation-level PREMIS maps verbatim ("other" preservation metadata).
   → v1 sets `EmitPackagePremis: false, EmitRepresentationPremis: false` (matches
   the essence+descriptive scope; Phase 2's premis-less template guards finally get
   exercised). Rep-level PREMIS can switch on later when provenance/events mature.
2. **Descriptive metadata must be `dc_SimpleDC20021212`** to be rendered, indexed,
   and form-edited by RODA (`ui.browser.metadata.descriptive.types`): dmdSec
   `MDTYPE="DC"` + `MDTYPEVERSION="SimpleDC20021212"`, simple-DC document shape.
   The unused `"dc"` define in `encoders/metadata` must be verified against a
   known-good sample — the RODA checkout's test corpora contain packages RODA
   ingests in its own test suite; use them as the reference.
3. **Identity mapping:** AIP type ← package `CONTENTINFORMATIONTYPE` (so eark uses
   a real vocabulary value, e.g. `MIXED`, not `OTHER` + URL); RODA representation
   id ← rep METS `OBJID` (our directory name), status ORIGINAL; `documentation/`
   and `schemas/` map straight into the AIP.

**Open check (benefits basic, not blocking):** whether meemoo 2.x itself mandates
the E-ARK SIP profile URL — if yes, basic's SIP2 failure is a one-line registry fix.
Recorded in [TODO.md](../TODO.md) Known-INVALID.

## Design

- **One writer.** `write.go` keeps the single canonical emission order. The eark
  path adds one step (documentation), conditional on the graph carrying
  documentation nodes — not on the family.
- **`Definition.Family`** (`FamilyMeemoo`, `FamilyEARK`): a typed string constant —
  pure data, serializable, declared explicitly by every registry entry; unknown or
  empty family is a build error naming the definition. Internally the family
  resolves to its one behavioral choice: the **descriptive encoder**
  (meemoo → the existing `Descriptive.Encode` / dc+schema; eark → `metadata.EncodeDC`,
  the `"dc"` define corrected to the `dc_SimpleDC20021212` shape).
- **dmdSec typing becomes data**: `sip.Spec` gains the descriptive `MDTYPE` and
  `MDTYPEVERSION` values (meemoo: `DC` and empty — byte-identical output, the
  version attribute renders only when set; eark: `DC` / `SimpleDC20021212`).
- **Documentation**: the input tree gains an optional package-level
  `documentation/` directory; the assembler walks it into `Package.DocumentationFiles`
  (shared, unconditional — harmless for basic, whose fixture has none); the writer
  copies it; the shared package-METS template renders `fileGrp USE="Documentation"`
  and the structMap documentation division inside `{{ with .DocumentationFiles }}`
  (activating CSIP94 when present). Whether the validator *also* demands
  per-representation documentation folders is verified empirically during E4
  (its CSIPSTR16 message points inside `representation_1`) — rep-level support is
  added then if required, not speculatively.
- **The basic profile is untouched behaviorally**: shared templates change only
  inside data-guards that are empty for basic; every step re-runs the baseline gate.

## Steps

- **E2 — the family seam.** `Family` type + constants + field; `basic` declares
  `FamilyMeemoo`; `Build` (or the writer's descriptive step) resolves the family's
  descriptive encoder and errors on an unknown family; test: hand-built definition
  with a bogus/empty family → clean error, nothing written. No renames, no new
  files beyond tests. Gate green.
- **E3 — the eark data and encodings.** `metadata.EncodeDC` corrected against the
  RODA corpora sample; `MDTYPE`/`MDTYPEVERSION` onto `sip.Spec` + shared templates
  (gate-safe for basic); documentation assembly + writer step + template blocks;
  the `eark` registry entry (UGent agents, vocabulary content types, premis flags
  off, `LocalIdentifierScheme: ""`); definition-driven tests (no MEEMOO-LOCAL-ID,
  documentation nodes, premis-less output, simple-DC shape).
- **E4 — validator acceptance loop.** A sibling `build-eark.sh` (build.sh stays
  the basic gate; merge once basic itself goes VALID): generate from `tmp/basic`
  (+ a documentation fixture), validate, iterate until **VALID**. Settle the
  rep-level-documentation question here.
- **E5 — RODA desk-check + handoff.** Walk the eark package through
  `EARKSIP2ToAIPPluginUtils` semantics on paper (descriptive type recognized,
  AIP/representation identity, nothing silently dropped); write a short ingest
  runbook for the first submission to UGent's RODA instance — that first real
  ingest is the field acceptance, performed by the operator, not this plan.
- **E6 — docs.** Design doc (families, eark profile, documentation contract),
  CLAUDE.md, README (`--profile eark`, documentation input), TODO housekeeping,
  status notes here and in the refactoring plan.

## Verification

1. `./build.sh` + `baseline-diff.sh`: basic output structurally identical after
   every step — the meemoo path must not move.
2. `build-eark.sh`: commons-ip reports **VALID** for the eark package (target; the
   loop in E4 iterates until it holds).
3. Go tests: family dispatch (unknown family errors, nothing written), eark
   assembly (no MEEMOO-LOCAL-ID, documentation nodes, premis flags respected),
   simple-DC encoding shape.
4. Negative paths by hand: eark build with no documentation input (SHOULD-level
   warning acceptable, build succeeds); premis-less output renders valid METS
   (no dangling ADMID).
5. Desk-check record in E5: each `EARKSIP2ToAIPPluginUtils` mapping step answered
   for our package, in the plan or runbook.

## Out of scope, recorded

- meemoo SIP 2.1 (parked → [TODO.md](../TODO.md)); submission-agreement fields;
  archival-creator/preservation agents; PREMIS in eark packages (revisit with the
  provenance/events work); a local RODA instance; agents-to-organization-config
  extraction (library-embeddability item).
