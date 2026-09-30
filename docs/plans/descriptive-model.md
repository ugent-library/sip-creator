# Plan: the descriptive model follows its standard, and supplied documents return for the eark profiles

*Status: **S6 done, S7 next** (2026-09-30; S4 withdrawn). Drafted on the branch
`descriptive-model` from the review of 2026-09-30 that closed the
[eark-mods plan](../archive/eark-mods.md) after its S3, with
[ADR-0021](../decisions/0021-descriptive-model-follows-its-standard.md)
recording the decision. The reader takes an `input.Vocabulary`, the
profiles' vocabularies live in `cli/input/vocabulary`, `build.Definition`
has no constructor for a transport, and `earkmods.Record` is typed by
field with its key table on the CLI side; both DC profiles validate VALID
with 0 warnings and compare identical to their reference copies, and the
MODS document is pinned byte for byte. `items.csv` is withdrawn (the
review of S3, 2026-09-30): the rows carry flat statements only. The
library accepts a supplied `dc.xml` or `mods.xml` as a `build.DescriptiveDocument`
under the two eark profiles; the folder does not yet. Update this line as
steps land.*

## Context

The eark-mods plan delivered the `eark-mods` profile with a MODS
description modelled as flat statements plus a list of items. Its next
step, `items.csv` in the CLI, needed a neutral item shape in `sip/` and a
widened constructor on `build.Definition`, because the CLI's reader knows
no profile and the model is a list. That was the flat model straining, and
the review that followed ([ADR-0021](../decisions/0021-descriptive-model-follows-its-standard.md))
decided three things:

1. The model follows its standard: a typed struct where the standard is a
   tree (MODS), a list of `sip.Term` where it is flat (dc+schema, Simple
   DC).
2. The CSV owns its syntax, vocabulary and mapping on the CLI side, one
   adapter per profile. The library carries no constructor for a
   transport.
3. A supplied document is a second description type behind
   `sip.Description`, accepted by the two eark profiles and refused by
   `basic`.

This plan builds those in that order, then carries over the eark-mods
acceptance work (fixture, XML catalog, xmllint in build.sh, reference
capture) the old plan never reached.

## Shape after the plan

**Library.** `build.SourcePackage.Description` and each representation's
`Description` take a `sip.Description`, which is one of: `meemoo.Terms`
(list of `sip.Term`), `eark.Terms` (list of `sip.Term`), `earkmods.Record`
(typed fields: identifier, titles, items, later names and dates), or
`build.DescriptiveDocument` (a finished `dc.xml` or `mods.xml`, eark profiles only).
`build.Definition` has no `NewDescription`. A profile's encoder `Check`
accepts its model, and for the eark profiles also a document whose root is
its standard's.

**CLI.** `cli/input` keeps the generic reader: folder rules, the CSV
syntax, and the document file names. One adapter per
profile (`cli/input/meemoo`, `cli/input/eark`, `cli/input/earkmods`)
implements a small interface declared in `cli/input` and imports its
profile package; the reader takes the adapter. `cli/profile.go` pairs each
`--profile` value with its definition and its adapter.

**Folder.** Per level, exactly one of `description.csv` or the profile's
document (`dc.xml` under `eark`, `mods.xml` under `eark-mods`, none under
`basic`). The rows are flat statements about the level's entity; a
record's copies and any other structure travel in a supplied `mods.xml`.

## File specification (to fold into input-spec.md when shipped)

| profile | `description.csv` keys | document |
|---|---|---|
| `basic` | meemoo's dc+schema table | not accepted |
| `eark` | the fifteen Simple Dublin Core elements | `dc.xml`, root `simpledc` |
| `eark-mods` | `identifier`, `title`, later the record's other flat fields | `mods.xml`, root `mods:mods` with `version="3.7"` |

Rules, all MUST violations collected by `check`:

- At the input root, exactly one of `description.csv` or the profile's
  document. Both, or neither, is a violation. Inside a representation
  directory, at most one of the two.
