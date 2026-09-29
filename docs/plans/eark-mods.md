# Plan: MODS 3.7 descriptive metadata for the plain E-ARK output

*Status: **S2 done, S3 next** (2026-09-28): the refactor is on the branch
`eark-mods-support`. Go tests, the structural comparisons of both profiles
against their reference copies, and the commons-ip runs of `./build.sh`
for both profiles (VALID, 0 warnings) are clean. The supplied-document
route (decisions 6, 7, 8 and S5) was withdrawn on 2026-09-28 before it was
committed ([ADR-0017](../decisions/0017-supplied-descriptive-document-deferred.md));
descriptive metadata arrives as terms only. ADR-0015 and ADR-0016 are
drafted, including the items table (decision 12). The MODS element list
beyond identifier and title is still open and does not block S3. Update
this line as steps land.*

## Context

The tool emits one descriptive document per profile: meemoo's
`dc+schema` document for the `basic` profile, a simple Dublin Core document
for the `eark` profile. Bibliographic material at UGent Library needs the
plain E-ARK output to carry [MODS 3.7](https://www.loc.gov/standards/mods/)
instead, so the repository that ingests it can index title, creator and
dates from a bibliographic record rather than from fifteen flat DC elements.

Two routes must work:

1. **The library.** Systems that automate ingest workflows hold descriptive
   metadata as flat columns (an identifier, a title, ...) and construct
   `profiles.Input` directly. They need to hand the library terms and get a
   MODS document back, the way they hand it DC terms today. This is the main
   route and ships first.
2. **The CLI.** The input folder today takes one `metadata.csv` that always
   becomes Dublin Core. It must also take rows that become MODS. (A
   finished `dc.xml` or `mods.xml` prepared elsewhere, copied into the
   package after a check of its shape, was part of this route until
   2026-09-28; it is deferred, see decision 6.)

What already fits: a family exists to select the descriptive encoding and
that is the only thing it does today ([ADR-0007](../decisions/0007-profile-families-share-one-writer.md));
the METS typing of the descriptive document (`MDTYPE`, `MDTYPEVERSION`) is
profile data; [ADR-0011](../decisions/0011-closed-descriptive-vocabulary.md)
already states that a MODS family brings its own vocabulary table.

What does not fit: the descriptive model is a flat list of terms where each
term names the one element it emits, and MODS is nested. A MODS title is
`titleInfo/title`; a name carries parts and a role. The domain model and the
library input are typed on the DC terms type, and the identity checks name
the DC identifier element literally. The CLI reader resolves CSV keys against
the DC table while walking the folder, before any profile is known.

## Design decisions (agreed 2026-09-15)

1. **Separate descriptive worlds, no shared term type.** Each world owns
   its terms type, closed vocabulary table, validation and templates.
   There are three (revised 2026-09-23; the review of 2026-09-15 had one
   DC world with two templates): `profiles/meemoo` for meemoo's
   `dc+schema` document (Dublin Core terms plus schema.org, EDTF typing,
   cardinality and language rules), `profiles/eark` for Simple Dublin Core
   (fifteen elements), and `profiles/earkmods` (paths as of 2026-09-28,
   [ADR-0018](../decisions/0018-engine-and-profile-packages.md): each world
   lives in its profile's own package next to the profile's `Definition`;
   they were `encoders/dcschema`, `encoders/dc` and `encoders/mods`
   before). Meemoo's document and Simple
   DC share only the Go shape of a term; the DCMI dumb-down that derived
   the eark document from meemoo's vocabulary is removed with the split,
   because it silently dropped keys with no Simple DC parent
   (`artmedium`, `artform`, `rightsholder`), the lossy mapping ADR-0011
   rejects. Rejected: one neutral term type keyed by plain vocabulary key,
   shared by all worlds. It would have unified the CSV and the library on
   one key language, but a MODS term produces a subtree while a DC term
   produces an element, and forcing both through one type hides that
   difference. (Narrowed 2026-09-28: the statement shape is shared as
   `sip.Term`, keyed by the plain key; each world keeps its own `Terms`
   type, table, rules and template. Only the struct is shared, not the
   methods, so the objection does not apply. ADR-0015 carries the note.)
2. **One key emits one complete MODS element with fixed attributes.** The
   MODS table maps a plain key to a template fragment plus the attribute
   values it needs (`identifier` becomes `mods:identifier` with a `type`
   attribute; `title` becomes `mods:titleInfo/mods:title`). Operators and
   library callers never set attributes, as today (ADR-0011). Roles, title
   types and identifier types are rows (`author`, `promotor`, `alternative`,
   `isbn`), names come in personal and corporate rows, and dates are single
   EDTF values (revised 2026-09-22, recorded in ADR-0015). Consequence: the
   flat model cannot express several parts under one parent: given and
   family name parts, a name with two roles, or a subject with several
   sub-terms. That is acceptable because the source data is flat columns.
   A richer record has no route into the package until the table grows a
   row for its shape or the supplied-document route returns (decision 6).
3. **The profile fixes the descriptive standard.** The input supplies
   terms to encode in that standard. MODS terms given to a DC profile are
   a build error before any disk write, and the reverse likewise. (Until
   2026-09-28 the input could also supply a ready document of the
   standard; see decision 6.) Rejected: letting the input decide the
   standard and the profile follow. It would make one profile emit two different documents
   depending on what it was given, and `MDTYPE` would stop being profile
   data.
4. **Each profile names its descriptive standard directly** (revised
   2026-09-23; the review of 2026-09-15 had three families behind a
   `Family` constant). `profiles/descriptive.go` holds the closed set of
   standards: `meemooDC` (the `dcschema` world), `simpleDC` (the `dc`
   world) and `mods` once it exists, each a type check on the input's
   descriptions plus the encoder, built from its own package with no
   shared constructor. A registry entry names one in an
   unexported field. (Revised 2026-09-28, ADR-0018: the standard is the
   exported interface `build.DescriptionEncoder`, implemented by an
   unexported type in each profile package next to its exported
   `Definition`; the registry in `profiles/` closes the set.) Profiles
   stay what operators type: `basic`, `eark`
   (name kept) and the new `eark-mods`, whose entry copies `eark` with
   `descriptive: mods`, `MDTYPE="MODS"`, `MDTYPEVERSION="3.7"`, `mods.xml`
   as the document name, identifier and title required, no cardinality or
   language rule, no PREMIS. The `Family` constant is retired: with one
   profile per standard it named the same thing twice, and nothing
   consumed the data-only `Definition` it protected (ADR-0007 superseded
   in that one respect). Meemoo's OR-id rule becomes the data flag
   `RequireSubmitterORID`.
5. **A small interface in the domain model.** `sip/` declares
   `Description` with `LocalIdentifier() string`, `Validate() error` and
   `ValidateRequired(keys ...string) error` (the third added
   2026-09-23 so a profile checks its required keys without knowing
   the terms type); every terms type implements it, and `sip/` stops
   importing an encoder package. The profile's descriptive standard asserts the concrete type at
   build time (decision 3). The identifier swap stays a method on the
   meemoo terms type in `encoders/dcschema`, the one world that has it.
   Required elements become plain vocabulary keys every world's table
   resolves: identifier and title, the identity every package states,
   are checked once in `Input.Validate` (`profiles.ValidateIdentity`,
   reused by `check`); a `Definition` lists only what its own spec adds
   (revised 2026-09-24). Revised again 2026-09-28 on review of S2's
   result: the interface is `Validate` and `ValidateRequired()`; what a
   package-level description must state is each world's own list,
   checked in `Input.Validate` and by `check`, so `Definition` names no
   descriptive elements; the swap sits behind the meemoo standard, in
   `profiles/meemoo` since the layout change of the same day; the
   encoders trust validated terms. ADR-0015 carries the note.
6. **Supplied documents reuse the essence path.** Withdrawn 2026-09-28
   ([ADR-0017](../decisions/0017-supplied-descriptive-document-deferred.md)).
   The route (a `DescriptiveDocument` path next to `Descriptive` on the
   input and on each representation, copied into the package like essence
   after a check of its root element) was implemented in the working tree
   and discarded before commit. The cost was not the check of the document
   but the second input path it opened at every level. Descriptive
   metadata arrives as terms only until the route returns; its design
   stays recorded in ADR-0016.
7. **Structural checks only on supplied documents.** Withdrawn with
   decision 6. What stays: no XSD validation in the tool
   ([ADR-0003](../decisions/0003-validation-stays-external.md)), and
   build.sh runs xmllint over the emitted `mods.xml` as acceptance (S6).
8. **No identity enforcement on supplied documents.** Withdrawn with
   decision 6.
9. **CLI file names by standard.** `dcschema.csv` (meemoo's dc+schema
   vocabulary), `dc.csv` (Simple DC) and `mods.csv` hold rows, at the
   package root and inside each representation directory (`dc.xml` and
   `mods.xml` as supplied documents went with decision 6). `items.csv`
   (MODS only, package root only, next to `mods.csv`) holds one row per
   physical copy, see decision 12.
   One source per level, one standard per folder. The folder stays
   self-describing, so `check` keeps taking no configuration
   ([ADR-0010](../decisions/0010-config-over-self-describing-input.md));
   a mismatch with the chosen profile surfaces at `create`. `metadata.csv`
   is withdrawn: its presence is a violation telling the operator to rename
   it to `dcschema.csv` (meemoo) or `dc.csv` (plain E-ARK). The name
   `dcschema.csv` was chosen 2026-09-23 over `meemoo.csv` because it names
   what the rows are, mirroring the emitted `dc+schema.xml`. Rejected:
   keeping `metadata.csv` and letting the profile
   decide its meaning, which forces a profile flag onto `check`. Rejected:
   one merged key space serving both standards, which reintroduces the
   silent lossy mapping ADR-0011 removed.
   (Superseded 2026-09-28, later: one name, `description.csv`, under every
   profile, and `--profile` on `check` as on `create` says which
   vocabulary it is in. The rejected profile flag on `check` is what was
   chosen after all: ADR-0010 puts the profile on the command line, and
   its config rule is about `.env`, which `check` still does without.
   `dcschema.csv` and `dc.csv` are withdrawn names and `mods.csv` will
   not exist; `items.csv` stays, allowed only under the profile that
   reads it. ADR-0016 carries the note.)
10. **Schema set as profile data.** `mods-3-7.xsd` joins the bundle (it
    imports `xlink.xsd` and `xml.xsd`, already bundled). A `Definition`
    lists the XSD files its packages ship, concatenated from the lists
    the encoders export for the schemas their documents point at (revised
    2026-09-24; the first cut kept the names on the profile). `basic` and
    `eark` list exactly the eleven files they ship today, so their output
    is unchanged; `eark-mods` lists the METS set plus the MODS list.
11. **The MODS table starts with two rows.** `identifier` and `title`, the
    two columns known to be present in every record. Further rows are data:
    a table row, a template fragment when the shape is new, and a line in
    the input specification. The owner of the repository side supplies the
    list together with the fields the repository must index.
12. **A record's physical copies are a second flat table** (agreed
    2026-09-23, recorded in ADR-0015). One package is one bibliographic
    record, and a record has several copies: each a call number, usually a
    barcode, and for journals, periodicals and newspapers a volume or issue
    designation. A `key,value` file cannot pair those; `items.csv` with the
    columns `callnumber` (required), `barcode` (optional, unique across
    rows) and `enumeration` (optional) can, one row per copy, rendered as
    one `location/holdingSimple/copyInformation` each. In the library the
    MODS description type is a struct of terms plus items. `items.csv`
    accompanies `mods.csv` only: the DC world has no place to pair them.
    Descriptive
    metadata stays at package level; a representation is a CSIP rendition
    of the same content, never a copy or a volume. Rejected: one
    representation per copy (misuses the CSIP concept; neither E-ARK
    specification models items); pairing inside `mods.csv` by order, index
    or delimiter.

