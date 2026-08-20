# Plan: the descriptive vocabulary becomes one closed table

*Status: **Draft** (2026-08-20) — for review. Accompanied by
[ADR-0011](../decisions/0011-closed-descriptive-vocabulary.md) (proposed).
Independent of the [input-convention plan](input-convention.md)'s remaining
steps; its I7 docs sweep should land after this.*

## Context

A design review (2026-08-20) of the terms machinery, grounded in three
sources, found the weight in the encoder comes from one decision —
supporting an open vocabulary no downstream consumer accepts:

- **The meemoo basic profile is closed.** "Descriptive metadata in
  dc+schema.xml MUST be limited to the DCTERMS and SCHEMA elements outlined
  in the table" — ~25 elements with cardinalities (many 0..1), four
  required at 1..1 (`title`, `identifier`, `description`, `created`), and
  EDTF typing on the dates. (The XSD itself is lax — an order-free choice
  of any dc/dcterms/schema element — so CSV-order emission is safe; the
  strictness lives in the profile.)
- **Our validation vocabulary is the wrong list twice over**: the embedded
  DCMI-55 table accepts terms meemoo rejects (`dcterms:accrualPolicy`),
  and the open `schema:*` passthrough cannot express what meemoo actually
  wants from schema.org (structured values like `schema:creator` with
  `@roleName`). The 40-entry `dumbDown` map exists only to cover all 55
  DCMI terms' refinement relations — generality the open input forces.
- **Peer tools keep vocabulary as data, not code.** RODA-in's
  standard-specific knowledge lives in template files (its engine is
  generic); Archivematica's `metadata.csv` is minimally-validated
  pass-through. The packaging tool's engine should be generic; the
  vocabulary should be data.

Also exposed, as conformance gaps rather than style: we require 2 of
meemoo's 4 mandatory elements, and enforce no 0..1 cardinalities — repeated
`abstract` rows produce meemoo-invalid documents today.

## Design

**One table is the metadata model.** A vocabulary table in
`encoders/metadata`, each row carrying everything the four scattered
structures encode today:

| key | element | required | repeatable | xsi:type | simple-DC parent |
|---|---|---|---|---|---|
| `created` | `dcterms:created` | yes | no | `edtf:EDTF-level1` | `date` |
| `subject` | `dcterms:subject` | no | yes | — | `subject` |
| `abstract` | `dcterms:abstract` | no | per-language | — | `description` |

`repeatable` is tri-state: **no** / **yes** / **per-language**. meemoo counts
cardinality per `xml:lang` value for its lang-tagged elements — two
`title[nl]` rows are invalid, `title[nl]` + `title[en]` is fine — so the
1..1/0..1 elements that carry a language (`title`, `description`,
`abstract`, `rights`) are *per-language*, not *no*.

This **replaces**: `plainKeys` (cli/input), the DCMI-55 `dctermsProperties`
map, `schemaPropertyRx`, `isEDTFTyped`, and the `dumbDown` map. The CSV
decoder maps keys through the table; validation checks membership,
requiredness, and repeatability against it; the templates render what the
row says. Net effect: ~100 lines of vocabulary machinery deleted, ~30 lines
of table added — and the table reads as the meemoo metadata model at a
glance.

**The key set follows the meemoo basic profile**, extended beyond the
current spec §3 table with its flat-expressible elements: `alternative`,
`abstract`, `issued`, `available`, `temporal`, and (as the two schema.org
text elements) `artmedium`, `artform`. Structured schema.org values
(`schema:creator` with roles, `schema:isPartOf` variants, the quantitative
height/width/depth/weight) cannot live honestly in `key,value` and stay
deferred with §8's operator-supplied XML.

**Requiredness is profile data.** meemoo requires four elements; plain
E-ARK has no such rule. The `required` marks apply per family: the basic
profile enforces the table's marks; eark enforces only the input
convention's own MUSTs (identifier, title — identity plumbing). Mechanism:
the `Definition` names its required set (data, per ADR-0007's
profiles-as-data), the builder checks it in `Build`.

**The Dutch-entry MUST is family data too.** meemoo requires that every
lang-tagged element present includes an occurrence with `xml:lang="nl"`
(other languages are welcome alongside; Dutch must be among them). Same
mechanism as requiredness: the `Definition` carries a required language
(`nl` for the meemoo family, empty for eark), the builder checks it.

**Prefixed keys (`dcterms:*`, `schema:*`) leave the input convention.**
Spec §3's "MAY use prefixed keys" clause moves to §8 (deferred, chosen
against for v1): every element meemoo accepts has a plain key, so the open
escape hatch only lets operators build invalid packages.

## Steps

### V1 — the table

