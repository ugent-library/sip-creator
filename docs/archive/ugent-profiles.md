# Plan: a profile is a content type; the ugent family replaces eark

*Note, 2026-10-07, after the plan shipped: the separate model packages of step 3
(`profiles/simpledc`, `profiles/mods`) were folded into `profiles/ugent` the same day,
before the branch was merged: `ugent.Terms` and `ugent.Record`. Decision 3 of
[ADR-0034](../decisions/0034-a-profile-is-a-content-type.md) records why. The text
below keeps the plan as it ran.*

*Status: **shipped** 2026-10-07 and archived (drafted and revised the same day). The design is in [sip-creator-design.md](../sip-creator-design.md), the decisions in [ADR-0034](../decisions/0034-a-profile-is-a-content-type.md), the rules in the [profile pages](../profiles/). Step 1 landed 2026-10-07:
[ADR-0034](../decisions/0034-a-profile-is-a-content-type.md) records decisions 1, 2, 3, 8
and 9. Step 2 landed 2026-10-07: [ugent-basic](../profiles/ugent-basic.md) and
[ugent-bibliographic](../profiles/ugent-bibliographic.md), written ahead of the code,
mark each rule with the step that checks it. Step 3 landed 2026-10-07: the models live in
`profiles/simpledc` and `profiles/mods`, each exporting `Model`. Step 4 landed
2026-10-07: `ugent/basic` and `ugent/bibliographic` replace `eark/dc` and `eark/mods`.
Step 5 landed 2026-10-07: the package METS declares `OTHER` with the profile's name; the
profile pages gained a fourth marker, "written", for a rule the tool holds by writing the
value itself. Step 6 landed 2026-10-07: no flat input folder; until step 8 the reader
also refuses a folder without `representations/`, because `check` does not run the
library's `SourcePackage.Validate`. Step 7 landed 2026-10-07: `RepresentationTypes`; under
a vocabulary an empty type resolves to the name, not the label; the UGent examples hold
`archival/` (a TIFF) and `access/` (a JPEG). A `representations.csv` type other than the
folder is refused by `ValidateSource`, naming the representation, not the row's line.
Step 8 landed 2026-10-07: `MinRepresentations`; the UGent profiles take a package
without representations, which commons-ip reports VALID with two SHOULD-level warnings
(CSIPSTR11, CSIPSTR13; see docs/TODO.md). Step 9 dropped 2026-10-07: the tool does not
check an identifier's syntax. Step 10 landed 2026-10-07: TODO, the runbook and the
input specification's §4 follow; ADR-0029 has a note on CSIPSTR9. The plan takes over from the retired
[profile-rules-and-names plan](../archive/profile-rules-and-names.md): its parked step 5
(the representation minimum) returns here as step 8; its step 8 (`eark/none`) is dropped;
its steps 2 to 4 (a profile without descriptive metadata) are absorbed by the parked
[entities-and-descriptions plan](../plans/entities-and-descriptions.md), where an empty list of
models means no descriptive metadata. Update this line as steps land.*

## Context

### What a profile is

A profile is a content type: an information type with its own semantics, and the rules
and constraints on how a SIP for it is built, serving the ingest workflows and archival
management of the institution that defines it. The descriptive standard is one
consequence of the content type, not what the profile is named after.

That is how the field works. CSIP calls it the content information type: a package
declares one in `csip:CONTENTINFORMATIONTYPE`, and a specification per type (ERMS, SIARD,
geospatial data, eHealth, archival information, 3D product models) fixes its structure and
metadata. None of the published ones covers digitised library material. Meemoo SIP 1.2
defines three content profiles, basic, bibliographic and material artwork, each declared
by a versioned URI in `csip:OTHERCONTENTINFORMATIONTYPE`, and each fixing the descriptive
standard, the representation rules and the PREMIS rules from the content: basic requires
`dc+schema.xml` and exactly one representation; bibliographic requires `mods.xml` (MODS
3.7), one page per TIFF or ALTO file and a structMap with ordered page divisions;
material artwork requires `dc+schema.xml` and PREMIS at both levels. Rosetta has material
flows with an entity type per flow. Riksarkivet adopted CSIP plus E-ARK SIP as its
packaging specification and names content-type specifications next to it.