- A document under a profile that takes none (`basic`) is a violation. A
  document of another standard's name (`mods.xml` under `eark`) is
  content, not a document; the profile's METS would otherwise declare a
  type the file does not have.
- A document MUST be well-formed XML with the standard's root element;
  the tool checks nothing else in it (ADR-0003). build.sh runs xmllint over
  every `mods.xml` in a package as acceptance.
- The rows carry flat statements only. A record's copies have no rows
  file: `items.csv` (ADR-0015, twelfth decision) was withdrawn on
  2026-09-30 in the review of S3, before it shipped; copies travel in a
  supplied `mods.xml`, or in the library's record.
- `description.csv` keeps today's rules: `key,value` header, UTF-8,
  `key[lang]`, unknown keys and a second row for a single-valued key are
  violations, `identifier` and `title` required at the root.

## Execution steps

Library first, output unchanged until S8 captures the eark-mods reference.
Every step ends with `go test ./...` green, `./build.sh basic` and
`./build.sh eark` VALID with 0 warnings, and the structural comparison in
scripts/reference-diff.sh clean for both against their reference copies.

Each step is a checkbox list. A box is ticked when its commit is on the
branch; every commit message is proposed in chat and committed after
approval.

### S1: docs first

- [x] This plan.
- [x] [ADR-0021](../decisions/0021-descriptive-model-follows-its-standard.md).
- [x] Notes on ADR-0011, 0015, 0016 and 0018; ADR-0017 superseded.
- [x] The eark-mods plan closed (S1 to S3 shipped, S4 to S6 superseded here)
      and moved to `docs/archive/`; the links that pointed at it follow.
      Commit `Added: ADR-0021 and the descriptive-model plan; eark-mods plan
      archived`.

### S2: the CLI adapters, model unchanged

A pure move of the row-to-description step from the library to the CLI.
The model stays the flat statement list, so this step is a refactor and
both comparisons stay clean.

- [x] **The adapter interface.** In `cli/input`, a one-method interface
      named for what it does: it gives the decoded rows of one level their
      meaning and returns the profile's description plus findings, each
      with the row's line. It takes the rows the reader decoded (key,
      language, value, line) and, for the package level, the item rows of
      an `items.csv` when one is present (S4 adds the file; the parameter
      exists from here so the signature does not change twice). The
      reader keeps the syntax rules it has: header, two columns, the
      `[lang]` shape, no prefixed keys. It keeps running `Validate` and
      `ValidateRequired` on the result and mapping a `*sip.TermError` to a
      line; a finding that is not a term error is printed with the file
      name. (Landed as `input.Vocabulary` with `Row`, `ItemRow` and
      `Finding` in `cli/input/vocabulary.go`; the name is the input
      specification's word for what the profile decides about the rows.
      Renamed in the review of S3: a row is a `Statement`, one thing the
      folder states about the level's entity, and a problem with one is a
      `*StatementError`, an error value mirroring `sip.TermError`, returned
      in a plain `[]error` next to errors about the file; `ItemRow` went
      with `items.csv`.
      `cli/profile.go` resolves the vocabulary next to the definition, so
      the commands change once. The reader's tests keep two small test
      vocabularies of their own: they call an unexported function, so
      they stay internal, and the CLI's vocabularies import the reader.)
- [x] **Three adapters.** `cli/input/meemoo`, `cli/input/eark` and
      `cli/input/earkmods`, each importing its profile package and, for
      now, wrapping the rows as the profile's `NewDescription` did. The
      earkmods adapter reports item rows as unsupported until S3 gives the
      record its fields. (Landed as one package, `cli/input/vocabulary`,
      with the types `Meemoo`, `Eark` and `EarkMods`: three packages named
      after the profiles would each have imported a profile package of
      the same name under an alias, and the pairing of names to
      vocabularies is one table either way. The package is the one CLI
      package importing the profile packages for descriptive metadata.
      Item rows were one finding about the file under every profile,
      never dropped, until `items.csv` was withdrawn in the review of S3.)
- [x] **The profile table.** `cli/profile.go` resolves `--profile` to the
      definition and the adapter; `check` and `create` construct the
      reader with `input.New(adapter)`. `build.Definition.NewDescription`
      goes, with its field doc and the registry test that called it; that
      test moves to the CLI, asserting each adapter builds what its
      profile's `Check` accepts. (Landed as `vocabulary.For(name)`, keyed
      by the definitions' own names; the moved test also pins that every
      registered profile has a vocabulary and that rows keep their order.)
