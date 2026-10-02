# Plan: the metadata model

*Status: **proposed** (2026-10-02). Nothing implemented yet.*

This plan names the concept the descriptive side of the tool is built around and makes
the code use that name. It started in the [cli cleanup plan](../archive/cli-cleanup.md), when
`input.DocumentVocabulary` turned out to be no vocabulary at all and became
`input.DocumentFormat`, next to the library's `build.DocumentFormat`.

## The concepts

> A profile has a **metadata model**; a **description** is an instance of it; a
> description is stored in the model's **document format**, or supplied as rows in the
> model's **vocabulary**.

- **Metadata model**: what the tool knows about one kind of description: which fields
  exist and what they mean, how a description is written as a document, and which XSDs
  that document points at. There are three: Meemoo's dc+schema.org model, Simple
  Dublin Core, and MODS 3.7. "Model" rather than "standard", because Meemoo's
  dc+schema.org is a data model Meemoo designed, not a universal standard; MODS and DC
  are standards, but each defines a model too.
- **Description**: one instance of a metadata model: the description of one
  intellectual entity (OAIS's descriptive information, the content of a METS
  `dmdSec`). `sip.Description`, and the types behind it: `meemoo.Terms`,
  `eark.Terms`, `earkmods.Record`, `build.DescriptiveDocument`.
- **Document format**: how a model is stored as a file: the root element, namespace
  and version that make a file a document in the model (`simpledc` without namespace,
  `mods:mods` declaring 3.7), and the file's name. The term follows OAI-PMH, where a
  metadata format is a prefix, a schema and a namespace.
- **Vocabulary**: the keys an input folder's description.csv may use and what each one
  means under the model. The CLI's side only.

"Model" means the data model only. A value is a description *in* a model, not "a
model". The `sip/` package stays the "domain model" (package, entity,
representation, file); the qualifier "metadata" keeps the two apart.

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

## Step 1: rename to the concepts

Renames only; no change in behavior or output.

| Now | New | Why |
|---|---|---|
| `build.DescriptionEncoder` | `build.MetadataModel` | What `dcschema`, `simpledc` and `mods` are. |
| `Definition.Encoder` | `Definition.Model` | The profile's metadata model. |
| `Definition.DescriptiveName` | `Definition.DocumentName` | The document's file name; matches `input.DocumentFormat.DocumentName`. |
| `Check(d)` | `ValidateType(d)` | Returns why `d` is not a description in this model; CLAUDE.md names a function that returns an explanatory error `Validate…`. |
| Comments that call a description value "a model" | "a description in the model" | "Model" means the data model only. |

Stays as it is: `build.DocumentFormat`, `build.IdentifierSwapper`, `Encode`, `Schemas`,
the implementation types (`dcschema`, `simpledc`, `mods`, already named after their
model), `sip.Description` and the description types, `MetsDeclaration.DescriptiveMDType`
(METS's own term, used where the METS is written), and `input.Vocabulary`.

Touches `build/`, the three profile packages, `cli/input/vocabulary`, the README's
library section, `build/example_test.go`, CLAUDE.md and `sip-creator-design.md`. The
library's API changes; the project is not in use yet, so no compatibility names.

## Step 2: one value per model (decide in review)

The file name and the METS label belong to the model too, but live in `Definition`
and `MetsDeclaration`. Two ways:

- **Leave them where they are.** Each profile's `profile.go` already sets the model,
  the file name and the METS label side by side, and `MetsDeclaration` is the data the
  METS templates read ([ADR-0020](../decisions/0020-profile-is-builder-configuration.md):
  a profile is builder configuration, as data).
- **Move them into `MetadataModel`** (`DocumentName()`, `MDType()`), and have the
  engine copy the label into the declaration it hands the METS encoder. One place per
  model, but a structural change against ADR-0020's data-first shape.

Whichever is chosen is recorded in an ADR.

## Step 3: the name of the supplied document

`build.DescriptiveDocument` is a description supplied as a file in the model's
document format. It keeps "descriptive", which this plan moves away from, but no
better name has come up yet; `SuppliedDocument` was rejected earlier because it reads
as the file rather than the description in it. Decide with step 1 or leave it.

## Step 4: the input side of a model

`cli/input/vocabulary` holds `Meemoo`, `Eark` and `EarkMods`: for each profile, the
input folder's side of its metadata model (the vocabulary for the rows, and for the two
eark profiles the document format). `input.Vocabulary` stays: giving the rows their
meaning is what a vocabulary does. Whether the package, now named after one of its two
parts, should be named after the model instead is the open question in
`docs/TODO.md`; decide it here.

## Acceptance

Every step leaves `go test ./...`, the commons-ip validation in `build.sh` (all three
profiles VALID) and the structural comparison in `scripts/reference-diff.sh` passing
with no change to the reference copy: none of this changes a generated package.

## When this plan ships

Write an ADR for the vocabulary (metadata model, description, document format,
vocabulary), noting that it gives ADR-0015's "world" its name in the code. Add the
definitions to `sip-creator-design.md`, update CLAUDE.md's "System shape", and move
this plan to `docs/archive/`.
