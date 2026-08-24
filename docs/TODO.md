# TODO

The live backlog. Items that a plan or ADR now owns point there rather than
sitting as loose unknowns; see [plans/](plans/) and [decisions/](decisions/).
The system as it exists today is described in [sip-creator-design.md](sip-creator-design.md);
its rough edges are collected under [Known gaps](sip-creator-design.md#known-gaps).

## Open design questions (not yet owned)

These need a decision before they become plan work.

- **Format-identification provenance as a PREMIS event.** We record *what* was identified but not *who/when/how*. Proper preservation practice is a PREMIS event ("format identification", agent: siegfried + version, signature file + date). That is what a future archivist actually needs to trust the format claim. Since ADR-0009 the raw material is finally in hand: the sidecar report's header carries exactly this (siegfried version, signature file, scandate), currently ignored by `DecodeSiegfried`. Blocked on `sip.Event` (an empty stub) growing up; when events are modeled, this should be the first one emitted. Raises the feature from "enriches a SHOULD field" to production-grade provenance. No event design exists yet anywhere; picking this up needs an ADR covering the event shape (LoC eventType vocabulary, dateTime, detail, linking identifiers), where events attach on the graph (per-file vs. package-level), PREMIS agents vs. the METS-specific `sip.Agent`, and pass-through of *received* vendor events ([input-spec.md](input-spec.md)) alongside self-generated ones.

- **meemoo SIP 2.x migration, when it stabilizes.** The basic profile targets [1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/), the stable spec ([meemoo-12 plan](archive/meemoo-12.md)); 2.0/2.1 are release candidates meemoo's production ingest does not accept. The former 2.0 target's values live in git history (pre-2026-07-17 `profiles/definition.go`). When 2.x goes stable: a *family-level* change (vocabularies, encodings, values, and the 2.x removal of the bag layer); likely a versioned meemoo family alongside 1.2, and the designated promotion trigger for the `Family` constant growing into an internal struct ([ADR-0007](decisions/0007-profile-families-share-one-writer.md)). Watch for meemoo publishing 2.x validation assets at the same moment (the parked meemoo-validation-harness idea). Needs its own spec-delta plan.

- **Identifier minting authority.** SIP Creator mints only package-local `uuid-<uuid>` IDs and asserts no authority over them ([ADR-0001](decisions/0001-package-builder-not-archive.md)). Still open:
  - Who mints the *package* identifier, and the identifiers of intellectual entities & representations: the producer, or a downstream identifier service? (Ref: [CSIP1](https://earkcsip.dilcis.eu/).)
  - Is a UUID meaningful only within the SIP acceptable as the common key tying description / IE / representation / file together? Where would externally-minted IDs be recorded if not?

- **Multiple descriptions / entities / formats.** How should the tool handle multiple descriptive records, sub-intellectual-entities, or multiple formats per representation? The model has the slots (`Entity.Entities`, per-file `Format`) but the `basic` profile builds only a single root entity.
  - Is the intended shape to walk an LD graph and populate the package from it (cf. [sipin-mh-sip-creator](https://github.com/viaacode/sipin-mh-sip-creator/tree/main/tests/resources))? Implications: identifiers would be minted by external services; there is no strong Go triplestore library, so this likely needs a supporting query API or a tech change. (Format characterisation is meanwhile decided: optional pre-computed sidecar input, [ADR-0009](decisions/0009-characterization-as-sidecar-input.md), superseding ADR-0006's mechanism; fixity stays native and in-process.)
  - For a bibliographic profile: source for mapping to MODS: BIBFRAME or otherwise? (Ref: [MODS–BIBFRAME mapping](https://www.loc.gov/standards/mods/modsrdf/mods-bibframe-mapping.html).)

- **Library builder API: the stream-first remainder.** Most of the embeddability list shipped with the [input-convention plan](archive/input-convention.md) (2026-08-20): the builder is data-fed (`profiles.Input` — descriptive terms, documentation, received PREMIS, characterization report), takes administrative metadata as data (`sip.MetsDeclaration`/`sip.Agent`), accepts a caller-supplied package identifier (updates reuse the original `mets/@OBJID`), keeps representation labels free-form, validates caller-supplied input data, and builds to a caller-controlled destination. What remains: accept essence as **streams** (`io.Reader`/`fs.FS` + logical path), not only filesystem paths, and accept **pre-computed fixity**, computing checksums only when none are supplied. See the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc.
  - **It must also ship `sip.Package.Validate()`**, the first domain-validation method (`Validate` methods on `sip/` types per `CLAUDE.md`), called between assemble and write, covering the graph-level checks: identifier uniqueness across the graph (a set-check; decided 2026-07 when the inert METS `idStore` was removed: no minting-time machinery, catch *systematic* duplication like a node wired in twice, the ~10⁻²⁵ UUIDv4 collision comes free, commons-ip's `xs:ID` XSD check stays the document-level net) and the no-empty-`Mime` invariant ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). The *input-data* validation slice already shipped with the input-convention plan ("validation splits in two"); these graph checks become real error classes when callers construct graphs and supply identifiers wholesale.

## Validator status

**Resolved 2026-07-17** ([meemoo-12 plan](archive/meemoo-12.md)): both sample
packages report **VALID**: `basic` against E-ARK 2.0.4 (the era meemoo 1.2
builds on; its SIP2 check expects exactly the unversioned profile URL meemoo
mandates), `eark` against 2.2.0. The old SIP2 failure was a spec-version
mismatch, not a package defect; the old XSD-failure evidence stopped
reproducing under commons-ip 2.11.2.

**Resolved 2026-08-20** ([input-convention plan](archive/input-convention.md)
I6b): the last SHOULD-level warning, **CSIPSTR16**, cleared once `tmp/basic`
gained `documentation/` at package and representation level. Both profiles
now validate **VALID with zero warnings**.