- [x] **Docs.** `CLAUDE.md`: the system shape's input paragraph (the reader
      takes the profile's adapter; the generic reader imports no profile
      package, the adapters do). Design doc: the CLI/library boundary and
      the input contract paragraph. ADR-0018's note. Commit `Changed: the
      CLI's vocabularies build a profile's description from
      description.csv rows`. (The design doc's `cli/` bullet and the test
      inventories in both files name `cli/input/vocabulary` too; the
      encoder comments that named the constructor lost that sentence.)
- [x] **Acceptance.** `go test ./...`; both profiles VALID; both
      comparisons clean. (Run 2026-09-30 with `CSIP_CMD` pointing at the
      commons-ip 2.11.2 jar on host Java 27, the jar fetched from the
      release URL the validator Dockerfile pins and checked against its
      sha256; basic passed=132 and eark passed=129 with 28 skipped, the
      same counts as the reference captures.)

### S3: the typed MODS record

- [x] **`earkmods.Record` by field.** `Identifier string` (the catalogue
      number; the `type` attribute stays the constant `mmsIDType`),
      `Titles []Title` (`Value`, `Lang`), `Items []Item` as today. `Validate`
      keeps the item rules and checks each title's value and language tag;
      `ValidateRequired` requires the identifier and one title.
      `vocabulary.go` and `sip.Term` leave the package; the template walks
      the fields in a fixed order (identifier, titles, location). The
      document for an identifier, a title and two items is byte-identical
      to today's, pinned by the encoder test. (Landed as planned; the
      bytes were captured from the old template before the change and
      are the golden test's constant. `Validate` also refuses a blank
      identifier, since the field being empty means "none"; the
      one-identifier rule is gone, the type enforces it. `mmsIDType` sits
      next to the template now. The template lost its `element` func and
      the self-referencing parse: it walks fields, no key reaches it.)
- [x] **The earkmods adapter's key table.** The key table moves to
      `cli/input/earkmods`: each key names the field it fills and whether
      it may repeat (`identifier` once, `title` per language). An unknown
      key, a second `identifier` and a repeated language are file findings
      with lines. Item rows fill `Items` in row order. (Landed in
      `cli/input/vocabulary/earkmods.go` as `modsKeys`, a map from key to
      `placement`: a fill function, whether the key takes a language tag,
      and a cardinality. A statement the vocabulary refuses is not placed,
      so the record's own `Validate` does not report it a second time; the
      refused kinds are an unknown key, a language on `identifier`, an
      empty value, a second `identifier` and a repeated language, each a
      `*input.StatementError` at its line naming the line of the statement
      that was placed. Item rows are not read: `items.csv` was withdrawn
      in the review of this step, see S4. A malformed language tag stays
      the record's rule and is reported with the title's position, not a
      line; see the open question.)
- [x] **Tests.** Record validation and encoding in `profiles/earkmods`;
      the adapter in `cli/input/earkmods`; the engine cases in
      `build/assemble_test.go` construct a `Record` by field. (Plus a run
      of `check` and `create --profile eark-mods` on scratch folders: the
      valid one builds a package whose `mods.xml` equals the golden bytes
      and whose METS types it `MDTYPE="MODS" MDTYPEVERSION="3.7"`; the
      one with every refused kind of row reports each at its line.)
- [x] **Docs.** README library example (a `Record` by field); design doc
      domain-model line; the field docs on `SourcePackage.Description` and
      `sip.Description` that name the types. Commit `Changed: the MODS
      record is a typed model`. (The design doc's descriptive bullet and
      its validation paragraph, `sip.Description`'s doc and `CLAUDE.md`'s
      profile paragraph describe the typed record; the field docs on
      `SourcePackage` name `earkmods.Record` unchanged.)
- [x] **Acceptance.** As S2. (Run 2026-09-30 as in S2: basic passed=132,
      eark passed=129, 28 skipped, VALID with 0 warnings; both structural
      comparisons identical; `go test ./...` green with the golden test.)

### S4: `items.csv`

Withdrawn 2026-09-30 in the review of S3, before it shipped. A one-to-many
child table is the flat-to-rich device the CSV should not grow
([ADR-0021](../decisions/0021-descriptive-model-follows-its-standard.md)):
the rows carry flat statements about the level's entity and nothing else,
and a record's copies reach the package through the library's record, as
the ingest system supplies them, or through a supplied `mods.xml` (S6,
S7). `Record.Items`, its rules and the `copyInformation` rendering stay in
the library. The input specification's MODS keys under `eark-mods` land
with S7's docs. ADR-0015's twelfth decision and ADR-0016 carry the note.

### S5: the library as a reference implementation

Added 2026-09-30 in the review of S3. The library serves UGent Library
first, but it should be usable by other institutions as it stands and
read as a reference implementation of E-ARK packaging with meemoo's and
RODA's profiles layered on: an institution with a different descriptive
world brings its own profile. That route exists since ADR-0018 to 0020
(`build.Definition` and `build.DescriptionEncoder` are exported and
`build.New` takes any definition), but no document says so. The same
review decided that the eark-mods profile, read that way, is plain E-ARK
with a MODS 3.7 writer, and that the writer should cover the standard's
top-level elements in MODS's own words, in tiers. That is a plan of its
own, [mods-coverage](mods-coverage.md), which starts after this plan
ships the document route (S6 to S8); this step records the aim and states
it in the docs. Docs only, no code.

- [x] **Decision record.** [ADR-0022](../decisions/0022-reference-implementation-bring-your-own-profile.md),
      one page: the library is a reference implementation others can use;
      the in-tree profiles are reference implementations of the encoder
      interface and the definition, and an institution brings its own as
      a package handed to `build.New`, outside the registry, which is the
      CLI's list; where a profile would decide an attribute value for the
      caller, the value is a typed constant set the caller picks from,
      closed like the CSV vocabulary; the eark-mods profile is plain E-ARK
      with a MODS writer that speaks MODS; the eark profiles' choices that
      are RODA's (no PREMIS, the representation type in the content
      typing) are named as the reference's choices for RODA rather than
      E-ARK's rules.
- [x] **The coverage plan.** [mods-coverage](mods-coverage.md), drafted as
      the parked plan: the element set in three tiers, the common
      attributes in and out, `relatedItem` recursive, `extension` out,
      the identifier's type as the first choice a caller makes, and the
      open questions (one identifier or several, the constant sets, the
      CSV keys for new elements). Commit `Added: ADR-0022 and the
      mods-coverage plan; the library is a reference implementation`.
- [x] **UGent as the example, not the rule.** The comments in
      `profiles/earkmods` that name the Alma MMS ID and "the owner of the
      repository side" describe the record's local identifier and give
      UGent's as the example. `CLAUDE.md`'s audience section names other
      institutions and the reference-implementation aim. README: the
      introduction states the aim, and the library section gains a
      paragraph on bringing your own profile (a description type
      implementing `sip.Description`, an encoder implementing
      `build.DescriptionEncoder`, an exported `build.Definition`, handed to
      `build.New`). Design doc: the CLI/library boundary section states
      the aim and names the eark profiles' RODA choices as such. Commit
      `Changed: docs state the reference-implementation aim and the
      bring-your-own-profile route`. (The constant `mmsIDType` is
      `localIdentifierType` now, the one code change: its old name was the
      UGent word for a MODS value.)

### S6: supplied documents in the library

- [x] **`build.DescriptiveDocument`.** A finished descriptive document supplied as a
      file: its path, and after `Validate` its root element. `Validate`
      reads the file once and requires well-formed XML; `ValidateRequired`
      is a no-op. The root reader is lifted out of the received-PREMIS
      check into one function both call. (Landed as `build.DescriptiveDocument{Source}`
      with `Validate`, `ValidateRequired` and `Root()`, which the engine
      calls for the profile's check. The type stays a plain value and
      caches nothing: the file is read once for the root and once when the
      writer copies it, which a small document makes cheap. The shared
      reader is `encoders/xmldoc.Root`, with `Attr` for the version
      attribute; the received-PREMIS check calls it and keeps its own
      messages.)
- [x] **Encoders.** `eark` and `earkmods` accept their model or a document
      whose root is their standard's (`simpledc` without namespace;
      `mods:mods` in the MODS v3 namespace with `version="3.7"`), and
      `Encode` streams the file for a document. `basic` refuses a document
      with a message naming the swap. `Schemas()` is unchanged: the package
      ships the standard's schema whatever the document points at.
      (Revised in review before the commit: the first cut had the two
      encoders type-switch on the document in `Check` and stream it from
      `Encode`, which made `Check` do file I/O and `Encode` ignore its
      schemas argument. Landed instead as `build.DescriptiveDocumentChecker`, the
      optional interface next to `IdentifierSwapper`, with
      `CheckDescriptiveDocument(root)` on the two eark encoders, a pure judgement of
      the root element; the engine reads the root and refuses a document
      for an encoder without the interface, which is how `basic` refuses;
      the writer copies a document with `store.CopyFile` where it would
      render a model. `Check` and `Encode` see models only. The MODS
      namespace and version are two constants the template and the check
      share; a MODS document without a version attribute is refused too,
      since the METS declares one.)
- [x] **Tests.** A `dc.xml` handed to `eark` and a `mods.xml` to
      `eark-mods` land under `metadata/descriptive/` with fixity and the
      right `dmdSec` typing; a `dc.xml` handed to `eark-mods`, any document
      to `basic`, a malformed file and a MODS document of another version
      are refused before any write; a document on a representation lands
      in that representation's METS. (In `build/document_test.go`, plus
      an `oai_dc` wrapper refused under `eark`, a missing file, a MODS
      document without a version and `DescriptiveDocument.Validate` on its own; the
      copied bytes are compared in the build test; `encoders/xmldoc` has
      its own table.)
- [x] **Docs.** Design doc (domain model, build lifecycle, validation);
      README library example (a `DescriptiveDocument` variant). Commit `Added: a
      supplied descriptive document is a second description type for the
      eark profiles`. (Also the design doc's `build/` and `encoders/`
      bullets, the write-phase items, `CLAUDE.md`'s encoders and domain
      model lines, the `sip.Description` doc and the field docs on
      `SourcePackage`.)
- [x] **Acceptance.** As S2. (Run 2026-09-30 as in S2: basic passed=132,
      eark passed=129, 28 skipped, VALID with 0 warnings; both structural
      comparisons identical.)

### S7: supplied documents in the folder

- [ ] **File names and rules.** The adapter names the document it accepts
      (`dc.xml`, `mods.xml`, or none). The reader treats that name as
      reserved at the root and inside each representation directory, and
      applies the one-per-level rule of the file specification above.
- [ ] **Tests.** Each rule, under each profile.
- [ ] **Docs.** Input spec §1, §3, §7 and §8 (the deferred item becomes
      current for the eark profiles); README Input section. Commit `Added:
      dc.xml and mods.xml in the input folder`.
- [ ] **Acceptance.** As S2.

### S8: eark-mods acceptance

Carried over from the eark-mods plan's S6.

- [ ] **Fixture.** `tmp/eark-mods/`: a copy of `tmp/eark` (its
      `documentation/` included, which keeps CSIPSTR16 satisfied) whose
      package-level description is a supplied `mods.xml` carrying an
      identifier, a title and two copies, one with an enumeration, and
      whose representation carries a `description.csv` with a `title[nl]`,
      so the acceptance run exercises the document route and the rows
      route in one package. (Revised 2026-09-30 with the withdrawal of
      `items.csv`; needs S6 and S7 first.)
- [ ] **XML catalog.** `scripts/schema-catalog.xml` rewriting the two
      loc.gov URLs the MODS schema imports
      (`http://www.loc.gov/standards/xlink/xlink.xsd`,
      `http://www.loc.gov/mods/xml.xsd`) onto the bundled copies in
      `schemas/`, so xmllint runs with `--nonet`.
