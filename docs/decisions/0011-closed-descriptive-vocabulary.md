# 0011 — The descriptive vocabulary is one closed table, owned as profile data

Status: **Accepted** (2026-08-20 — shipped the same day with the
[descriptive-vocabulary plan](../archive/descriptive-vocabulary.md);
drafted with it).

## Context

The input convention takes descriptive metadata as key–value CSV. The first
implementation accepted an open vocabulary: plain keys from a table, plus
any `dcterms:*` term (validated against the full DCMI list of 55) and any
`schema:*` property (validated by shape only). Serving that openness cost
real machinery — an embedded DCMI vocabulary, a 40-entry dumb-down map
covering every DCMI refinement, per-element special cases in the encoder.

The openness serves nobody. The meemoo basic profile is **closed** ("MUST
be limited to the DCTERMS and SCHEMA elements outlined in the table", with
cardinalities and EDTF-typed dates), so open input lets an operator build
invalid packages; and flat `key,value` cannot express the structured
schema.org values meemoo actually wants (`schema:creator` with `@roleName`).
Peer tools point the same way: RODA-in keeps its standard-specific
knowledge in template data files over a generic engine; Archivematica's
`metadata.csv` is a minimally-validated pass-through.

## Decision

**The operator vocabulary is a closed table, and the table is data.** One
row per key: CSV key, emitted element, required, repeatable, `xsi:type`,
simple-DC dumb-down parent. The parsing, validation, and rendering engines
are generic and read the table; nothing else in the code knows an element
name. Requiredness, the cardinality marks, and meemoo's required-language
rule (a Dutch entry wherever a lang-tagged element appears) apply per
profile family — the basic profile enforces them, plain E-ARK does not —
carried on the `Definition` per ADR-0007's profiles-as-data rule. Prefixed keys are not accepted; the key
set follows the meemoo basic profile's flat-expressible elements.

## Alternatives rejected

- **Open `dcterms:*`/`schema:*` passthrough (the status quo).** Validates
  against the wrong vocabulary — broader than any consumer accepts,
  narrower than schema.org needs — and pays for the generality in code.
- **Opaque metadata files, RODA-in style.** The right model for rich or
  pre-existing metadata, but it abandons the operator-friendly CSV; it is
  the spec §8 operator-supplied-XML feature, deferred, not this.
- **In-process schema validation of the emitted document.** ADR-0003:
  validation stays external; meemoo's SHACL and commons-ip are the
  referees. The table enforces only what the tool must not emit wrongly.

## Consequences

- New elements are one table row (and a spec §3 line), not code. A future
  profile family brings its own table (bibliographic/MODS will need one).
- Operators cannot emit arbitrary DC terms; the escape hatch for rich cases
  is §8's operator-supplied XML, chosen against for v1, not forgotten.
- Structured schema.org values remain inexpressible by design — the CSV
  stays honest about what a flat format can say.
- The DCMI-55 list, the refinement dumb-down map, and the encoder's
  per-element special cases are deleted; conformance *improves* (meemoo's
  required elements and 0..1 cardinalities become enforceable from the
  table).
- The input convention's promise weakens slightly: §3's "prefixed keys MAY
  be used" is withdrawn before any operator relied on it.
