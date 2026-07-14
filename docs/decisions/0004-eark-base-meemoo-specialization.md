# 0004 — E-ARK CSIP is the base; meemoo SIP 2.0 is a specialization

Status: **Accepted** (2026-07-14)

## Context

SIP Creator has two audiences that are not the same target. The immediate one is **meemoo**: packages ingested into the Flemish heritage archive (hetarchief.be) must satisfy meemoo SIP Specification 2.0. The broader one is **E-ARK CSIP**: the pan-European profile that RODA and other CSIP-conformant systems accept. meemoo SIP 2.0 is itself *defined as* a specialization of E-ARK CSIP — it adds constraints (a specific `OTHERCONTENTINFORMATIONTYPE` URL, a descriptive namespace, agent conventions) on top of the CSIP base.

The code grew meemoo-first: the meemoo `TYPE`, the `2.0/basic` profile URL, and the UGent agent block are baked directly into the shared METS templates. There is also a second profile, `roda.go`, that was an attempt at a RODA-targeted variant — but it is a near-copy of the meemoo `Basic()` path with representation PREMIS omitted, is unreachable from the CLI, and is known-broken. This left the relationship between "meemoo package" and "CSIP/RODA package" muddled in the code even though the spec relationship is clear.

## Decision

**Treat E-ARK CSIP as the base contract every package must satisfy, and meemoo SIP 2.0 as a specialization layered on top of it.** The precedence rule, stated in CLAUDE.md: when in doubt, CSIP validity wins over convenience, and the meemoo spec wins over convenience *within* the meemoo layer.

Architecturally this means:

- The `sip/` domain graph is **spec-neutral** and shared across profiles.
- Variation *within* the meemoo family (basic vs. a future content profile) is **data** — profile-specific values, not duplicated code paths.
- Variation *across* families (a genuine E-ARK/RODA SIP vs. a meemoo SIP) is a **writer + template set**: the seam is a different METS/descriptive template family reading the same graph, not a forked build pipeline.

**The broken `roda.go` is to be deleted, not migrated.** A real E-ARK target is a plain, valid CSIP SIP produced by a dedicated writer — not a meemoo variant with a field removed. Reviving the copy would entrench exactly the confusion this decision resolves.

## Alternatives rejected

- **meemoo-only, ignore CSIP as a first-class target.** Rejected: it forecloses RODA and any other CSIP consumer, and it obscures that meemoo validity *is* CSIP validity plus constraints. Building on the base keeps the door open and matches how the specs actually relate.
- **Keep and fix `roda.go` as the RODA profile.** Rejected: it is a meemoo pipeline with representation PREMIS dropped, not an E-ARK SIP. Fixing it in place would produce a "RODA-ish meemoo package," not a clean CSIP one, and would duplicate ~90% of `Basic()` — the opposite of the base/specialization factoring.

## Consequences

- **Meemoo literals must move out of shared templates** into profile-specific data so the CSIP base is genuinely reusable. This is factoring work, sequenced in the [refactoring plan](../plans/refactoring-plan.md) (declarative profile spec + registry), and gated on behavior preservation via external validation ([ADR-0003](0003-validation-stays-external.md)).
- A future E-ARK/RODA writer is a **new template family reading the existing graph**, added when that work starts — the refactor creates the seam, not the full E-ARK implementation.
- Acceptance for the eventual E-ARK target is two-sided: commons-ip validation in E-ARK SIP mode **and** an actual ingest test against a RODA instance. Spec-on-paper and what-RODA-accepts must both hold.
- Until that writer exists, only the meemoo `basic` profile is real; the design doc's [Known gaps](../sip-creator-design.md#known-gaps) list the baked-in literals and the dead `roda.go` as open items this decision governs.
