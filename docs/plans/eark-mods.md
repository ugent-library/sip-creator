# Plan: MODS 3.7 descriptive metadata for the plain E-ARK output

*Status: **agreed, not started** (2026-09-15). Design decisions settled in
review; the MODS element list beyond identifier and title is still open and
does not block the first steps. Update this line as steps land.*

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
   library callers never set attributes, as today (ADR-0011). Consequence:
   the flat model cannot express given and family name parts, corporate
   versus personal names, date ranges with start and end points, or a
   subject with several sub-terms. That is acceptable because the source
   data is flat columns; richer records arrive as supplied documents.
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
   inside each representation directory. One source per level, one standard
   per folder. The folder stays self-describing, so `check` keeps taking no
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
- The profile chosen at `create` must match the folder's standard: `basic`
  and `eark` take DC, `eark-mods` takes MODS. `basic` takes rows only.

## Execution steps

Library first. Every step ends with `go test ./...` green and
`./build.sh basic` and `./build.sh eark` VALID with 0 warnings.

- **S1: docs first.** This plan; ADR drafts
  `0015-descriptive-worlds-dc-and-mods.md` (decisions 1, 2, 4, 5) and
  `0016-descriptive-input-rows-or-supplied-document.md` (decisions 3, 6, 7,
  8, 9).
- **S2: pure refactor, output unchanged.** Capture the current `eark`
  output outside the repo first, since `tmp/reference/pkg` covers `basic`
  only. Rename `encoders/metadata` to `encoders/dc`; add the `Description`
  interface to `sip/`; resolve `Family` to the internal struct with the
  world check running before validation; required elements become plain
  keys; schema list on `Definition`; `Input` and `SourceRepresentation`
  gain the document field and the one-of check; the writer copies a
  descriptive file with `Source` set. Design doc and CLAUDE.md system shape
  follow the rename. Acceptance: the structural comparison in
  scripts/reference-diff.sh clean for `basic` against `tmp/reference/pkg`
  and for `eark` against the captured copy.
- **S3: the mods world and the eark-mods profile.** `schemas/mods-3-7.xsd`;
  `encoders/mods` with terms, the two-row table, validation, the template
  and `ValidateDocument`; the `eark-mods` family and registry entry. Go
  tests: table invariants, template output (root, namespaces, version,
  escaping, `xml:lang`, refuses invalid terms without writing), world
  mismatch at build writes nothing, meemoo refuses a supplied document,
  schema set per profile, and a first test for `encoders/mets` asserting
  the dmdSec carries `MDTYPE` and `MDTYPEVERSION` from the declaration.
  The library route is complete after this step.
- **S4: CLI rows.** `cli/input` learns `dc.csv` and `mods.csv`: shared row
  reading and key parsing, one small builder per world, the one-source and
  one-standard rules, the `metadata.csv` violation. Fixtures rename their
  `metadata.csv` to `dc.csv`. Input spec §3 and §7, README.
- **S5: CLI supplied documents.** `dc.xml` and `mods.xml` through the
  world's `ValidateDocument`, violations naming the file. Input spec §3 and
  §8 (the deferred operator-supplied XML item is now this), README.
- **S6: acceptance and closing docs.** `tmp/eark-mods/` fixture (a copy of
  `tmp/eark` with `mods.csv`); build.sh gains the `eark-mods` case against
  E-ARK 2.2.0 and an xmllint pass over every emitted `mods.xml` against the
  bundled XSD, with xmllint added to the documented requirements. Check
  that the package METS dmdSec reads `MDTYPE="MODS" MDTYPEVERSION="3.7"`.
  README profile list, design doc, TODO (the MODS question is answered),
  this plan's status line, then archive the plan per the docs lifecycle.

## Open questions

- **The `type` attribute on `mods:identifier`** for the MMS ID: one constant,
  decided by the owner of the repository side before S3 ships or changed
  afterwards in one place.
- **Further MODS rows** and their fixed attribute values (name type, role
  vocabulary, date encoding): supplied with the repository's index fields,
  each a table row.
- **`eark` still ships `descriptive_basic.xsd`**, which it never references.
  Dropping it is a deliberate output change and stays out of this plan.
- **Library callers** change their import from `encoders/metadata` to
  `encoders/dc` in S2; the commit message records the rename.
