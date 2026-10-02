# Plan: the metadata model

*Status: **draft** (2026-10-02), revised after a first review. Nothing implemented yet.*

This plan names the concept the descriptive side of the tool is built around and makes
the code use that name. It started in the [cli cleanup plan](../archive/cli-cleanup.md), when
`input.DocumentVocabulary` turned out to be no vocabulary at all and became
`input.DocumentFormat`, next to the library's `build.DocumentFormat`. The first review
went further: the word "vocabulary" leaves the code, and the CLI's `input.Statement`
turned out to be a copy of `sip.Term`.

## The concepts

> A profile has a **metadata model**; a **description** is an instance of it. A
> description is written as an XML document in the model's **document format**, or
> supplied as **terms** in a description.csv.

- **Metadata model**: what the tool knows about one kind of description: which fields
  exist and what they mean, how a description is written as a document, and which XSDs
  that document points at. There are three: Meemoo's dc+schema.org model, Simple
  Dublin Core, and MODS 3.7. "Model" rather than "standard", because Meemoo's
  dc+schema.org is a data model Meemoo designed, not a universal standard; MODS and DC
  are standards, but each defines a model too.
- **Description**: one instance of a metadata model: the description of one
  intellectual entity (OAIS's descriptive information, the content of a METS
  `dmdSec`). `sip.Description`, and the types behind it: `meemoo.Terms`,
  `eark.Terms`, `earkmods.Record`, `build.DescriptiveDocument`. Which fields exist and
  what rules they follow live on these types, not on the model's interface.
- **Document format**: how a model is stored as a file: the root element, namespace
  and version that make an XML file a document in the model (`simpledc` without
  namespace, `mods:mods` declaring 3.7), and the file's name. The term follows
  OAI-PMH, where a metadata format is a prefix, a schema and a namespace.
- **Term**: one key, an optional language tag and a value: one line of a
  description.csv, and the shape `sip.Term` already has. Dublin Core uses the word
  for its properties ("DCMI Metadata Terms", the `dcterms:` namespace).

Types are named by their role; comments and docs say which syntax they read (XML, CSV).
A name like `XMLFormat` or `CSVFormat` was considered and rejected: the types never see
the syntax (`xmldoc.Root` parses the XML, `cli/input/csv.go` the CSV), "XML" does not
say which namespace and version make a document, and naming both by syntax hides that
the document is the standard's own form while description.csv is this tool's input
convention.

"Model" means the data model only. A value is a description *in* a model, not "a
model". The `sip/` package stays the "domain model" (package, entity,
representation, file); the qualifier "metadata" keeps the two apart.

### Why not "vocabulary"

In METS, PREMIS, CSIP and SKOS a vocabulary is a controlled list of values; the CLI
already uses the word that way for `--status` (the SIP3 vocabulary) and
`--content-category` (the CSIP vocabulary). The types in `cli/input/vocabulary` hold no
such list. For `basic` and `eark` they copy the rows into terms, and the key tables
live in `profiles/meemoo` and `profiles/eark`. Only `EarkMods` has a table of its own,
and it says where each key's value goes in the record, not which terms are allowed.

## Context

ADR-0015 named this concept a **world**: "everything the tool knows about one
standard: a terms type, a closed vocabulary table, the validation of those terms, the
templates that render them, and the list of bundled XSDs". ADR-0018 moved the worlds
into the profile packages, and the concept stopped having a name in the code. Its
parts now sit in four places:

| Part of the model | Where it lives now |
|---|---|
| Check a description's type, write it as a document, list the XSDs | `build.DescriptionEncoder` (`Check`, `Encode`, `Schemas`), implemented by `dcschema`, `simpledc`, `mods` |
| Recognize a supplied file as a document in the format | `build.DocumentFormat.ValidateDocumentRoot` |
| The document's file name | `Definition.DescriptiveName` |
| The METS label for the model | `MetsDeclaration.DescriptiveMDType` and `DescriptiveMDTypeVersion` |

Two words carry too much. "Description" and "descriptive" appear at every level
(description.csv, `sip.Description`, `DescriptionEncoder`, `DescriptiveDocument`,
`DescriptiveName`, `DescriptiveMDType`), so they no longer say which level a name is
at. "Encoder" names one of the interface's three methods.

