# TODO

The live backlog. Items that a plan or ADR now owns point there rather than
sitting as loose unknowns — see [plans/](plans/) and [decisions/](decisions/).
The system as it exists today is described in [sip-creator-design.md](sip-creator-design.md);
its rough edges are collected under [Known gaps](sip-creator-design.md#known-gaps).

## Owned by a plan or ADR (scheduled, not yet shipped)

These are decided and sequenced; remove each line when the work lands.

- **Identifier uniqueness across the package graph** → an invariant for the future `sip.Package.Validate()` (domain validation lives on the `sip/` types per `CLAUDE.md`; post-Phase-2, once the graph API settles). Decided when Step 7 removed the inert METS `idStore` (it never recorded a minted ID, so it guaranteed nothing): keep minting UUIDv4 with no minting-time machinery, and make emitting a duplicate impossible instead — `Validate`, called between assemble and write, collects every identifier in the graph into a set and fails the build on a duplicate. The real target is *systematic* duplication (a node wired into the graph twice, a copied identifier), which no random-collision defence catches; the ~10⁻²⁵ UUIDv4 collision comes along for free, and commons-ip's XSD check on `xs:ID` remains the document-level net.
- **`mets/@TYPE` should draw from a vocabulary, not a hardcoded literal** → the value is now data in the `basic` registry entry (`profiles/definition.go`), so this is a one-line fix once the right CSIP content-category value is chosen. Ref: [CSIP2](https://earkcsip.dilcis.eu/#CSIP2).
## Open design questions (not yet owned)

These need a decision before they become plan work.

- **Batch format identification per representation.** `Identify` spawns one `sf` process per essence file, and every spawn reloads the ~10MB signature file (~100–200ms) — irrelevant for a 3-file SIP, a minute of pure reloading for a 500-file batch. Fixable inside the exec model: `sf` accepts multiple paths (or a directory) per invocation and returns one JSON with a `files[]` array, so the siegfried adapter could identify a representation's files in one spawn. Internal to the adapter, or at most a second optional batch interface the assembler detects. Do when a real multi-hundred-file use case appears, not before. (Exec-vs-library was weighed 2026-07-16: exec stays for the CLI; an embedded-library identificator — `siegfried.Load` once, `Identify(io.Reader)` per file — becomes the right *additional* registry entry when the stream-first library-embeddability work lands, not before: it would trade the operational binary dependency for a signature-currency and bus-factor dependency in `go.mod`.)

- **Format-identification provenance as a PREMIS event.** We record *what* was identified but not *who/when/how*. Proper preservation practice is a PREMIS event ("format identification", agent: siegfried + version, signature file + date) — what a future archivist actually needs to trust the format claim. Blocked on `sip.Event` (an empty stub) growing up; when events are modeled, this should be the first one emitted. Raises the feature from "enriches a SHOULD field" to production-grade provenance. No event design exists yet anywhere — picking this up needs an ADR covering the event shape (LoC eventType vocabulary, dateTime, detail, linking identifiers), where events attach on the graph (per-file vs. package-level), PREMIS agents vs. the METS-specific `sip.Agent`, and pass-through of *received* vendor events ([input-spec.md](input-spec.md)) alongside self-generated ones.

- **meemoo SIP 2.1 migration.** The repo targets 2.0 throughout; UGent will eventually move to [2.1](https://developer.meemoo.be/docs/diginstroom/sip/2.1/). Conceptually a *family-level* change (vocabularies, encodings, values) — likely a versioned meemoo family alongside 2.0 rather than in-place edits, and the designated promotion trigger for the `Family` constant growing into an internal struct ([ADR-0007](decisions/0007-profile-families-share-one-writer.md)). Needs its own spec-delta plan; deliberately parked (2026-07-16) until the [eark profile](plans/eark-writer.md) lands.

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

- **Library builder API — embeddability requirements.** So larger ingest-automation systems can drive the library without touching the CLI's input convention, the builder API must: accept essence as streams (`io.Reader`/`fs.FS` + logical path), not only filesystem paths; accept pre-computed fixity and only compute checksums when none are supplied; take descriptive metadata as data (structs), not files to parse; take administrative metadata (agents, submission agreement, record status, content category) as data — `sip.Spec`/`sip.Agent` (landed with [refactoring plan](plans/refactoring-plan.md) Steps 7–8) is the vehicle; accept a caller-supplied package identifier (updates reuse the original `mets/@OBJID`) and mint a UUID only when none is given; keep representation labels free-form in the model (the profile decides SIP-side directory naming); build to a destination the caller controls. See the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc.

- **METS `@MIMETYPE` for essence files is a hardcoded lie.** Representation METS stamps every essence file `MIMETYPE="text/xml"` (`encoders/mets/encoder.go:75-78`), and CSIP makes `@MIMETYPE` a MUST. The identificator's mime output (parsed and currently discarded) is a candidate source when identification runs; an extension-based fallback is needed when it doesn't. Decide the source of truth, then fix.

## Known-INVALID: concrete validator evidence

The sample package does not yet validate. Keep these until fixed (they are the
gate that must go green before blocking automation — [ADR-0003](decisions/0003-validation-stays-external.md)).

Current failures (commons-ip 2.11.2, spec 2.2.0; re-measured 2026-07-16):

- **SIP2 [MUST]** — `mets/@PROFILE` declares the CSIP profile URL; the validator requires `https://earksip.dilcis.eu/profile/E-ARK-SIP.xml`. Open check: whether meemoo 2.x itself mandates the E-ARK SIP URL — if yes this is a one-line registry fix for `basic` (surfaced by the [eark-writer plan](plans/eark-writer.md) research).
- **CSIPSTR16 [SHOULD]** — no `documentation` folder. The [eark-writer plan](plans/eark-writer.md) adds documentation support for the eark profile; whether `basic` should emit it too is open.

(Older evidence here — `dc+schema.xml` failing XSD validation with `cvc-elt.1`/EDTF errors — no longer reproduces under commons-ip 2.11.2 and was removed 2026-07-16.)