## File specification (to fold into input-spec.md when shipped)

Descriptive metadata for the package lives in exactly one of these files at
the input root; the same rule applies inside each representation directory,
where the file is optional:

| file | contents | standard |
|---|---|---|
| `dcschema.csv` | rows, `key[lang],value`, keys from meemoo's dc+schema table | meemoo dc+schema |
| `dc.csv` | rows, `key[lang],value`, keys from the Simple Dublin Core table | Simple DC |
| `mods.csv` | rows, `key[lang],value`, keys from the MODS table | MODS |

(`dc.xml` and `mods.xml`, finished documents, were two more rows of this
table until 2026-09-28; deferred with decision 6.)
(Since later on 2026-09-28 the three rows are one file, `description.csv`,
whose vocabulary the `--profile` flag decides; the rows above say what it
holds under each profile. Of the rules below, the first three and the
last are moot: the profile flag decides, and the former names are not
reserved.)

`items.csv` MAY accompany `mods.csv` at the input root, one row per physical
copy, with the columns `callnumber`, `barcode` and `enumeration`.

Rules, all MUST violations collected by `check`:

- More than one of the three files at one level is a violation.
- All descriptive files in one input folder speak the same standard; files
  of two different standards anywhere in the folder are a violation.
- `metadata.csv` is a violation; rename it to `dcschema.csv` (meemoo) or
  `dc.csv` (plain E-ARK).
