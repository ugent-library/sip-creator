# TODO

The live backlog. Items that a plan or ADR now owns point there rather than
sitting as loose unknowns — see [plans/](plans/) and [decisions/](decisions/).
The system as it exists today is described in [sip-creator-design.md](sip-creator-design.md);
its rough edges are collected under [Known gaps](sip-creator-design.md#known-gaps).

## Owned by a plan or ADR (scheduled, not yet shipped)

These are decided and sequenced; remove each line when the work lands.

- **Identifier uniqueness across the package graph** → an invariant for the future `sip.Package.Validate()` (domain validation lives on the `sip/` types per `CLAUDE.md`; post-Phase-2, once the graph API settles). Decided when Step 7 removed the inert METS `idStore` (it never recorded a minted ID, so it guaranteed nothing): keep minting UUIDv4 with no minting-time machinery, and make emitting a duplicate impossible instead — `Validate`, called between assemble and write, collects every identifier in the graph into a set and fails the build on a duplicate. The real target is *systematic* duplication (a node wired into the graph twice, a copied identifier), which no random-collision defence catches; the ~10⁻²⁵ UUIDv4 collision comes along for free, and commons-ip's XSD check on `xs:ID` remains the document-level net.
- **`mets/@TYPE` should be operator- or content-selectable per package** → the registry value (`Photographs – Digital`) is *legal* meemoo-1.2 vocabulary and apt for the sample fixture, but a real archive packages more than photographs; the remaining work is input/config plumbing for the content category, not a vocabulary fix. Ref: [CSIP2](https://earkcsip.dilcis.eu/#CSIP2), meemoo 1.2 §package METS.
## Open design questions (not yet owned)

These need a decision before they become plan work.

- **Format-identification provenance as a PREMIS event.** We record *what* was identified but not *who/when/how*. Proper preservation practice is a PREMIS event ("format identification", agent: siegfried + version, signature file + date) — what a future archivist actually needs to trust the format claim. Since ADR-0009 the raw material is finally in hand: the sidecar report's header carries exactly this (siegfried version, signature file, scandate), currently ignored by `DecodeSiegfried`. Blocked on `sip.Event` (an empty stub) growing up; when events are modeled, this should be the first one emitted. Raises the feature from "enriches a SHOULD field" to production-grade provenance. No event design exists yet anywhere — picking this up needs an ADR covering the event shape (LoC eventType vocabulary, dateTime, detail, linking identifiers), where events attach on the graph (per-file vs. package-level), PREMIS agents vs. the METS-specific `sip.Agent`, and pass-through of *received* vendor events ([input-spec.md](input-spec.md)) alongside self-generated ones.

- **meemoo SIP 2.x migration, when it stabilizes.** The basic profile targets [1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/), the stable spec ([meemoo-12 plan](archive/meemoo-12.md)); 2.0/2.1 are release candidates meemoo's production ingest does not accept. The former 2.0 target's values live in git history (pre-2026-07-17 `profiles/definition.go`). When 2.x goes stable: a *family-level* change (vocabularies, encodings, values, and the 2.x removal of the bag layer) — likely a versioned meemoo family alongside 1.2, and the designated promotion trigger for the `Family` constant growing into an internal struct ([ADR-0007](decisions/0007-profile-families-share-one-writer.md)). Watch for meemoo publishing 2.x validation assets at the same moment (the parked meemoo-validation-harness idea). Needs its own spec-delta plan.

- **Representation directory naming.** The `representation_([0-9]+)$` regex is stricter than either spec, and a non-matching dir (e.g. `master`) is silently skipped rather than erroring.
  - CSIP: representation folder names are free-form; only requirement is uniqueness within the package.
  - meemoo 2.0: the dir name is not fixed to `representation_1`, but MUST equal the representation METS's `mets/@OBJID`; `representation_1` is only the illustrative example.
  - Decide: (a) accept free-form names per CSIP, or (b) keep a convention but fail loudly on non-matching dirs — and either way resolve the "OBJID must == dir name" coupling for the meemoo profile. (Deferred input-side change; see the [refactoring plan](archive/refactoring-plan.md) tail.)

- **Identifier minting authority.** SIP Creator mints only package-local `uuid-<uuid>` IDs and asserts no authority over them ([ADR-0001](decisions/0001-package-builder-not-archive.md)). Still open:
  - Who mints the *package* identifier, and the identifiers of intellectual entities & representations — the producer, or a downstream identifier service? (Ref: [CSIP1](https://earkcsip.dilcis.eu/).)
  - Is a UUID meaningful only within the SIP acceptable as the common key tying description / IE / representation / file together? Where would externally-minted IDs be recorded if not?

- **Multiple descriptions / entities / formats.** How should the tool handle multiple descriptive records, sub-intellectual-entities, or multiple formats per representation? The model has the slots (`Entity.Entities`, per-file `Format`) but the `basic` profile builds only a single root entity.
  - Is the intended shape to walk an LD graph and populate the package from it (cf. [sipin-mh-sip-creator](https://github.com/viaacode/sipin-mh-sip-creator/tree/main/tests/resources))? Implications: identifiers would be minted by external services; there is no strong Go triplestore library, so this likely needs a supporting query API or a tech change. (Format characterisation is meanwhile decided: optional pre-computed sidecar input — [ADR-0009](decisions/0009-characterization-as-sidecar-input.md), superseding ADR-0006's mechanism; fixity stays native and in-process.)
  - For a bibliographic profile: source for mapping to MODS — BIBFRAME or otherwise? (Ref: [MODS–BIBFRAME mapping](https://www.loc.gov/standards/mods/modsrdf/mods-bibframe-mapping.html).)

- **New input contract (convention-based folders + key–value `metadata.csv`)** → drafted in [input-spec.md](input-spec.md); supersedes the earlier loose "CSV descriptive-metadata input" idea (multi-valued fields are solved there by repeated keys). Needs an implementation plan; that plan should be accompanied by an ADR recording the config-over-self-describing-input trade-off (organization details come from configuration, so an input folder alone does not fully determine the package). Features the spec explicitly defers (chosen against for v1, not forgotten): an explicit per-file manifest, operator-supplied descriptive XML, multiple intellectual entities per package, BagIt input, per-package overrides of configured values.

- **Library builder API — embeddability requirements.** So larger ingest-automation systems can drive the library without touching the CLI's input convention, the builder API must: accept essence as streams (`io.Reader`/`fs.FS` + logical path), not only filesystem paths; accept pre-computed fixity and only compute checksums when none are supplied; take descriptive metadata as data (structs), not files to parse; take administrative metadata (agents, submission agreement, record status, content category) as data — `sip.Spec`/`sip.Agent` (landed with [refactoring plan](archive/refactoring-plan.md) Steps 7–8) is the vehicle; accept a caller-supplied package identifier (updates reuse the original `mets/@OBJID`) and mint a UUID only when none is given; keep representation labels free-form in the model (the profile decides SIP-side directory naming); build to a destination the caller controls. See the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc.

## Validator status

**Resolved 2026-07-17** ([meemoo-12 plan](archive/meemoo-12.md)): both sample
packages report **VALID** — `basic` against E-ARK 2.0.4 (the era meemoo 1.2
builds on; its SIP2 check expects exactly the unversioned profile URL meemoo
mandates), `eark` against 2.2.0. The old SIP2 failure was a spec-version
mismatch, not a package defect; the old XSD-failure evidence stopped
reproducing under commons-ip 2.11.2.

One SHOULD-level warning remains for `basic`: **CSIPSTR16** (no `documentation`
folder). The code supports documentation for every profile — the warning is
fixture-level; add a `documentation/` dir to `tmp/basic` to clear it.