### The tension in today's names

The registry mixes two readings. `meemoo/basic` is a content profile, Meemoo's. `eark/dc`
and `eark/mods` are named by their descriptive standard, because the eark family has no
content profile to name: they declare `CONTENTINFORMATIONTYPE="MIXED"`, CSIP's value for
"no content type specification applies", and the choice of model went on the name
instead ([ADR-0030](../decisions/0030-profile-names-by-family.md)). The second part of a
name answers a different question per family, and every new profile would have to be
forced onto one reading or the other.

### UGent's two content types

UGent Library delivers to its own RODA instance, and to Meemoo under Meemoo's profiles,
which this plan does not touch. Two content types cover what it packages for RODA:

- **basic**: digital-born or digitised resources the library has not necessarily
  accessioned or catalogued, but that still need packaging: an internal database dump, a
  deposited set of files. The description is a short Simple Dublin Core record.
- **bibliographic**: digital-born or digitised resources the library has formally
  accessioned, catalogued and recognised as its holdings: books, manuscripts, plans, maps,
  pictures, notated music, artworks. The catalogue record is the description's source,
  written as MODS 3.7, and its record identifier (the Alma MMS ID) identifies the
  intellectual entity.

The two share a name with Meemoo's profiles and mean something else. Meemoo's basic is
one media file in one representation; UGent's basic is anything uncatalogued. Meemoo's
bibliographic is paged text with TIFF, ALTO, page order and transcription events; UGent's
criterion is a management one, whether the library catalogued the item, so a map or a
photograph of a painting is bibliographic too, and the profile fixes no page structure.
The owner in the name keeps them apart; the profile pages say so.

### Facts that shape the plan

- RODA sets the AIP type from the package METS's content information type. When the type
  is `OTHER` with an other value, that value becomes the AIP type (commons-ip
  `IPContentType.asString`, RODA `EARKSIP2ToAIPPluginUtils.getType`). RODA's default
  configuration accepts any string there (`core.aip_type.controlled_vocabulary: false`).
- A representation's type reaches RODA through the representation METS
  ([ADR-0013](../decisions/0013-representation-type-from-label.md)). Unchanged here.
- One SIP becomes one AIP in RODA. Hierarchy is parent AIPs, assigned by the ingest job's
  dropfolder or named by a RODA-specific ancestors structMap. Sub-entities inside a
  package are therefore not this plan's concern; the parked
  [entities-and-descriptions plan](../plans/entities-and-descriptions.md) has them.
- Systems that automate UGent's RODA ingest already enforce, on their own input: a
  representation directory is named from a closed vocabulary (`preservation`, `access`,
  `archival`), one directory per type, at least one, each holding files; the content
  inside a representation is free (digitised physical items nest one directory per copy,
  named by barcode); one intellectual entity, identified by the catalogue record and
  described in MODS by title, author and typed identifiers, with one barcode identifier
  per copy; documentation optional. Rules that describe the SIP move to the profile, so
  the CLI and a program that embeds the library produce the same package and the check
  exists once. Rules about such a system's own drop folder stay with it.
- The MODS record the tool writes is already UGent's application profile of MODS: a
  catalogue identifier, titles, and copies with call number, barcode and enumeration.
  Naming it `ugent/bibliographic` makes that honest and settles the vocabulary question
  [ADR-0033](../decisions/0033-ugent-first-profiles-of-your-own.md) reopened: the model
  keeps the library's catalogue words. The parked [mods-coverage plan](../archive/mods-coverage.md)
  gets a note.
- commons-ip checks `OTHER` plus a value under 2.2.0 as it does for `meemoo/basic` under
  2.0.4 (CSIP5 and CSIP6), so the declaration change should validate; step 5 confirms
  it with build.sh. CSIP6 asks only that the other value state the content information
  type; it does not ask for a URI.

## Decisions (proposed; each confirmed when its step starts)

