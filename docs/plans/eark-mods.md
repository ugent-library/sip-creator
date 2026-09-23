# Plan: MODS 3.7 descriptive metadata for the plain E-ARK output

*Status: **S1 done, S2 next** (2026-09-23): ADR-0015 and ADR-0016
drafted, including the items table (decision 12). Design decisions settled
in review on 2026-09-15; the MODS element list beyond identifier and title
is still open and does not block the first steps. Update this line as steps
land.*

## Context

The tool emits one descriptive document per profile family: meemoo's
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
   becomes Dublin Core. It must also take rows that become MODS, and a
   finished `dc.xml` or `mods.xml` prepared elsewhere (a catalogue export,
   for instance) that the tool copies into the package after checking its
   shape.

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

1. **Two separate descriptive worlds, no shared term type.** `encoders/dc`
   and `encoders/mods` each own their terms type, closed vocabulary table,
   validation and templates. `encoders/metadata` is renamed to `encoders/dc`
   and keeps both existing templates (meemoo `dc+schema`, simple DC).
   Rejected: one neutral term type keyed by plain vocabulary key, shared by
   both worlds. It would have unified the CSV and the library on one key
   language, but a MODS term produces a subtree while a DC term produces an
   element, and forcing both through one type hides that difference.
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
   sub-terms. That is acceptable because the source data is flat columns;
   richer records arrive as supplied documents.
3. **The profile fixes the descriptive standard.** The input supplies either
   terms to encode in that standard or a ready document of that standard.
   MODS terms or a `mods.xml` given to a DC profile is a build error before
   any disk write, and the reverse likewise. Meemoo profiles accept terms
   only: their document needs the swapped identifier and the meemoo
   namespace. Rejected: letting the input decide the standard and the
   profile follow. It would make one profile emit two different documents
   depending on what it was given, and `MDTYPE` would stop being profile
   data.
4. **Families and profiles.** Families become `meemoo` (dc world, dc+schema),
   `eark-dc` (dc world, simple DC) and `eark-mods` (mods world). Profiles
   stay what operators type: `basic` (meemoo), `eark` (eark-dc, name kept)
   and the new `eark-mods`. The `eark-mods` entry copies `eark` with
   `MDTYPE="MODS"`, `MDTYPEVERSION="3.7"`, `mods.xml` as the document name,
   identifier and title required, no cardinality or language rule, no
   PREMIS. ADR-0007's promotion trigger fires: the `Family` constant resolves
   to an internal struct of choices (which world, which encoder, whether
   supplied documents are accepted).
5. **A small interface in the domain model.** `sip/` declares
   `Description` with `LocalIdentifier() string` and `Validate() error`;
   both terms types implement it, and `sip/` stops importing an encoder
   package. The family asserts the concrete world at build time (decision
   3). The dc-only identifier swap stays a method on the dc terms type.
   Required elements on a `Definition` become plain vocabulary keys
   (`identifier`, `title`), valid in both worlds.
6. **Supplied documents reuse the essence path.** A descriptive `*sip.File`
   with `Source` set is copied by the writer with fixity computed by the
   store, one without is generated. `profiles.Input` and
   `SourceRepresentation` gain a `DescriptiveDocument` source next to
   `Descriptive`; exactly one of the two per level (the package needs one,
   a representation may have none).
7. **Structural checks only on supplied documents.** In process, with the
   standard library: well-formed XML, the expected root element and
   namespace (`simpledc` without namespace for DC, matching what the eark
   template emits; `mods` in `http://www.loc.gov/mods/v3` with
   `version="3.7"` for MODS). No XSD validation in the tool
   ([ADR-0003](../decisions/0003-validation-stays-external.md)); build.sh
   runs xmllint over the emitted `mods.xml` as acceptance. Rejected: XSD
   validation in process, which needs a cgo binding or executing an
   external tool, both against the project's rules.
8. **No identity enforcement on supplied documents.** Rows must carry
   identifier and title at package level; a supplied document is trusted
   for its content. Enforcing identity would mean parsing two XML shapes
   for one check the repository performs anyway.