- Row files follow today's `metadata.csv` rules: `key,value` header, UTF-8,
  unknown keys are violations, repeat a key for multiple values, `[lang]`
  suffix for the language. At package level `identifier` and `title` are
  required.
- `items.csv` follows the `representations.csv` rules: UTF-8, a header
  naming `callnumber` and optionally `barcode` and `enumeration` in any
  order (case-insensitive; an unknown or repeated column is a violation),
  `callnumber` non-empty on every row, a `barcode` value unique across
  rows. Next to `dc.csv` or `dcschema.csv`, or inside a representation
  directory, it is a violation.
- The profile chosen at `create` must match the folder's standard: `basic`
  takes meemoo dc+schema, `eark` takes Simple DC, `eark-mods` takes MODS.

## Execution steps

Library first. Every step ends with `go test ./...` green and
`./build.sh basic` and `./build.sh eark` VALID with 0 warnings.

Each step is a checkbox list. A box is ticked when its commit is on the
branch; the acceptance line at the end of each step is run before the
step counts as done. Every code step keeps the two existing profiles
unchanged: the structural comparison in scripts/reference-diff.sh must be
clean for `basic` against `tmp/reference/pkg` and for `eark` against the
copy captured in S2.

### S1: docs first

- [x] This plan.
- [x] ADR draft `0015-descriptive-worlds-dc-and-mods.md` (decisions 1, 2,
      4, 5, 12).
- [x] ADR draft `0016-descriptive-input-rows-or-supplied-document.md`
      (decisions 3, 6, 7, 8, 9, and the `items.csv` file rules of 12).

### S2: pure refactor, output unchanged

Steps 2 and 3 go back to back: step 2 leaves a type assertion in the
assembler that step 3 turns into a guaranteed one. The two steps added on
2026-09-23 (input files named by standard, then the split of the DC
package) change the input folder's file names, so S2 is no longer purely
internal, but the emitted packages stay unchanged.

- [x] **Capture the `eark` reference.** `./build.sh eark` VALID, then copy
      the package directory to `tmp/reference/eark/pkg` next to the `basic`
      copy and note it in `tmp/reference/README.md`. Run `./build.sh` for
      `basic` and check both comparisons are clean before any code moves.
      Nothing to commit; `tmp/` is untracked.