1. **A profile is a content type, named `<owner>/<content type>`.** The owner is who
   defines the rules: `meemoo` or `ugent`. The names are `meemoo/basic`, `ugent/basic`
   and `ugent/bibliographic`. The specification a package conforms to beyond CSIP (Meemoo
   SIP 1.2, or the plain E-ARK SIP 2.2.0) is a property of the profile, declared in
   `mets/@PROFILE`, no longer a family. `eark/dc` and `eark/mods` retire: their
   definitions become the UGent profiles. Supersedes ADR-0030.
2. **Each UGent profile declares its content type.** The package METS carries
   `CONTENTINFORMATIONTYPE="OTHER"` and, in `OTHERCONTENTINFORMATIONTYPE`, the profile's
   name as a plain value: `ugent/basic` or `ugent/bibliographic` (settled 2026-10-07).
   RODA makes that value the AIP type, so the two content types can be told apart there.
   Not a URI, as Meemoo uses: a URI needs a namespace someone at the library owns and
   keeps, and CSIP6 does not ask for one. No version in the value for now; when a rule
   change would refuse a package ingested earlier, the value gains one. The
   representation METS keeps ADR-0013's type declaration.
3. **Models are standards; profiles pick them.** The Simple Dublin Core model and `Terms`
   move to `profiles/simpledc`; the MODS model and `Record` to `profiles/mods`; the two
   UGent definitions live in `profiles/ugent` as `ugent.Basic` and `ugent.Bibliographic`.
   A later `meemoo/bibliographic`, which also requires MODS 3.7 at package level, imports
   the same model. Meemoo's `dc+schema` model stays in `profiles/meemoo` until a second
   profile needs it. The extension point for a profile outside the module (ADR-0033) is
   unchanged.
4. **A written specification per profile**: `docs/profiles/ugent-basic.md` and
   `docs/profiles/ugent-bibliographic.md`, written the way Meemoo's profile pages are:
   scope, declaration, descriptive metadata, representations, files, preservation
   metadata, documentation, updates, and what RODA makes of each part. Design genre (what
   is), listed in `docs/README.md` as a tier. The input specification keeps the folder
   rules and points at the profile pages for the rules that differ per profile. Every rule
   field on the definition quotes the sentence it enforces.