9. **CLI file names by standard.** `dc.csv` and `mods.csv` hold rows,
   `dc.xml` and `mods.xml` hold supplied documents, at the package root and
   inside each representation directory. `items.csv` (MODS only, package
   root only, next to `mods.csv`) holds one row per physical copy, see
   decision 12. One source per level, one standard per folder. The folder stays self-describing, so `check` keeps taking no
   configuration ([ADR-0010](../decisions/0010-config-over-self-describing-input.md));
   a mismatch with the chosen profile surfaces at `create`. `metadata.csv`
   is withdrawn: its presence is a violation telling the operator to rename
   it to `dc.csv`. Rejected: keeping `metadata.csv` and letting the profile
   decide its meaning, which forces a profile flag onto `check`. Rejected:
   one merged key space serving both standards, which reintroduces the
   silent lossy mapping ADR-0011 removed.
10. **Schema set as profile data.** `mods-3-7.xsd` joins the bundle (it
    imports `xlink.xsd` and `xml.xsd`, already bundled). A `Definition`
    lists the XSD files its packages ship. `basic` and `eark` list exactly
    the eleven files they ship today, so their output is unchanged;
    `eark-mods` lists the METS core set plus the MODS XSD.
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
    accompanies `mods.csv` only: a supplied `mods.xml` carries its own
    holdings, and the DC world has no place to pair them. Descriptive
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
| `dc.csv` | rows, `key[lang],value`, keys from the Dublin Core table | DC |
| `mods.csv` | rows, `key[lang],value`, keys from the MODS table | MODS |
| `dc.xml` | a finished simple Dublin Core document (`simpledc` root) | DC |
| `mods.xml` | a finished MODS 3.7 document (`mods` root, `version="3.7"`) | MODS |

`items.csv` MAY accompany `mods.csv` at the input root, one row per physical
copy, with the columns `callnumber`, `barcode` and `enumeration`.

Rules, all MUST violations collected by `check`:

- More than one of the four files at one level is a violation.
- All descriptive files in one input folder speak the same standard; a DC
  file next to a MODS file anywhere in the folder is a violation.
- `metadata.csv` is a violation; rename it to `dc.csv`.
- Row files follow today's `metadata.csv` rules: `key,value` header, UTF-8,
  unknown keys are violations, repeat a key for multiple values, `[lang]`
  suffix for the language. At package level `identifier` and `title` are
  required.
- Supplied documents must be well-formed XML with the expected root element
  and namespace; a MODS document must declare version 3.7. Nothing else is
  checked; the repository validates content.
- `items.csv` follows the `representations.csv` rules: UTF-8, a header
  naming `callnumber` and optionally `barcode` and `enumeration` in any
  order (case-insensitive; an unknown or repeated column is a violation),
  `callnumber` non-empty on every row, a `barcode` value unique across
  rows. Next to `dc.csv`, `dc.xml` or `mods.xml`, or inside a
  representation directory, it is a violation.
- The profile chosen at `create` must match the folder's standard: `basic`
  and `eark` take DC, `eark-mods` takes MODS. `basic` takes rows only.

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
assembler that step 3 turns into a guaranteed one. Step 6 lands the
assembler and writer halves in one commit, because a declared file that is
never written breaks the package METS.

- [ ] **Capture the `eark` reference.** `./build.sh eark` VALID, then copy
      the package directory to `tmp/reference/eark/pkg` next to the `basic`
      copy and note it in `tmp/reference/README.md`. Run `./build.sh` for
      `basic` and check both comparisons are clean before any code moves.
      Nothing to commit; `tmp/` is untracked.
- [ ] **Rename `encoders/metadata` to `encoders/dc`.** `git mv` the
      directory, change the package clause in its files, fix the importers
      (`cli/input/input.go`, `cli/input/metadata.go` and their tests,
      `sip/entity.go`, `sip/representation.go`, the `profiles/` files and
      tests). Update the package name in `docs/sip-creator-design.md` and
      `CLAUDE.md`. Commit `Changed: encoders/metadata renamed to
      encoders/dc`; the message records the import break for library
      callers.