- [x] **Rename `encoders/metadata` to `encoders/dc`.** `git mv` the
      directory, change the package clause in its files, fix the importers
      (`cli/input/input.go`, `cli/input/metadata.go` and their tests,
      `sip/entity.go`, `sip/representation.go`, the `profiles/` files and
      tests). Update the package name in `docs/sip-creator-design.md` and
      `CLAUDE.md`. Commit `Changed: encoders/metadata renamed to
      encoders/dc`; the message records the import break for library
      callers.
- [x] **The `Description` interface in `sip/`.** New `sip/description.go`
      with `LocalIdentifier() string` and `Validate() error`.
      `Entity.Description`, `Representation.Description`,
      `Input.Descriptive` and `SourceRepresentation.Descriptive` take the
      interface; `sip/` drops its encoder import. `Input.Validate` checks
      nil instead of length. The swap branch in `assembleDescriptive`
      asserts `dc.Terms`, with a comment naming the descriptive-standard
      check of the next step as the guarantee. Pitfall: a nil `dc.Terms` stored in the
      interface is a non-nil interface, so the mapping onto
      `profiles.Input` in `cli/input` assigns a representation's terms only
      when a file was read. Commit `Changed: the domain model describes
      entities through a Description interface`.
- [x] **Each profile names its descriptive standard.** (Revised
      2026-09-23 from "`Family` resolves to a struct": with one profile per
      standard the constant added a name without fan-in, see decision 4.)
      `profiles/descriptive.go` holds the `descriptive` struct (a check
      that the input's descriptions have the standard's type, plus the
      encoder taking a `sip.Description`) and the values `meemooDC` and
      `simpleDC` (an `acceptsDocument` flag was to join the struct in the
      supplied-document step, since withdrawn). `Definition` gains
      the unexported `descriptive` field and `RequireSubmitterORID`, which
      replaces the meemoo family test in `WithSubmitter`; `Family` and its
      constants are removed. `Builder.Build` refuses a definition without a
      standard, runs the standard's check on the input, then validates and
      assembles. Tests: a fake `Description` type, and a definition without
      a standard, each fail the build before any write. ADR-0007's status
      notes the partial supersession. Commit `Changed: each profile names
      its descriptive standard directly`.
- [x] **Input files named by standard, first part.** (Added 2026-09-23:
      once the two DC worlds have different tables, the CLI cannot decode
      one `metadata.csv` for both profiles, because the folder does not
      know the profile; ADR-0010.) `cli/input` reads `dcschema.csv`
      (meemoo rows) and `dc.csv` (Simple DC rows) at the root and inside
      each representation directory, exactly one per level; both still
      decode through the one current table in this step. `metadata.csv`
      becomes a violation telling the operator to rename it to
      `dcschema.csv` or `dc.csv`; `dcschema.csv` next to `dc.csv` anywhere
      in one folder is a violation (the one-standard rule). The reader
      tracks the folder's vocabulary for that rule; it lands on `Package`
      in the next step, where the decoder choice first reads it.
      `metadata.go` and its test are renamed `rows.go`. Fixtures:
      `tmp/basic/metadata.csv` becomes `dcschema.csv`;
      `tmp/eark/metadata.csv` and `tmp/eark/representations/master/metadata.csv`
      become `dc.csv`; the `cli/input` test fixtures follow. Input spec §1
      and §3, README Input section. Output unchanged for both profiles.
      Commit `Changed: metadata.csv becomes dcschema.csv or dc.csv`.