5. **Rules are typed fields on `build.Definition`, checked in `ValidateSource`**
   ([ADR-0026](../decisions/0026-profile-rules-on-the-definition.md)), and a field is
   added only when a profile page states the rule. This plan adds two: the
   representation vocabulary and the representation minimum (ADR-0029's). The
   identifier's syntax was proposed and dropped (step 9). Not added, because no page states them yet: nesting inside a
   representation, allowed formats per representation, required documentation, the
   fixity algorithm, required PREMIS events, file naming, page order. Each is data when
   it comes; page order also needs the graph and the METS encoder to express an ordered
   structMap, and stays out until a page says MUST.
6. **The representation vocabulary** is `preservation`, `access` and `archival`: exact
   lowercase names, at most one representation per type. The name is the directory under
   `representations/`, the representation METS `OBJID`, and the type ADR-0013 emits; the
   label stays free. The vocabulary says nothing about how many representations a package
   needs: that is `MinRepresentations` (decision 7, step 8). Every representation rule is
   checked per representation, so a metadata-only package, which has none, passes them
   with nothing to check; only the minimum can refuse it. Both UGent profiles allow all
   three types, in any combination: none, one, two or all three. Essence always sits in
   a named representation folder; the input folder no longer stands for one
   representation itself (decision 9). Their meaning (settled 2026-10-07, for the profile pages):
   - `preservation` is the copy as delivered: the raw data, the born-digital file as
     deposited, the files of a digitisation as the scanning produced them. Anything that
     comes from outside, such as born-digital material from an estate, lands here.
   - `archival` is a copy in a format suitable for long-term archiving, derived from the
     delivered copy where that is not in one: a conversion to PDF/A, for instance.
   - `access` is the copy that will be disseminated.
7. **Per profile:**

   | | `ugent/basic` | `ugent/bibliographic` |
   |---|---|---|
   | content information type | `OTHER`, `ugent/basic` | `OTHER`, `ugent/bibliographic` |
   | content category (`mets/@TYPE`) | `Mixed`, overridable as today | `Mixed`, overridable as today |
   | descriptive standard | Simple Dublin Core, `dc.xml` | MODS 3.7, `mods.xml` |
   | package description | required: identifier and title | required: the catalogue identifier and a title; copies optional |
   | representation descriptions | allowed | allowed |
   | supplied document | `dc.xml` | `mods.xml` |
   | representations | the vocabulary; zero or more | the vocabulary; zero or more |
   | PREMIS | none generated; supplied documents accepted | none generated; supplied documents accepted |
   | documentation | optional | optional |

8. **No profile without descriptive metadata.** `eark/none` is dropped: basic needs a
   description, because a package nothing describes is what basic exists to avoid.
   ADR-0029's decision that a nil model means no descriptive metadata stands as
   architecture; its implementation steps stay parked. The reverse is allowed: a package
   with a description and no representations, under either UGent profile, whether it
   is new or an update (settled 2026-10-07). It describes an intellectual entity whose
   content is not, or not yet, in the archive.
9. **No flat input folder.** Today an input folder without `representations/` is one
   representation named after the folder. That goes, for every profile: the reader cannot
   know which kind of representation loose files are, and under a vocabulary the folder's
   own name would have to be one of its words. Essence lives in
   `representations/<name>/`, with an optional `representations.csv`; a folder without
   `representations/` has no representations. `cli/input` stays unaware of profiles:
   the rule is the input specification's, the same for all. `meemoo/basic` keeps its
   one representation through `MinRepresentations` and `MaxRepresentations`; all three
   examples already use `representations/`.

## Steps

### Step 1. The ADR and the status notes

`Added: ADR-0034: a profile is a content type defined by its owner; ugent/basic and ugent/bibliographic replace eark/dc and eark/mods`

- ADR-0034: the context above, decisions 1, 2, 3, 8 and 9, the alternatives (keep the
  eark names and add rules to them; one `eark` profile with a flag for the standard; name
  the UGent profiles after their standard; a versioned URI as the content type, as
  Meemoo declares; keep the flat input folder and let the vocabulary refuse its name),
  the consequences (the name changes, the import path changes, the output change of step
  5, the retired `eark/none`, the flat input folder gone for every profile).
- Dated notes under the status lines of ADR-0030 (superseded), ADR-0018 (superseded in
  part: a model's package is named after its standard, a second profile imports it),
  ADR-0029 (its table and `eark/none`), ADR-0015 (the standard is the model's, no longer in the name), ADR-0022
  and ADR-0033 (the MODS vocabulary question is settled: the library's words), ADR-0014
  (its flat single-representation case is gone), ADR-0013 (cited, unchanged). The
  mods-coverage plan's note of 2026-10-06 gets its answer.

### Step 2. The profile pages

`Added: docs/profiles/ugent-basic.md and ugent-bibliographic.md, the written rules of the two UGent profiles`

- One page per profile with the sections of decision 4. Every MUST is either enforced by
  a later step, enforced by the input reader already, or stated as the operator's
  responsibility, and the page says which. Both pages state that a package holds zero or
  more representations, and that the profile generates no PREMIS but carries supplied
  PREMIS documents. A "what RODA does" section per part: the AIP type from the declared
  value, the representation type from the name, descriptive rendering
  (Simple DC renders and indexes by default; MODS needs a crosswalk configured in RODA,
  outside this tool), documentation and schemas mapped into the AIP, no preservation
  metadata from the SIP.
- `docs/README.md`: the `profiles/` tier, design genre.
- Written before the code, so steps 7 to 9 can quote the pages. Each rule says who
  checks it: checked, the operator, or "from step N"; the step that lands a rule turns
  its marker into "checked", and step 4 removes the note that the profile ships under
  its eark name.

### Step 3. Models move to packages named by their standard

`Changed: the Simple DC model moves to profiles/simpledc and the MODS model to profiles/mods; earkdc.Terms is simpledc.Terms and earkmods.Record is mods.Record`

- `git mv` the model files; `profiles/earkdc` and `profiles/earkmods` keep only their
  definitions, importing the models. Package docs: "Package simpledc is the Simple Dublin
  Core model: ..."; "Package mods is the MODS 3.7 model: ...".
- `cli/input/mapping`: `EarkDC` becomes `SimpleDC`, `EarkMods` becomes `MODS`; the file
  names follow. The mapper error strings already interpolate the definition's name.
- Imports and references: registry, `cli/profile.go`, `build/*_test.go`, `build/doc.go`
  and `build/source.go` comments, `sip/description.go`, the input tests, the README's
  library example, the design doc, CLAUDE.md's package map.
- The commit body records the import-path break, as ADR-0015 and ADR-0018 did.
- Output unchanged: `scripts/reference-diff.sh` clean for all three profiles.

### Step 4. The ugent profiles replace eark

`Changed: ugent/basic and ugent/bibliographic replace eark/dc and eark/mods; examples, scripts and docs follow`

- `profiles/ugent/profile.go`: `Basic` and `Bibliographic`, with the package doc stating
  decision 1; the values are today's eark values (profile URL, `Mixed`, `MIXED` until step
  5, no PREMIS, representation types emitted, representation descriptions allowed), each
  with its comment. `profiles/earkdc` and `profiles/earkmods` are removed.
- `profiles/registry.go`: the three keys. `cli/profile.go`: the mappers keyed by the new
  definitions' names.
- `examples/`: `git mv examples/eark/dc examples/ugent/basic` and
  `examples/eark/mods examples/ugent/bibliographic`; `examples/README.md`.
- `build.sh`: default stays `meemoo/basic`; the `case` maps `ugent/basic|ugent/bibliographic`
  to 2.2.0; the error list; header comments. `.gitignore` already covers `*-uuid/`.
  Comments that spell a profile name in `scripts/validate.sh`,
  `scripts/reference-diff.sh` and `scripts/schema-catalog.xml`.
- `tmp/reference/` (local, untracked): `eark/dc/pkg` moves to `ugent/basic/pkg`,
  `eark/mods/pkg` to `ugent/bibliographic/pkg`; a dated paragraph in its README; said in
  the commit message.
- Docs: README (intro, comparison table, the two profile sections, `--profile` examples,
  the input-spec anchor), CONTRIBUTING.md, CLAUDE.md, `docs/input-spec.md`,
  `docs/sip-creator-design.md` (profile table, Profiles section), `docs/TODO.md`,
  `docs/development-roda-ingest.md`, a note line at the top of `docs/plans/eark-writer.md`
  and `docs/plans/mods-coverage.md`. ADRs and `docs/archive/` untouched.
- A program that resolves a profile by name uses the new names; one that holds a
  definition directly imports `profiles/ugent`. The commit body says so.
- Output unchanged: `scripts/reference-diff.sh` clean against the moved copies.

### Step 5. The content type declaration

`Changed: the ugent profiles declare their content type: CONTENTINFORMATIONTYPE OTHER with the profile's name`

- Both definitions: `ContentInformationType: "OTHER"`, `OtherContentInformationType` the
  profile's name (decision 2), with a comment quoting CSIP6 ("When the
  `csip:CONTENTINFORMATIONTYPE` has the value OTHER the
  `csip:OTHERCONTENTINFORMATIONTYPE` must state the content information type") and the
  profile page.
- The package METS only; the representation METS keeps ADR-0013's declaration. A test
  asserts both.
- `./build.sh ugent/basic` and `ugent/bibliographic` VALID with zero warnings. A deliberate
  output change: the reference copies are updated, with a dated note in
  `tmp/reference/README.md` and the commit message.
- The runbook gains a row: the AIP type shows `ugent/basic` or `ugent/bibliographic`, or
  the label RODA's AIP type vocabulary gives it, which is RODA configuration and outside
  this tool.

### Step 6. No flat input folder

`Removed: the flat input folder; essence always sits in a representation folder under representations/`

- `cli/input/walker.go`: `readFlatRepresentation` goes. A folder without
  `representations/` has no representations; a content file or folder beside the
  reserved names is a violation whether or not `representations/` exists, with one
  message naming the entry and `representations/<name>/`. `representations.csv` without
  `representations/` stays a violation, its message without the flat case.
- Until step 8, a folder with no representations is still refused. `check` does not run
  `SourcePackage.Validate`, so the reader refuses a folder without `representations/`
  itself, as it already refuses an empty one; step 8 removes both. Loose content without
  `representations/` is one violation naming the first entries, so a folder laid out the
  old way does not get one line per file.
- Tests: the input tests for the flat case become tests for loose content beside the
  reserved names; a folder holding only `description.csv` reads as zero representations
  and is refused by the engine's rule.
- Output unchanged for the examples, which already use `representations/`:
  `scripts/reference-diff.sh` clean for all three profiles.
- Docs: `docs/input-spec.md` §2 (the flat case goes), README (the input folder
  sections), the design doc.

### Step 7. The representation vocabulary

`Added: Definition.RepresentationTypes, the closed set of representation names a profile allows, one representation per type`

- `build/definition.go`: `RepresentationTypes []string`; nil means no vocabulary
  (`meemoo/basic`, whose spec names none). Doc quoting the profile page. `ValidateSource`:
  a name outside the set adds "profile %q names its representations %s; %q is not one of
  them"; a `Type` set and different from the name adds "under profile %q a
  representation's type is its name; %q has type %q". Two representations of one name are
  already refused by uniqueness.
- `cli/input`: the folder's name is the name, so the check reaches the operator through
  `check`, and the message lists the allowed names; a `representations.csv` row whose
  `type` differs from the folder is reported with its line.
- Both UGent profiles list the three names.
- `examples/ugent/*`: representation folders renamed into the vocabulary. The output
  changes (directory names, `OBJID`, labels where they followed the name): reference
  copies updated and noted.
- Tests: `build/definition_test.go` (in and out of the set, type equal and different, no
  vocabulary), `cli/input` (the message names the file and the line),
  `profiles/registry_test.go` (which profiles carry a vocabulary).
- Docs: `docs/input-spec.md` §2 (the names per profile), the profile pages, README,
  design doc.

### Step 8. The representation minimum

`Changed: MinRepresentations on the definition replaces the engine's at-least-one rule; the ugent profiles allow a package without representations`

- `build/definition.go`: `MinRepresentations int` next to `MaxRepresentations`: "the
  number of representations a package needs at least; zero allows a package without any,
  a metadata-only package (CSIP58; the E-ARK SIP 2.2.0 introduction: a package with zero
  representations carries metadata only, to update the metadata of a package ingested
  earlier)". `MaxRepresentations`'s doc loses "With the one representation every package
  needs". `ValidateSource`: fewer than the minimum adds "profile %q needs at least %d
  representation(s), the package has %d".
- `build/source.go` `Validate`: drop "no representations supplied"; a representation that
  exists must still have files. The `Representations` doc says the profile decides how
  many a package needs.
- Profiles: `ugent.Basic` and `ugent.Bibliographic` 0, with a comment citing CSIP58 and
  the profile page (a package may describe an intellectual entity whose content is not
  in the archive, new or as an update); `meemoo.Definition` 1 next to its maximum.
- `cli/input/walker.go`: `readRepresentations` reads an empty `representations/` as zero
  representations instead of a violation, and a folder without `representations/` and
  without loose content is no longer a violation either (step 6 added it for `check`).
  Loose content at the top level stays a violation. An empty representation folder stays a
  violation; the `representations.csv` rules are unchanged. Comments say the count is
  the profile's verdict, in `ValidateSource`.
- `build/write.go` `writeSkeleton` unchanged: `representations/` is always created,
  because commons-ip 2.11.2 warns when the folder is missing (CSIPSTR9) and passes a
  package with no Representations file group (CSIP114 checks only the groups that
  exist).
- Tests: `build/assemble_test.go` loses the "no representations" case of its `Validate`
  table; `build/definition_test.go` gains "needs at least 1" for `meemoo/basic` and zero
  accepted for both UGent profiles; a new `TestBuildMetadataOnly` under each UGent
  definition (no
  `fileGrp USE="Representations/..."`, no representation division, Schemas and Metadata
  kept, `representations/` present and empty, the zip carries the directory entry); the
  input tests read a folder holding only `description.csv` as zero representations
  without a violation.
- Verification: a folder holding only a `description.csv`, built with
  `./build.sh ugent/basic <folder>` and `./build.sh ugent/bibliographic <folder>`, is
  VALID; the same under `meemoo/basic` is refused by `check` with "needs at least 1
  representation(s)".
- Docs: `docs/input-spec.md` §2 and the update section (a package without
  representations, new or as an update with `--status` and `--updates`), both profile
  pages, README
  ("Updating an earlier package"), CONTRIBUTING (validating a metadata-only package with
  build.sh's second argument), the design doc.

### Step 9. The identifier's syntax: dropped (decided 2026-10-07)

Not built. The tool takes an identifier as a string, with the place it came from, and
does not know the rules of the system that issued it: checking that an MMS ID is well
formed is the operator's responsibility, as the profile page says. A record that comes
from the catalogue carries a well-formed identifier already, and a check could not
reach a supplied `mods.xml`. The proposal, kept for the record:

- The definition speaks `sip.Description`, so the identifier is read through an optional
  interface, `build.Identified` with `Identifier() string`, the way `IdentifierSwapper` is
  (ADR-0024); `mods.Record` and `simpledc.Terms` implement it. `IdentifierPattern string`,
  a regular expression, empty for no check, with `IdentifierName string` for the message
  ("an Alma MMS ID"). `ugent/bibliographic` states the MMS ID's syntax as Alma issues it;
  `ugent/basic` states none.
- The package-level description only; a representation's description may state no
  identifier.
- Whether the step is worth its field is decided when step 8 is done: records that come
  from the catalogue already carry a well-formed identifier, so the check catches a typed
  one in `description.csv` and nothing else.

### Step 10. Housekeeping

`Changed: TODO and the runbook reflect the ugent profiles`

- `docs/TODO.md`: the validator status rows name the UGent profiles.
- `docs/input-spec.md` §4: the sentence that commons-ip warns when a representation has no
  `documentation/` folder. Found in step 7: the examples' `access/` has none and commons-ip
  2.11.2 reports zero warnings. Check what it reports, then correct the sentence.
- `docs/development-roda-ingest.md`: the profile name, the two new rows for the first
  ingest (AIP type, representation types).

## Verification

- `go test ./...` after every step.
- `./build.sh meemoo/basic` (2.0.4), `ugent/basic` and `ugent/bibliographic` (2.2.0)
  report VALID with zero warnings from step 4 on.
- `scripts/reference-diff.sh` clean after steps 3, 4 and 6 (output unchanged, copies
  moved in step 4); updated copies after steps 5 and 7, each with a dated note in
  `tmp/reference/README.md` and in the commit message.
- Step 6 by hand: a folder with a content file beside `description.csv` is refused by
  `check`, the message naming the file and `representations/<name>/`.
- Step 7 by hand: a folder with `representations/master/` under `ugent/basic` is refused
  by `check` with the allowed names; a `representations.csv` whose `type` differs from the
  folder is refused with its line.
- Step 8: the package without representations is VALID under both UGent profiles;
  under `meemoo/basic` refused.
- The first real ingest into UGent's RODA ([runbook](../development-roda-ingest.md))
  confirms the AIP type and the three representation types as RODA shows them.

## Open questions

Settled 2026-10-07: the content type as a plain value, not a URI (decision 2); the
meaning of the three representation types and that both UGent profiles allow all three
(decision 6); zero or more representations under both UGent profiles, new or as an
update (decisions 7 and 8); no flat input folder, for every profile (decision 9); no
PREMIS generated, supplied PREMIS accepted (decision 7).

1. **One package `profiles/ugent` with two definitions, or one package per profile.** The
   plan takes one package: the two definitions share their declaration values and differ
   in model and declared content type.
2. **Whether Meemoo's model moves to `profiles/dcschema`** for symmetry with decision 3.
   Not now: it has one user.