- [ ] **build.sh.** An explicit `eark-mods` case (E-ARK 2.2.0); after
      `create`, `xmllint --noout --nonet --schema <pkg>/schemas/mods-3-7.xsd`
      over every `mods.xml` in the package with `XML_CATALOG_FILES`
      pointing at the catalog, a failure exiting non-zero like an INVALID
      package. xmllint joins the documented requirements in the script
      header, README and `CLAUDE.md`.
- [ ] **Run it.** `./build.sh eark-mods` VALID with 0 warnings and the
      xmllint pass clean; the package METS `dmdSec` read by hand
      (`MDTYPE="MODS" MDTYPEVERSION="3.7"`). Capture the package as
      `tmp/reference/eark-mods/pkg` and note it in `tmp/reference/README.md`.
      Commit `Added: eark-mods fixture, XML catalog and xmllint pass in
      build.sh`.

### S9: closing docs

- [ ] README (three profiles, both routes); design doc (status line,
      package layout, validation section naming the xmllint pass);
      `CLAUDE.md` development commands ("all three profiles validate
      VALID"); `docs/TODO.md` (the stale `profiles/descriptive.go` pointer
      in the meemoo 2.x item); ADR-0021 to Accepted with the date; this
      plan's status line to shipped, then the plan moves to
      `docs/archive/`. Commit `Changed: docs for the descriptive model;
      plan archived`.