- [x] **Split the DC package into the meemoo and simple DC worlds.** (Added
      2026-09-23, decision 1. Done the same day; `Description` gained
      `ValidateRequired` so the required-elements check needs no type
      assertion, and `check` now reports a missing Dutch entry in a
      `dcschema.csv`, since that rule is the meemoo world's own.) `encoders/dcschema` takes the current table
      minus its `SimpleDC` column, the cardinality and required-language
      validation, the `dc+schema` template and the identifier swap.
      `encoders/dc` keeps only Simple Dublin Core: a fifteen-element table,
      a small terms type with `Validate` and `LocalIdentifier`, the
      `simpledc` template, still without `xml:lang` so the output stays
      unchanged (the bundled schema allows it; emitting it is a later,
      deliberate output change). `dumbDown` and the `SimpleDC` column are
      deleted. `Definition` loses `RequiredLang` and `EnforceCardinality`:
      those rules are the meemoo world's own and run in its validation;
      `RequiredKeys` stays profile data. `profiles/descriptive.go` builds
      `meemooDC` from `dcschema` and `simpleDC` from `dc`, no shared
      constructor (`fromDCTerms` goes): one struct literal per world, or
      one helper keyed by the terms type. `cli/input` decodes
      `dcschema.csv` into `dcschema.Terms` and `dc.csv` into `dc.Terms`;
      `Package.Descriptive` becomes a `sip.Description`. Fixture:
      `tmp/eark` rows become Simple DC keys (`abstract` → `description`,
      `license` → `rights`); the emitted `dc.xml` is byte-identical, so
      the comparison stays clean. Tests split with the packages;
      `TestBuildRejectsDescriptionOfAnotherStandard` gains meemoo terms
      handed to `eark`. Design doc and `CLAUDE.md` follow (three encoder
      packages; the meemoo rules inside their world). Commit `Changed:
      meemoo dc+schema and Simple DC are separate descriptive worlds`.
- [x] **Required elements become keys.** `Definition.RequiredElements`
      becomes `RequiredKeys`: `identifier` and `title` for `eark`, the
      meemoo table's required keys for `basic`. Each world's
      `ValidateRequired` resolves keys through its own table; a key the
      table does not know is reported, since it can only be a mistake in
      a profile definition. A meemoo finding names the key and the element
      it emits (`description (dcterms:description) is required but
      missing`), a Simple DC finding the key alone. Adjust the messages
      and the three tests that name elements. Commit `Changed: required
      descriptive elements are named by vocabulary key`. (Reversed
      2026-09-28: the required list is each world's own again, `required`
      in its encoder package rather than `Definition` data, spelled as
      elements; the finding keeps naming key and element. The keys had
      pulled `ResolveKey` into the library API for the CLI's sake.)
      (Reversed again later that day: terms are keyed by the plain key
      (`sip.Term`), `ResolveKey` is gone, and a finding names the key
      alone in both worlds: `description is required but missing`.)
- [x] **The identity rule is stated once.** (Added 2026-09-24 after
      review of the box above: requiredness was spelled three times, as
      a `Required` column in the meemoo table that only `basic` read, as
      a key literal on `eark`, and as element names in the CLI's identity
      check.) `profiles.ValidateIdentity(d sip.Description)` requires the
      keys `identifier` and `title`; `Input.Validate` applies it in place
      of the identifier-only check, and `cli/input` calls it in place of
      its own `requireIdentity`, so `check` and `create` use the same
      words and the CLI knows no element names. `Definition.RequiredKeys`
      lists only what the profile's spec adds: `description` and
      `created` for `basic`, nothing for `eark` (and later `eark-mods`).
      The `Required` column and `dcschema.RequiredKeys()` go. A registry
      test asserts every profile's keys resolve in its own world and pins
      the two sets. Output unchanged. Commit `Changed: the identity rule
      is stated once; profiles list only what their spec adds`. (Revised
      2026-09-28: the rule is stated once per world, as the first two
      entries of its required list; `Input.Validate` and `cli/input` call
      the same `ValidateRequired`, so `profiles.ValidateIdentity` is
      gone and a profile lists nothing.)
- [x] **Schema list on `Definition`.** A `Schemas` field listing file
      names; both profiles list the eleven files in `schemas/` today. The
      assembler builds the schema nodes from the list, sorted as now; the
      writer looks each up in the bundle. A test asserts every listed name
      exists in the bundle. (Done 2026-09-24. On review the names moved
      to the encoders: each exports the schemas its document points at
      plus their imports (`mets.Schemas`, `dcschema.Schemas`,
      `dc.Schemas`), the profile concatenates them, and the assembler
      ships each name once and refuses one the bundle does not hold
      before any write. `eark` concatenates the dcschema list too, so its
      output stays unchanged; see the open question. Each encoder tests
      its list against the bundle, and the registry test pins that both
      profiles ship the whole bundle.) Commit `Added: profile definitions
      list the XSDs their packages ship`. (Reversed 2026-09-28: the field
      is gone; the assembler ships `mets.Schemas` plus what the profile's
      descriptive encoder reports through `Schemas()`, and nothing else.
      `eark` therefore dropped the six meemoo XSDs and its reference copy
      was refreshed, the open question resolved.)
- **Withdrawn 2026-09-28: a supplied document travels the essence path.**
      Implemented in the working tree on 2026-09-24 (`DescriptiveDocument`
      on `Input` and `SourceRepresentation`, `dc.ValidateDocument`, an
      `encoders/xmldoc` package shared with the received-PREMIS check, a
      copy branch in the writer), reviewed, and discarded before commit:
      decision 6,
      [ADR-0017](../decisions/0017-supplied-descriptive-document-deferred.md).
      The Go files are back at the schemas commit; nothing to commit.
- [x] **Docs sweep.** Design doc: domain model, profile section, build
      lifecycle for the interface, the descriptive standard on the profile
      and the schema list. This plan's status line. (Drafted 2026-09-24
      together with the document route, never committed; redone without
      the route after the withdrawal of 2026-09-28. The representation's
      description fields, the eark layout, the validation section and the
      test inventory in `CLAUDE.md` stay in.) Commit `Changed: design doc
      follows the S2 refactor`.
- [x] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings; both comparisons clean. (Run 2026-09-24
      with `CSIP_CMD` pointing at the pinned commons-ip 2.11.2 jar on a
      host Java runtime, the Docker path being unsafe on the machine that
      day because of its open-file limit; same jar, same spec versions.
      After the withdrawal of 2026-09-28 the Go tree equals the schemas
      commit, where `go test ./...` is green.)

### S3: the mods world and the eark-mods profile

The library route is complete after this step.

- [ ] **`mods-3-7.xsd` in the bundle.** Fetch the schema (loc.gov refuses
      scripted downloads; the Internet Archive serves the raw file), check
      the header says MODS 3.7 and the size is about 53 KB, add it to
      `schemas/`. It imports `xml.xsd` and `xlink.xsd` by absolute loc.gov
      URLs; both are bundled already, and S6 resolves the URLs with an XML
      catalog. Commit `Added: MODS 3.7 XSD in the schema bundle`.
- [ ] **`profiles/earkmods` model.** (The package is the `eark-mods`
      profile's own, per ADR-0018; the boxes below said `encoders/mods`
      until 2026-09-28.) `description.go`: `Description` holding
      `Terms []sip.Term` (the shape the DC worlds share since 2026-09-28)
      and `Items []Item` (`CallNumber`, `Barcode`, `Enumeration`),
      implementing `sip.Description`. `vocabulary.go`: the key table with
      `identifier` and `title`, each row naming its template fragment and
      fixed attribute values; the MMS ID `type` value is one constant here
      (open question); the `required` list (identifier and title).
- [ ] **`profiles/earkmods` validation.** Known key, language tag shape,
      non-empty value, every finding joined; per item a non-empty call
      number and a barcode unique across items; `ValidateRequired()`
      over the package's own required list.
- [ ] **`profiles/earkmods` template.** Root `mods:mods` with the MODS
      namespace, `version="3.7"` and an `xsi:schemaLocation` onto
      `{{.Schemas}}/mods-3-7.xsd`; one fragment per key (`identifier` with
      its `type`, `titleInfo/title` with `xml:lang`); the items as one
      `location/holdingSimple` with one `copyInformation` per item
      (`shelfLocator`, `enumerationAndChronology` when set, `itemIdentifier
      type="barcode"` when set), omitted when there are no items; every
      value escaped. `Encode(w, d, schemas)` trusts validated terms and
      guards the interpolated element names in the template, as the two
      DC worlds do.
- [ ] **`profiles/earkmods` tests.** Table invariants (unique keys, fragment
      per row); encoder output (root, namespaces, version, escaping,
      `xml:lang`, one `copyInformation` per item with the right children,
      no `location` without items); refuses invalid terms or items without
      writing. Commit `Added: profiles/earkmods with the two-row key table
      and the items table`.
- [ ] **The `eark-mods` profile.** In `profiles/earkmods`, an unexported
      type implementing `build.DescriptionEncoder`: `Check` asserts the
      package's `Description`, `Encode` calls its `Encode`; no swap. An
      exported `Definition` copying `eark`'s values: `DescriptiveName
      "mods.xml"`, no cardinality or language rule, no PREMIS,
      `EmitRepresentationType` true; the encoder's `Schemas()` returns
      the package's own list (`mods-3-7.xsd` with `xlink.xsd` and `xml.xsd`,
      which it imports by loc.gov URL and S6 maps onto the package
      copies), six distinct files; declaration `DescriptiveMDType "MODS"`,
      `DescriptiveMDTypeVersion "3.7"`, the eark profile URL. One line in
      the registry in `profiles/`, so `profiles.Names()` and the CLI's
      unknown-profile message list it.
- [ ] **`profiles/` tests.** dc terms handed to `eark-mods`, and a mods
      description handed to `eark` or `basic`, fail before any write; a
      full `eark-mods` build has `metadata/descriptive/mods.xml`, exactly
      the six schema files, and a package METS `dmdSec` reading
      `MDTYPE="MODS" MDTYPEVERSION="3.7"`. Commit `Added: eark-mods
      profile`.
- [ ] **First `encoders/mets` test.** The `dmdSec` carries `MDTYPE` and
      `MDTYPEVERSION` from the declaration, and omits `MDTYPEVERSION` when
      the declaration leaves it empty. Commit `Added: mets encoder test for
      the dmdSec typing`.
- [ ] **Docs.** Design doc: encoders list, descriptive standards, profile table; the
      README's library example gets a MODS variant; `CLAUDE.md` system
      shape names `profiles/earkmods`. Commit `Changed: docs for the
      eark-mods profile`.
- [ ] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings; both comparisons clean. `eark-mods` has
      no fixture yet; the Go build test stands in until S6.

### S4: CLI rows

`description.csv` under `basic` and `eark` arrived in S2 (named by
standard until 2026-09-28, when the profile flag took over); this step
adds MODS rows and `items.csv`.

- [ ] **MODS rows.** `profiles/earkmods`'s encoder implements `NewDescription`
      like the DC worlds, so `check --profile eark-mods` and `create
      --profile eark-mods` read `description.csv` through the same
      decoder with no change to `cli/input`'s rows code. Items need a
      second input: extend `NewDescription`, or add an optional interface the
      CLI detects as the engine detects `IdentifierSwapper`; decide then.
      Decide the constructor's home with it. `NewDescription` is the one
      method on `build.DescriptionEncoder` the engine never calls; the
      CLI is its only caller, through `input.DescriptionBuilder`
      (2026-09-29). Move it off the encoder onto `Definition`: as a
      function field, the way `image.RegisterFormat` takes its decode
      function, or as a second small interface the profile's encoder type
      also implements. `DescriptionEncoder` then faces the engine only
      (`Check`, `Encode`, `Schemas`, the optional swap) and the CLI takes
      the constructor. The items shape (terms alone, or terms plus items)
      fixes the constructor's signature, which is why the move waits for
      this box instead of being done twice.
- [ ] **Reserved names.** `items.csv` joins the reserved names at the
      root.
- [ ] **`items.csv`.** Decoded like `representations.csv`: closed header
      in any order, case-insensitive, unknown or repeated column a
      violation; `callnumber` non-empty on every row; `barcode` unique;
      allowed only at the root and only under a profile that takes items.
- [ ] **Mapping onto `build.Material`.** `Package` carries the decoded
      `sip.Description` per level; `Package.Material` assigns it only when
      a file was read (the typed-nil pitfall from S2).
- [ ] **Tests.** mods rows, items rows and each items violation,
      `items.csv` under `basic` or `eark`.
- [ ] **Docs.** Input spec §1 (reserved names), §3 (the MODS key table
      under `eark-mods`, `items.csv`), §7 (mapping table); README Input
      section; design doc CLI paragraph. Commit
      `Added: MODS rows and items.csv in the input folder`.
- [ ] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings; both comparisons clean.

### S5: CLI supplied documents

Withdrawn 2026-09-28 with decision 6
([ADR-0017](../decisions/0017-supplied-descriptive-document-deferred.md)).
`dc.xml` and `mods.xml` in the input folder return with the route, if it
returns; the input specification keeps operator-supplied descriptive XML
as a deferred item (§8).

### S6: acceptance and closing docs

- [ ] **Fixture.** `tmp/eark-mods/`: a copy of `tmp/eark` whose
      `description.csv` holds MODS keys (`identifier`, `title[nl]`) and an `items.csv`
      of two copies, one carrying an enumeration.
- [ ] **XML catalog.** `scripts/schema-catalog.xml` rewriting the loc.gov
      URLs the MODS schema imports (`http://www.loc.gov/mods/xml.xsd`,
      `http://www.loc.gov/standards/xlink/xlink.xsd`) onto the package's
      `schemas/` copies, so xmllint runs with `--nonet`.
