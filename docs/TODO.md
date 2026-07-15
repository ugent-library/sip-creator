# TODO

The live backlog. Items that a plan or ADR now owns point there rather than
sitting as loose unknowns — see [plans/](plans/) and [decisions/](decisions/).
The system as it exists today is described in [sip-creator-design.md](sip-creator-design.md);
its rough edges are collected under [Known gaps](sip-creator-design.md#known-gaps).

## Owned by a plan or ADR (scheduled, not yet shipped)

These are decided and sequenced; remove each line when the work lands.

- **Split filesystem handling into a `store` package** → [refactoring plan](plans/refactoring-plan.md), Step 1.
- **Path & base-path handling** (the mutated `BaseDir`, string-sliced `File.Path`) → [refactoring plan](plans/refactoring-plan.md) (`File.Path` semantics; the `store` package roots callers in relative paths).
- **Fix identifiers in the METS file** (the inert `idStore`/`lo.Contains` check) → [refactoring plan](plans/refactoring-plan.md), Step 7.
- **`mets/@TYPE` should draw from a vocabulary, not a hardcoded literal** → becomes a one-line registry fix once profile values are data ([refactoring plan](plans/refactoring-plan.md), Step 8). Ref: [CSIP2](https://earkcsip.dilcis.eu/#CSIP2).
- **Pass additional metadata (agents, etc.) into the METS files** → covered by the declarative profile spec ([refactoring plan](plans/refactoring-plan.md), Step 7–8): agents and profile literals become data.
- **RODA: representation PREMIS reported missing / mislocated** → the dead `Roda()` omits rep PREMIS; direction is a genuine E-ARK writer, not that copy → [ADR-0004](decisions/0004-eark-base-meemoo-specialization.md) + [refactoring plan](plans/refactoring-plan.md).
- **Go tests for our own logic** → the internal/external split is set: Go tests cover our assembly logic, commons-ip covers CSIP validity → [ADR-0003](decisions/0003-validation-stays-external.md). First concrete tests arrive with the [refactoring plan](plans/refactoring-plan.md) (store + assembler).
- **Siegfried is a hard dependency it shouldn't be** → format identification becomes an optional enricher; essence fixity/size (the spec MUSTs) move to native streamed computation in the store → [format-identification-optional plan](plans/format-identification-optional.md) + [ADR-0006](decisions/0006-format-identification-optional.md).
- **Siegfried/config defects** (fixed by that same plan; recorded here so they aren't lost if it stalls):
  - `siegfried.Process` mutates its args slice — every essence file after the first gets the *first* file's checksum/size/format (`formats/siegfried/siegfried.go:73`).
  - Exec errors are swallowed (`_ = bin.Run()`), so a missing `sf` panics on empty output instead of erroring (`siegfried.go:75-84`).
  - `env.Parse`'s error is discarded in `services.ConfigFromEnv`, making the `notEmpty` config validation inert (`services/config.go:33`).
  - The PREMIS `premis:format` block dereferences `.Format` unguarded — a no-match file breaks rendering (`encoders/premis/encoder.go:80-86`).

## Open design questions (not yet owned)

These need a decision before they become plan work.

- **Representation directory naming.** The `representation_([0-9]+)$` regex is stricter than either spec, and a non-matching dir (e.g. `master`) is silently skipped rather than erroring.
  - CSIP: representation folder names are free-form; only requirement is uniqueness within the package.
  - meemoo 2.0: the dir name is not fixed to `representation_1`, but MUST equal the representation METS's `mets/@OBJID`; `representation_1` is only the illustrative example.
  - Decide: (a) accept free-form names per CSIP, or (b) keep a convention but fail loudly on non-matching dirs — and either way resolve the "OBJID must == dir name" coupling for the meemoo profile. (Deferred input-side change; see the [refactoring plan](plans/refactoring-plan.md) tail.)

- **Identifier minting authority.** SIP Creator mints only package-local `uuid-<uuid>` IDs and asserts no authority over them ([ADR-0001](decisions/0001-package-builder-not-archive.md)). Still open:
  - Who mints the *package* identifier, and the identifiers of intellectual entities & representations — the producer, or a downstream identifier service? (Ref: [CSIP1](https://earkcsip.dilcis.eu/).)
  - Is a UUID meaningful only within the SIP acceptable as the common key tying description / IE / representation / file together? Where would externally-minted IDs be recorded if not?

- **Multiple descriptions / entities / formats.** How should the tool handle multiple descriptive records, sub-intellectual-entities, or multiple formats per representation? The model has the slots (`Entity.Entities`, per-file `Format`) but the `basic` profile builds only a single root entity.
  - Is the intended shape to walk an LD graph and populate the package from it (cf. [sipin-mh-sip-creator](https://github.com/viaacode/sipin-mh-sip-creator/tree/main/tests/resources))? Implications: identifiers would be minted by external services; there is no strong Go triplestore library, so this likely needs a supporting query API or a tech change. (Format characterisation is meanwhile decided: it stays available at build time but is optional — [ADR-0006](decisions/0006-format-identification-optional.md); fixity stays native and in-process.)
  - For a bibliographic profile: source for mapping to MODS — BIBFRAME or otherwise? (Ref: [MODS–BIBFRAME mapping](https://www.loc.gov/standards/mods/modsrdf/mods-bibframe-mapping.html).)

- **New input contract (convention-based folders + key–value `metadata.csv`)** → drafted in [input-spec.md](input-spec.md); supersedes the earlier loose "CSV descriptive-metadata input" idea (multi-valued fields are solved there by repeated keys). Needs an implementation plan; that plan should be accompanied by an ADR recording the config-over-self-describing-input trade-off (organization details come from configuration, so an input folder alone does not fully determine the package). Features the spec explicitly defers (chosen against for v1, not forgotten): an explicit per-file manifest, operator-supplied descriptive XML, multiple intellectual entities per package, BagIt input, per-package overrides of configured values.

- **Library builder API — embeddability requirements.** So larger ingest-automation systems can drive the library without touching the CLI's input convention, the builder API must: accept essence as streams (`io.Reader`/`fs.FS` + logical path), not only filesystem paths; accept pre-computed fixity and only compute checksums when none are supplied; take descriptive metadata as data (structs), not files to parse; take administrative metadata (agents, submission agreement, record status, content category) as data — the planned `sip.Spec`/`sip.Agent` ([refactoring plan](plans/refactoring-plan.md), Steps 7–8) is the vehicle; accept a caller-supplied package identifier (updates reuse the original `mets/@OBJID`) and mint a UUID only when none is given; keep representation labels free-form in the model (the profile decides SIP-side directory naming); build to a destination the caller controls. See the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc.

- **File copy performance.** Is there a better / more performant way to copy essence assets than the current 2 KB-buffer loop?

- **METS `@MIMETYPE` for essence files is a hardcoded lie.** Representation METS stamps every essence file `MIMETYPE="text/xml"` (`encoders/mets/encoder.go:75-78`), and CSIP makes `@MIMETYPE` a MUST. The identificator's mime output (parsed and currently discarded) is a candidate source when identification runs; an extension-based fallback is needed when it doesn't. Decide the source of truth, then fix.

## Known-INVALID: concrete validator evidence

The sample package does not yet validate. Keep these until fixed (they are the
gate that must go green before blocking automation — [ADR-0003](decisions/0003-validation-stays-external.md)).

Descriptive metadata (`dc+schema.xml`) fails XSD validation:

```
6:  54  cvc-elt.1:   Cannot find the declaration of element 'metadata'.
16: 48  cvc-elt.4.2: Cannot resolve 'edtf:EDTF-level1' to a type definition for element 'dcterms:created'.
```

→ the `<metadata>` element has no resolvable schema declaration, and the EDTF type for `dcterms:created` does not resolve. Likely a missing/mislinked XSD for the `metadata` element and the `edtf` namespace.

In RODA:

```
ERROR
representations/representation_1/representations/representation_1/metadata/preservation/premis.xml
Preservation metadata file not found.
```

→ doubled `representations/representation_1/...` path — a path-construction bug in the representation PREMIS href (related to the path-handling item above).