## Open questions

- **The `type` attribute on `mods:identifier`**: one constant, `local`,
  the value MODS suggests for an identifier local to the describing
  institution's system. It becomes the caller's choice from a closed set
  in the [mods-coverage plan](mods-coverage.md)'s first tier (decided
  2026-09-30; it had been "decided by the owner of the repository side").
- **Further MODS fields.** Names (personal and corporate, with a relator
  code per role) and dates (`encoding="edtf"`) as fields on the record,
  each with a template line, an adapter key and a spec line; supplied with
  the repository's index fields (carried over). A record richer than the
  fields is a supplied `mods.xml`.
- **Items on a representation-level record.** The library has no
  per-level hook, so a caller who puts items on a representation's record
  gets them rendered there; the CLI refuses `items.csv` inside a
  representation directory. Left as is (carried over from 2026-09-29).
- **The eark profiles' RODA choices.** `eark` and `eark-mods` emit no
  PREMIS, because RODA drops package PREMIS without agents or events, and
  carry the representation type in the content typing, because RODA reads
  it there (ADR-0013). Neither is an E-ARK rule. A generic E-ARK consumer
  may want PREMIS. Whether those choices become data a caller sets on the
  definition, or a fourth profile, is decided when such a consumer
  appears; S5 names them as the reference's choices for RODA.
- **A malformed language tag on a MODS title has no line.** The tag's
  shape is the record's rule (`Validate`), and the record names the title
  by position ("title 2"), which the reader prints with the file name but
  no line. The flat worlds report the same finding with a line through
  `*sip.TermError`. Giving it a line means either checking the tag's shape
  in the reader for every profile (a syntax rule next to the `[lang]`
  bracket check) or repeating the check in the MODS vocabulary. Left as
  is on 2026-09-30; the position is enough to find the row.