- [ ] **build.sh.** An `eark-mods` case (E-ARK 2.2.0, the default branch
      already does this; make it explicit); after `create`, `xmllint
      --noout --nonet --schema <pkg>/schemas/mods-3-7.xsd` over every
      `mods.xml` in the package, with `XML_CATALOG_FILES` pointing at the
      catalog; a failure exits non-zero like an INVALID package. xmllint
      joins the documented requirements in the script header, README and
      `CLAUDE.md`.
- [ ] **Run it.** `./build.sh eark-mods` VALID with 0 warnings and the
      xmllint pass clean; read the package METS `dmdSec` by hand:
      `MDTYPE="MODS" MDTYPEVERSION="3.7"`. Capture the package as
      `tmp/reference/eark-mods/pkg` and note it in `tmp/reference/README.md`.
      Commit `Added: eark-mods fixture, XML catalog and xmllint pass in
      build.sh`.
- [ ] **Closing docs.** README profile list (three profiles); design doc
      (profile table, package layout with `mods.xml`, validation section
      naming the xmllint pass); `CLAUDE.md` development commands ("all
      three profiles validate VALID"); `docs/TODO.md` drops the MODS
      question; ADR-0015 status to Accepted with the date (ADR-0016's
      status was settled on 2026-09-28); this plan's status line to
      shipped, then the plan moves to
      `docs/archive/` per `docs/README.md`. Commit `Changed: docs for the
      eark-mods profile; plan archived`.

## Open questions

- **The `type` attribute on `mods:identifier`** for the MMS ID: one constant,
  decided by the owner of the repository side before S3 ships or changed
  afterwards in one place.
- **Further MODS rows** and the relator code per role: supplied with the
  repository's index fields, each a table row. Settled 2026-09-22: names
  need personal and corporate rows (faculties, research departments and
  competence centres appear as agents), dates carry `encoding="edtf"`.
- **`eark` still ships `descriptive_basic.xsd`**, which it never references.
  Dropping it is a deliberate output change and stays out of this plan.
  (Resolved 2026-09-28 with ADR-0018's follow-up: a package ships only
  what its documents point at, so `eark` dropped the six meemoo XSDs and
  its reference copy was refreshed.)
- **Library callers** change their import from `encoders/metadata` to
  `encoders/dcschema` (meemoo) or `encoders/dc` (plain E-ARK) in S2, and
  on 2026-09-28 to `profiles/meemoo` and `profiles/eark`, with
  `profiles.New`, `profiles.Config` and `profiles.Input` becoming
  `build.New`, `build.Config` and `build.Input` (ADR-0018); the commit
  messages record each break.

## Progress so far (2026-09-24)

- **S1 shipped.** ADR-0015 and ADR-0016 drafted; in review the plan's
  rejection of a key grammar was sharpened (roles, title types and
  identifier types are table rows, personal and corporate names both, dates
  as single EDTF values), and the items table was added: a record's copies
  as `items.csv` (`callnumber` required, `barcode` and `enumeration`
  optional), package level only, because a representation is a CSIP
  rendition, never a copy or a volume.
- **Work runs on the branch `eark-mods-support`** with this plan as the
  checklist: one commit per box, proposed as a one-line message in chat
  and committed only after approval.
- **S2 so far:** the `eark` reference is captured under
  `tmp/reference/eark/pkg`; `encoders/metadata` became `encoders/dc`;
  `sip.Description` replaced the concrete terms type in the domain model;
  the family struct was built and then folded into the profile, retiring
  the `Family` constant (ADR-0007 superseded in that respect).
- **The DC world was split in two** on review: meemoo's dc+schema
  (`encoders/dcschema`, with meemoo's cardinality and Dutch-language rules
  in its own `Validate`) and Simple Dublin Core (`encoders/dc`, fifteen
  elements, no dumb-down). Input rows are named by standard
  (`dcschema.csv`, `dc.csv`; `metadata.csv` withdrawn), `Description`
  gained `ValidateRequired`, and `check` now reports a missing Dutch entry
  in a `dcschema.csv`. Both profiles still compare identical to their
  references.
- **Next:** S3 to S6, starting with the MODS 3.7 XSD in the bundle. S2
  closed on 2026-09-24 with both profiles VALID and their output unchanged.
  That day required elements became
  vocabulary keys, the identity rule (identifier and title) moved into
  `Input.Validate` so a profile lists only what its spec adds, each
  encoder names the XSDs its document points at and the profile
  concatenates them.
- **The supplied-document route was withdrawn on 2026-09-28**, before its
  commit. Review found the cost in the second input path at every level
  (the fields, the rules that a level has terms or a document, the
  assembler and writer branches, a shared XML reader), not in the check of
  the root element. The Go files went back to the schemas commit and the
  three new files were deleted;
  [ADR-0017](../decisions/0017-supplied-descriptive-document-deferred.md)
  records the deferral, and ADR-0015 and this plan no longer point richer
  records at a supplied document.
- **The descriptive pipeline was simplified** the same day, output
  unchanged (both structural comparisons clean): terms validation has one
  owner (`Input.Validate` is the contract, the CLI's calls report, the
  encoders trust, with a template guard on the element name); the
  `Description` interface is `Validate` and `ValidateRequired()`; the
  identifier swap and the local-identifier read sit behind the meemoo
  standard's value, and the two `Definition` flags for them are gone;
  what a package must state is each world's own list, so
  `Definition.RequiredKeys` is gone and `ResolveKey` is the CLI's alone;
  the per-rule validators nobody called are unexported.
  ADR-0012 and ADR-0015 carry dated notes.
- **The layout followed the same day**
  ([ADR-0018](../decisions/0018-engine-and-profile-packages.md)): the
  engine moved from `profiles/` to `build/`, each world moved into its
  profile's own package (`profiles/meemoo`, `profiles/eark`) next to an
  exported `Definition` and an unexported standard implementing
  `build.DescriptionEncoder`, `profiles/` itself became the registry
  only, and `encoders/` kept the two renderers of the graph. Imports now
  run one way, from the CLI through the registry and the profile
  packages to the engine; the engine imports no profile. Output
  unchanged, both structural comparisons clean. In review the same day:
  the engine's interface became `build.DescriptionEncoder`, each profile
  package became one encoder struct (`dcschema`, `simpledc`) plus its
  `Definition`, and `Definition.Schemas` went: a package ships what its
  documents point at, so `eark` dropped the six unreferenced meemoo XSDs
  (its reference copy refreshed; `./build.sh eark` re-run the same day:
  VALID, 0 warnings, the same counts as before).
- **The statement shape became shared** later the same day, output
  unchanged (both structural comparisons clean): a term is a `sip.Term`
  keyed by the plain vocabulary key, and each world's `Terms` is a list
  of them with its own table, rules and template; the element a key
  emits is the template's business alone. `ResolveKey` and the CLI's
  per-world row builders are gone: the CLI decodes rows once and wraps
  them in the world's `Terms`, and a per-term finding (`*sip.TermError`)
  names the term's position, mapped back to the row's line. Findings
  name keys in both worlds (`description is required but missing`).
  ADR-0015 and ADR-0018 carry dated notes; S3's boxes follow.
- **The rows file became `description.csv`** later still, and `check`
  takes `--profile` as `create` does: the profile says which vocabulary
  the rows are in, the CLI hands decoded rows to the encoder's `NewDescription`,
  and `cli/input` imports no profile package. The names by standard, the
  one-standard-per-folder rule and `mods.csv` are withdrawn (ADR-0016's
  note; ADR-0010 notes that a per-run flag is not configuration).
  Fixtures renamed; output unchanged, both structural comparisons clean.
  S4's boxes follow.
- **The reader takes only a description builder** (2026-09-29):
  `input.ReadDirectory` became `input.New(builder).Read(root)`, the
  constructor-plus-method shape of `build` and `archive`. The reader
  holds an `input.DescriptionBuilder`, a one-method interface declared in
  `cli/input` that the profile's descriptive encoder satisfies, instead
  of the whole `build.Definition`; `check` and `create` construct it with
  `def.Encoder`. No profile or output change. Moving
  `NewDescription` off `build.DescriptionEncoder` is added to S4's MODS
  rows box, where the items shape decides the constructor's signature.
- **`build.Input` became `build.Material`** the same day, with
  `Package.BuilderInput()` becoming `Package.Material()` and the file
  `build/material.go`: the type is one package's source material as
  data, the word the docs already used for it, and "input" was spent
  twice, on the CLI's folder and on the library's data. Library callers
  change the type name; nothing else changes.