- [ ] **The `Description` interface in `sip/`.** New `sip/description.go`
      with `LocalIdentifier() string` and `Validate() error`.
      `Entity.Description`, `Representation.Description`,
      `Input.Descriptive` and `SourceRepresentation.Descriptive` take the
      interface; `sip/` drops its encoder import. `Input.Validate` checks
      nil instead of length. The swap branch in `assembleDescriptive`
      asserts `dc.Terms`, with a comment naming the family check of the
      next step as the guarantee. Pitfall: a nil `dc.Terms` stored in the
      interface is a non-nil interface, so the mapping onto
      `profiles.Input` in `cli/input` assigns a representation's terms only
      when a file was read. Commit `Changed: the domain model describes
      entities through a Description interface`.
- [ ] **`Family` resolves to a struct.** Lift the family code out of
      `profiles/definition.go` into `profiles/family.go`: a `family` struct
      holding the descriptive encoder (taking a `sip.Description`), a world
      check asserting every description is `dc.Terms`, and an
      `acceptsDocument` flag (false for meemoo). Rename `FamilyEARK` to
      `FamilyEARKDC` (`"eark-dc"`) per decision 4. `Builder.Build` resolves
      the family first, runs the world check, then validates and
      assembles. Test: a fake `Description` type fails the build before any
      write, next to `TestBuildInvalidConfigWritesNothing`. Commit
      `Changed: Family resolves to an internal struct of choices`.
- [ ] **Required elements become keys.** `Definition.RequiredElements`
      becomes `RequiredKeys`: `identifier` and `title` for `eark`, the dc
      table's required keys for `basic`. `dc.Terms.ValidateRequired`
      resolves keys through the table. Adjust the messages and the three
      tests that name elements. Commit `Changed: required descriptive
      elements are named by vocabulary key`.
- [ ] **Schema list on `Definition`.** A `Schemas` field listing file
      names; both profiles list the eleven files in `schemas/` today. The
      assembler builds the schema nodes from the list, sorted as now; the
      writer looks each up in the bundle. A test asserts every listed name
      exists in the bundle. Commit `Added: profile definitions list the
      XSDs their packages ship`.
- [ ] **A supplied document travels the essence path.**
      `DescriptiveDocument` on `Input` and `SourceRepresentation`; the
      one-of check in `Validate` (exactly one at package level, at most one
      per representation; the identifier check is skipped for a document).
      `dc.ValidateDocument`, modelled on `premis.ValidateReceived`:
      well-formed XML, `simpledc` root without namespace. The assembler
      validates the document as it does received PREMIS, declares the file
      node with `Source` set and no description, and skips the identifier
      lift and swap; the meemoo family refuses a document before
      validation. In `profiles/write.go` a description file with `Source`
      set is copied with `CopyFile`, one without is generated; the
      representation branch guards on the file node, not the description.
      Tests: both-or-neither in `TestInputValidate`; a supplied document
      copied byte for byte with its fixity in the METS; meemoo refusing
      with nothing written; `ValidateDocument` rejecting an `oai_dc:dc`
      root. Commit `Added: a supplied descriptive document travels the
      essence path`.