- [x] Vocabulary table type + the one instance in `encoders/metadata`
      (key, element, required, repeatable [no/yes/per-language], xsi:type,
      simple-DC parent). *(`vocabulary.go`: `vocabularyRow`, the
      `vocabulary` slice, indexes by key and element, `ResolveKey`.)*
- [x] CSV decoder maps keys through it; unknown key stays a violation with
      file/line context; prefixed-key parsing removed (a prefixed key gets
      a pointed "every element has a plain key" violation).
- [x] Validation against the table replaces the DCMI-55 membership check;
      `schemaPropertyRx` and the open-prefix branch go. Lang-shape,
      non-empty-value, and the validate-before-render guard stay as they
      are.
- [x] Templates read typing from the row (`isEDTFTyped` deleted);
      `EncodeDCTerms` maps through the row's simple-DC parent (`dumbDown`
      and the DCMI-refinement map deleted).
- [x] Tests updated: table membership, new keys (`abstract`, `alternative`,
      `issued`), rejection of `dcterms:accrualPolicy` and any prefixed key.
- [x] Gate: builds green; baseline clean (same elements emitted for the
      current fixtures once `dcterms:abstract[nl]` becomes `abstract[nl]`).
      *(2026-08-20: basic and eark both VALID, baseline structurally
      identical.)*

### V2 — profile conformance

- [x] Requiredness per family: basic enforces meemoo's four
      (`identifier`, `title`, `description`, `created`); eark the input
      convention's two. Carried as `Definition` data (`RequiredElements`),
      checked fail-fast in `Build` via `Definition.validateDescriptive`
      with all findings joined.
- [x] Repeatability enforced per the tri-state marks: a second `abstract`
      row *in the same language* is a violation (CLI — the convention's
      own §3 rule, so `check` stays configuration-free) and a fail-fast
      error (builder, for embedding callers under `EnforceCardinality`) —
      the standing two-audiences split. eark enforces none of the meemoo
      cardinality marks at the builder. One mechanism for both audiences:
      `Terms.ValidateCardinality`, run by the decoder on the finished list
      — a cross-row finding names the element and language, which locates
      the rows in a keyed file without line numbers (reviewed 2026-08-20:
      an incremental per-row variant wasn't worth its bookkeeping). The
      decoder's hand-rolled identifier counting is subsumed.
- [x] Required language enforced: under the basic definition, any
      lang-tagged element present must include an `nl` occurrence
      (`Definition.RequiredLang`, `Terms.ValidateRequiredLang`); eark
      carries no required language. Same two-audiences split.
- [x] The eark fixture's `metadata.csv` slims to the convention's own
      MUSTs (identifier, title — without `description`/`created`), so the
      eark gate exercises the relaxed requiredness on every run; the basic
      fixture keeps all four.
- [x] Tests pin the per-family required sets: terms without
      `description`/`created` build under the eark definition and are
      refused under basic (fail-fast) and by the CLI (violation).
- [x] Validator loop: both profiles VALID. *(2026-08-20.)*
- [x] Gate: builds green; baseline clean.

### V3 — spec and docs

- [x] [input-spec.md](../input-spec.md) §3: key table extended with the new
      keys and a "repeatable" column; the prefixed-keys clause moved to §8;
      required keys per profile noted. *(Done at plan drafting, 2026-08-20,
      so the operator contract leads the implementation.)*
- [ ] [ADR-0011](../decisions/0011-closed-descriptive-vocabulary.md) →
      Accepted.
- [ ] Design doc: encoder section describes the table; TODO housekeeping.
- [ ] Plan retires to `archive/`.

## Verification

1. `./build.sh basic` + `eark` VALID after every step; baseline clean
   throughout (no output change for the current fixtures beyond the
   fixture's own `dcterms:abstract` → `abstract` respelling).
2. Go tests: every table column exercised (required, repeatable including
   per-language counting, typing, dumb-down parent); prefixed keys
   rejected via CSV and via direct `Terms` construction; meemoo's
   four-required rule and the `nl` required-language rule on basic and
   their absence on eark.
3. By hand: a folder with a repeated `abstract` reports a violation naming
   the element and language; `check` needs no configuration, as before.

## Out of scope, recorded

Structured schema.org values and operator-supplied descriptive XML
(spec §8); RODA-in-style form/template UI; in-process XSD/SHACL validation
(ADR-0003 — meemoo's SHACL and commons-ip stay the referees); MODS and the
bibliographic profile (TODO).

Also deliberately out: **value-shape validation of typed values** — EDTF
on `created`/`issued`, `xs:duration` on `extent`, `xs:dateTime` on
`available`, BCP47 on `language` beyond the existing lang-shape check.
The table drives *emission* of `xsi:type`; checking the values themselves
means maintaining an EDTF parser plus two XSD lexical checkers that can
only drift from what meemoo's SHACL actually accepts. A malformed value
surfaces in the standing validator loop instead (ADR-0003).