### A gap the names leave: Meemoo's document format

By the definition above every model has a document format, Meemoo's included
(`dc+schema.xml` in the hetarchief 1.2 namespace). But `build.DocumentFormat` is
optional, and Meemoo's `dcschema` does not implement it, on purpose: Meemoo's document
must carry the entity identifier the build mints (ADR-0021). What the interface means
is narrower than its name: recognizing a *supplied* file as a document in the format.
The plan keeps the name and says this on the interface and in the ADR, so nobody
implements it for Meemoo to make the names line up.

## Step 1: rename to the concepts

Renames only; no change in behavior or output.

| Now | New | Why |
|---|---|---|
| `build.DescriptionEncoder` | `build.MetadataModel` | What `dcschema`, `simpledc` and `mods` are. Its doc comment points to the description type for the fields and their rules. |
| `Definition.Encoder` | `Definition.Model` | The profile's metadata model. |
| `Definition.DescriptiveName` | `Definition.DocumentName` | The document's file name. |
| `Check(d)` | `ValidateType(d)` | Returns why `d` is not a description in this model; CLAUDE.md names a function that returns an explanatory error `Validate…`. Pairs with `ValidateDocumentRoot`. |
| `build.DescriptiveDocument` | `build.EncodedDescription` | See step 3. |
| Comments that call a description value "a model", or the model "the encoder" (e.g. `validateDocumentRoot` in `cli/input/vocabulary`) | "a description in the model", "the model" | "Model" means the data model only. |

Stays as it is: `build.DocumentFormat`, `build.IdentifierSwapper`, `Encode`, `Schemas`,
the implementation types (`dcschema`, `simpledc`, `mods`, already named after their
model), and `sip.Description` and the description types.

Touches `build/`, the three profile packages, `cli/input/vocabulary`, the README's
library section, `build/example_test.go`, CLAUDE.md and `sip-creator-design.md`. The
library's API changes; the project is not in use yet, so no compatibility names.

## Step 2: the METS label moves into the model

The file name and the METS label both sit outside the model today. They are different
kinds of value, so they go different ways.

- **The METS label is a fact about the document.** The MODS version is written twice
  now: `version = "3.7"` in `profiles/earkmods/encoder.go` (the version the template
  writes and a supplied document must declare) and `DescriptiveMDTypeVersion: "3.7"` in
  `profiles/earkmods/profile.go`. Nothing ties them together. Change one and the
  dmdSec declares MODS 3.7 over a document of another version, which CSIP forbids and
  which no check in this repository catches. A profile written outside the module
  (ADR-0022) can pair `simpledc{}` with `MDTYPE="MODS"` without an error. The model
  gets two methods, `ModelType()` and `ModelTypeVersion()`, named for the model
  rather than after the METS attributes, and the assembler sets them on the
  descriptive file node (`sip.File.MDType` and `MDTypeVersion`, the METS side,
  which keeps METS's names), at the package and in each representation, where the
  dmdSec's mdRef reads them next to the file's `Mime`. `MetsDeclaration` loses
  `DescriptiveMDType` and `DescriptiveMDTypeVersion` and holds the profile's METS
  values only, so a profile has no field in which to set another model's label.
  (Review chose this over keeping the fields on `MetsDeclaration` and having the
  engine overwrite them, which left a field a profile could set and see ignored.)
- **The file name is a convention of the package.** `dc+schema.xml` is set by the
  Meemoo spec; `dc.xml` and `mods.xml` are choices of the eark profiles. It stays on
  `Definition` as `DocumentName` (step 1).

This stays within [ADR-0020](../decisions/0020-profile-is-builder-configuration.md)
(a profile is builder configuration, as data): the profile still declares its values,
and the model states facts about its own document. The new ADR records the split.

## Step 3: the name of the supplied document

`build.DescriptiveDocument` is a description supplied as a file in the model's
document format. Proposed: **`build.EncodedDescription`**. `Encode` turns a description
into a document; this type is a description that arrives already encoded. The name says
it is a description (it implements `sip.Description`), not the file, which was the
objection that ruled out `SuppliedDocument`. Done together with step 1, so the API
changes once.

## Step 4: the input side

`cli/input/vocabulary` holds `Meemoo`, `Eark` and `EarkMods`, which do two jobs for
their profile: turn the rows of a description.csv into a description, and, for the two
eark profiles, name and recognize a supplied document. The step takes the second job
out, folds the CLI's own row type into `sip.Term`, and renames what is left.

### 4a. The document half comes from the profile

`Eark.DocumentName` and `Eark.ValidateDocumentRoot` (and the same pair on `EarkMods`)
only forward to the definition's file name and to the model's `build.DocumentFormat`.
`cli/profile.go` already holds the definition, so it builds the document half from it
and passes it to `input.Read` as a value: the file name and the model's
`build.DocumentFormat`, or nothing for a profile whose model takes no supplied
document. `input.DocumentFormat` goes away, and the file name is written in one place.
`cli/input` still imports no profile package and takes no `build.Definition`
([ADR-0023](../decisions/0023-cli-input-one-package.md)); `check` still needs no
configuration (ADR-0010).

### 4b. `input.Statement` becomes `sip.Term`

`input.Statement` is `sip.Term` plus the line it was read from, and
`input.StatementError` (a finding by line) mirrors `sip.TermError` (a finding by index).
`cli/input/description.go` already turns a `TermError`'s index back into a line.

- The CSV parser returns `[]sip.Term`, and the walker keeps each term's line beside it,
  unexported.
- The per-profile type takes `[]sip.Term` and reports a finding about one term as a
  `*sip.TermError` by index. `EarkMods`'s placement findings (an unknown key, a repeated
  title) become `TermError`s, which is what they describe.
