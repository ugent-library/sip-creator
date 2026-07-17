# 0007 — Profile families share one writer and select encodings

Status: **Accepted** (2026-07-17, with the [eark-writer plan](../plans/eark-writer.md))

## Context

The profile registry (refactoring plan, Phase 2) made a profile *data*: one engine,
`Builder.Build(def)`, reads a `Definition` and emits a package. The eark profile
introduces a second output *family*: a plain E-ARK SIP next to the meemoo SIP
specialization. The profiles form a real hierarchy, mirroring the spec inheritance:

```
CSIP → E-ARK SIP → { plain ("eark") | meemoo 2.x → { basic, material-artwork, ... } }
```

The design question: how does a definition say which family it emits, and how much
machinery does a family get?

Auditing the actual variance between the two families, at the eark plan's scope,
against what Phase 2 already made data:

- METS attribute values (PROFILE, TYPE, content-information types) — data (`sip.Spec`).
- metsHdr agents — data.
- PREMIS emission — data (`Emit*Premis` flags) with template guards already in place.
- Documentation — conditional on the graph carrying documentation nodes.
- dmdSec typing (`MDTYPE`/`MDTYPEVERSION`) — one attribute pair, expressible as data.
- **Descriptive metadata encoding** — dc+schema vs simple DC: two genuinely
  different output *documents*. The only behavioral difference.

Both families are physically CSIP packages: same folder layout, same emission
order, same fixity discipline.

## Decision

`Definition` gains a `Family` field: a typed string constant (`FamilyMeemoo`,
`FamilyEARK`) — pure data, declared explicitly by every registry entry. **No
default**; an unknown or empty family is a build error naming the definition.

A family selects **encodings, not writers**. There is one writer — the canonical
emission order stays encoded exactly once (`write.go`, the Phase 1 invariant) — and
the family resolves internally to its behavioral choices, today exactly one: the
descriptive encoder (meemoo → dc+schema via `Descriptive.Encode`; eark →
`metadata.EncodeDC`). Everything else a family "owns" is data on `Definition`/`sip.Spec`.

Templates are shared and data-driven, organized per metadata standard in
`encoders/` (a template is its encoder's implementation detail — ADR-0002). A
family needing its own *document* adds a define inside the standard's package —
the precedent is `encoders/metadata` already holding `"dc+schema"` and `"dc"` as
two defines. No `templates/` package: grouping by material instead of purpose.

The guiding split: **behavior belongs to the family (a closed, code-level set —
one exists only when its encodings exist), values belong to the profile (an open,
data-level set — registry entries).**

## Fork triggers (recorded so this argument is not re-won from scratch)

- **Fork a template define** when a family needs *structure* that data cannot
  express cleanly — evidence required: a failing validator check or an explicit
  spec requirement, not anticipation.
- **Fork a writer** only when a family stops being CSIP-shaped — a different
  emission sequence or physical layout. Until then, the single orchestrator wins:
  today's "conditionals" are zero (every difference is an encoder swap or data).
- **Promote `Family` from constant to internal struct**
  (`family{descriptive encoder, ...}` resolved from the constant) when family-level
  choices multiply — e.g. the meemoo-2.1 family. The constant stays the data-level
  representation on `Definition`: serializable, embeddability-friendly.

## Consequences

- A new profile in an existing family is one registry entry.
- A new CSIP-shaped family is: an encoding choice, data values, one constant, one
  resolution case. Not a writer, not a template family.
- `Definition` remains pure data; the canonical emission order remains singular.
- The meemoo path keeps no privileged position: both families declare themselves
  explicitly.

## Rejected alternatives

- **Sibling writers** (`write_meemoo.go`/`write_eark.go`) — two copies of the
  load-bearing emission order; the `Basic()`/`Roda()` duplication (ADR-0004's
  motivating defect) reintroduced at writer level.
- **A sibling eark METS template family** — guarded against structural differences
  that do not exist at the current scope; over-parametrization rot cuts the other
  way when documents are structurally identical and differ in data.
- **A `Write func` field on `Definition`** — family-level behavior on leaf-level
  data: kills serializability/comparability, repeats the writer per leaf.
- **A nil-means-meemoo default** — hides a load-bearing choice in a zero value and
  silently privileges one family.
- **Definition-inherits-definition config chains** — simulated inheritance in
  config; Go composition (constant → internal struct, when needed) covers sharing.