- [ ] **Docs sweep.** Design doc: domain model, profile section, build
      lifecycle for the interface, the family struct, the schema list and
      the document route. This plan's status line. Commit `Changed: design
      doc follows the S2 refactor`.
- [ ] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings; both comparisons clean.

### S3: the mods world and the eark-mods profile

The library route is complete after this step.

- [ ] **`mods-3-7.xsd` in the bundle.** Fetch the schema (loc.gov refuses
      scripted downloads; the Internet Archive serves the raw file), check
      the header says MODS 3.7 and the size is about 53 KB, add it to
      `schemas/`. It imports `xml.xsd` and `xlink.xsd` by absolute loc.gov
      URLs; both are bundled already, and S6 resolves the URLs with an XML
      catalog. Commit `Added: MODS 3.7 XSD in the schema bundle`.
- [ ] **`encoders/mods` model.** `description.go`: `Description` holding
      `Terms []Term` (`Key`, `Lang`, `Value`) and `Items []Item`
      (`CallNumber`, `Barcode`, `Enumeration`), implementing
      `sip.Description`; `LocalIdentifier` returns the `identifier` term.
      `vocabulary.go`: the key table with `identifier` and `title`, each
      row naming its template fragment and fixed attribute values; the MMS
      ID `type` value is one constant here (open question). `ResolveKey`.
- [ ] **`encoders/mods` validation.** Known key, language tag shape,
      non-empty value; per item a non-empty call number and a barcode
      unique across items; `ValidateRequired(keys...)`.
- [ ] **`encoders/mods` template.** Root `mods:mods` with the MODS
      namespace, `version="3.7"` and an `xsi:schemaLocation` onto
      `{{.Schemas}}/mods-3-7.xsd`; one fragment per key (`identifier` with
      its `type`, `titleInfo/title` with `xml:lang`); the items as one
      `location/holdingSimple` with one `copyInformation` per item
      (`shelfLocator`, `enumerationAndChronology` when set, `itemIdentifier
      type="barcode"` when set), omitted when there are no items; every
      value escaped. `Encode(w, d, schemas)` validates first.
- [ ] **`mods.ValidateDocument`.** Well-formed XML, root `mods` in the MODS
      namespace, `version="3.7"`; modelled on `premis.ValidateReceived`.
- [ ] **`encoders/mods` tests.** Table invariants (unique keys, fragment
      per row); encoder output (root, namespaces, version, escaping,
      `xml:lang`, one `copyInformation` per item with the right children,
      no `location` without items); refuses invalid terms or items without
      writing; `ValidateDocument` accepts a 3.7 document and rejects a
      `modsCollection` root, another version, and non-XML. Commit `Added:
      encoders/mods with the two-row key table and the items table`.
- [ ] **The `eark-mods` family and profile.** `FamilyEARKMODS`
      (`"eark-mods"`) in `profiles/family.go`: world check asserts the mods
      `Description`, encoder `mods.Encode`, `mods.ValidateDocument`,
      accepts documents. Registry entry copying `eark`: `DescriptiveName
      "mods.xml"`, `RequiredKeys` identifier and title, no cardinality or
      language rule, no PREMIS, `EmitRepresentationType` true, `Schemas` =
      `mets1_12.xsd`, `DILCISExtensionMETS.xsd`,
      `DILCISExtensionSIPMETS.xsd`, `xlink.xsd`, `xml.xsd`, `mods-3-7.xsd`;
      declaration `DescriptiveMDType "MODS"`, `DescriptiveMDTypeVersion
      "3.7"`, the eark profile URL. `profiles.Names()` lists it, so the
      CLI's unknown-profile message does too.
- [ ] **`profiles/` tests.** dc terms handed to `eark-mods`, and a mods
      description handed to `eark` or `basic`, fail before any write; a
      full `eark-mods` build has `metadata/descriptive/mods.xml`, exactly
      the six schema files, and a package METS `dmdSec` reading
      `MDTYPE="MODS" MDTYPEVERSION="3.7"`; a supplied `mods.xml` builds
      under `eark-mods`. Commit `Added: eark-mods family and profile`.
- [ ] **First `encoders/mets` test.** The `dmdSec` carries `MDTYPE` and
      `MDTYPEVERSION` from the declaration, and omits `MDTYPEVERSION` when
      the declaration leaves it empty. Commit `Added: mets encoder test for
      the dmdSec typing`.
- [ ] **Docs.** Design doc: encoders list, families, profile table; the
      README's library example gets a MODS variant; `CLAUDE.md` system
      shape names `encoders/mods`. Commit `Changed: docs for the mods
      encoder and the eark-mods profile`.
- [ ] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings; both comparisons clean. `eark-mods` has
      no fixture yet; the Go build test stands in until S6.

### S4: CLI rows

- [ ] **Split row reading from world building.** In `cli/input`, one
      generic reader for `key[lang],value` rows (header check, key
      parsing, line numbers for violations) and one small builder per
      world: dc rows through `dc.ResolveKey` into `dc.Terms`, mods rows
      through `mods.ResolveKey` into a mods `Description`. Rename
      `metadata.go` and its test after the file it now reads.
- [ ] **Reserved names.** `dc.csv`, `mods.csv`, `dc.xml`, `mods.xml`,
      `items.csv` join the reserved names at the root and inside
      representation directories (`items.csv` root only). `metadata.csv`
      becomes a violation telling the operator to rename it to `dc.csv`.
- [ ] **One source per level, one standard per folder.** More than one
      descriptive file at one level is a violation; none at the root is a
      violation; the folder's standard is the standard of its first
      descriptive file, and any file of the other standard anywhere is a
      violation naming both files.
- [ ] **`items.csv`.** Decoded like `representations.csv`: closed header
      in any order, case-insensitive, unknown or repeated column a
      violation; `callnumber` non-empty on every row; `barcode` unique;
      allowed only at the root and only next to `mods.csv`.
- [ ] **Mapping onto `profiles.Input`.** `Package` carries the decoded
      `sip.Description` per level; `BuilderInput` assigns it only when a
      file was read (the typed-nil pitfall from S2).
- [ ] **Tests.** dc rows, mods rows, items rows and each items violation,
      the `metadata.csv` violation, two files at one level, mixed
      standards, `items.csv` next to `dc.csv`; `read_test.go`,
      `builder_test.go` and `representations_test.go` fixtures renamed.
- [ ] **Fixtures.** `tmp/basic/metadata.csv`, `tmp/eark/metadata.csv` and
      `tmp/eark/representations/master/metadata.csv` renamed to `dc.csv`.
- [ ] **Docs.** Input spec §1 (reserved names), §3 (the four row and
      document files, the MODS key table, `items.csv`), §7 (mapping
      table); README Input section; design doc CLI paragraph. Commits
      `Changed: metadata.csv becomes dc.csv` and `Added: mods.csv and
      items.csv rows in the input folder`.
- [ ] **Acceptance.** `go test ./...`; `./build.sh basic` and `./build.sh
      eark` VALID with 0 warnings from the renamed fixtures; both
      comparisons clean.

### S5: CLI supplied documents

- [ ] **Detection and validation.** `dc.xml` and `mods.xml` take part in
      the one-source rule; `cli/input` opens each and runs the world's
      `ValidateDocument`, turning an error into a violation naming the file
      and the reason. `Package` carries the document path per level;
      `BuilderInput` sets `DescriptiveDocument`.
- [ ] **Items rule.** `items.csv` next to `mods.xml` is a violation: the
      document carries its own holdings.
- [ ] **Tests.** A valid `dc.xml` accepted; `mods.xml` with another
      version, and a non-XML file, each a violation; `dc.xml` next to
      `dc.csv` a violation; a representation-level `mods.xml` accepted;
      `items.csv` next to `mods.xml` a violation.
- [ ] **Docs.** Input spec §3 (supplied documents and their checks) and §8
      (the deferred operator-supplied XML item is enacted for DC and MODS;
      other standards stay deferred); README Input section. Commit `Added:
      dc.xml and mods.xml as supplied descriptive documents`.
- [ ] **Acceptance.** `go test ./...`; both profiles VALID; both
      comparisons clean; `check` run by hand on a folder holding a
      `dc.xml`.

### S6: acceptance and closing docs

- [ ] **Fixture.** `tmp/eark-mods/`: a copy of `tmp/eark` with `dc.csv`
      replaced by `mods.csv` (`identifier`, `title[nl]`) and an `items.csv`
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
      question; ADR-0015 and ADR-0016 status to Accepted with the date;
      this plan's status line to shipped, then the plan moves to
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
- **Library callers** change their import from `encoders/metadata` to
  `encoders/dc` in S2; the commit message records the rename.