- `input.Statement`, `input.StatementError` and the `terms` copy in
  `cli/input/vocabulary/vocabulary.go` are removed.
- `sip.Term`'s doc comment drops "a key from the profile's vocabulary" and
  "statement": a term is a key, an optional language tag and a value.

### 4c. Name what is left (open)

After 4a and 4b, `input.Vocabulary` is one method: turn a level's terms into the
model's description. For `basic` and `eark` that is a type conversion
(`eark.Terms(terms)`); only `eark-mods` places terms into fields of a record. The
interface and the package `cli/input/vocabulary` need a name for that job. Considered
so far and rejected:

| Name | Why not |
|---|---|
| vocabulary | A controlled list of values in this field; see above. |
| row format, statement format, entry format | "Row" ties the type to CSV; "statement" implies linked data; "entry format" does not say what it is. Each named the row type that 4b removes. |
| CSV format | Named by syntax; the type never sees CSV. |
| crosswalk | Implies two schemas. Under `basic` and `eark` the keys are the model's own element names, so nothing is crossed. |

Since the job takes terms in and returns a description, a name can start from
there. Also open: whether an interface is still needed, or a function value
(`func([]sip.Term) (sip.Description, []error)`) per profile is enough. Decide in review,
after 4a and 4b, because they decide what the type takes.

### Prose

The input specification and README call the key tables "vocabularies" and the rows
"statements" ("The vocabularies are separate", "flat statements about the record").
These become "the profile's keys" or "key table", and "terms". ADRs keep their text.

## Order

1. Steps 1, 2 and 3 together: one change to the library's API.
2. Step 4a, then 4b: CLI only.
3. Step 4c once the name is chosen.

## Acceptance

Every step leaves `go test ./...`, the commons-ip validation in `build.sh` (all three
profiles VALID) and the structural comparison in `scripts/reference-diff.sh` passing
with no change to the reference copy: none of this changes a generated package. CLI
messages keep their wording and line numbers; `cli/input` tests pin them.

## When this plan ships

Write one ADR for the concepts (metadata model, description, document format, term),
noting that it gives ADR-0015's "world" its name in the code, why "vocabulary" left the
code, and the split in step 2 (the label is the model's, the file name the profile's).
Add the definitions to `sip-creator-design.md`, update CLAUDE.md's "System shape" (it
names `input.Vocabulary`, `input.DocumentFormat` and the vocabularies), close the item
in `docs/TODO.md`, and move this plan to `docs/archive/`.
