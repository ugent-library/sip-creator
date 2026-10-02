# 0024 — The metadata model: named concepts, and the library speaks the standard's names

Status: **Accepted** (2026-10-02, when the [metadata-model
plan](../archive/metadata-model.md) shipped). Revises in part
[ADR-0007](0007-profile-families-share-one-writer.md),
[ADR-0011](0011-closed-descriptive-vocabulary.md),
[ADR-0021](0021-descriptive-model-follows-its-standard.md) and
[ADR-0023](0023-cli-input-one-package.md).

## Context

[ADR-0015](0015-descriptive-worlds-dc-and-mods.md) named the descriptive side a
**world**: everything the tool knows about one standard. [ADR-0018](0018-engine-and-profile-packages.md)
moved the worlds into the profile packages, and the concept lost its name in the code.
Its parts sat in four places (`build.DescriptionEncoder`, `build.DocumentFormat`,
`Definition.DescriptiveName`, `MetsDeclaration.DescriptiveMDType`), and two words
carried too much: "description" and "descriptive" appeared at every level, and
"encoder" named one of three methods.

The CLI side used "vocabulary" for the types that turn description.csv into a
description. In METS, PREMIS, CSIP and SKOS a vocabulary is a controlled list of values,
and the CLI already used the word that way for `--status` and `--content-category`.

Two inconsistencies came out of the review. The MODS version was written twice, in the
template and in the profile's METS values, with nothing tying them together. And
Meemoo's library type was keyed by the CSV's spelling (`ispartof`), not by the element
Meemoo's specification names (`dcterms:isPartOf`), while the MODS keys lived in the CLI.

## Decision

**Four concepts, named in the code.** A profile has a **metadata model**; a
**description** is an instance of it; a description is written as an XML document in
the model's **document format**, or supplied as **terms** in a description.csv.

- Metadata model: `build.MetadataModel` (`ValidateType`, `Encode`, `Schemas`,
  `ModelType`, `ModelTypeVersion`), on `Definition.Model`. The fields and their rules
  live on the description type, not on the interface.
- Description: `sip.Description` and its types; a finished document supplied as a file
  is `build.EncodedDescription`.
- Document format: `build.DocumentFormat`, the optional part of a model that accepts a
  supplied document. Every model has a document format; only a model that takes
  supplied documents implements the interface.
- Term: `sip.Term`, one key, an optional language and a value.

Types are named by their role; comments and docs say which syntax they read (XML, CSV).
The CLI's mapping of terms onto a description is `input.Mapper` with `Map`, one type
per profile in `cli/input/mapping`; `cli/profile.go` pairs each profile with its mapper.
"Model" means the data model only; `sip/` stays the domain model.

**The model owns its METS label.** `ModelType` and `ModelTypeVersion` name the model as
the dmdSec types the document (`MDTYPE`, `MDTYPEVERSION`). The assembler sets them on
the descriptive file node (`sip.File.MDType`, `MDTypeVersion`), where the mdRef reads
them; `MetsDeclaration` holds the profile's METS values only. A profile cannot pair a
model with another model's label. The document's file name stays a convention of the
profile, `Definition.DocumentName`.

**The library speaks the standard's names; the CLI owns the CSV's keys.** A
description is keyed by the element's name as its standard spells it: `dcterms:title`
for Meemoo, `title` for Simple Dublin Core, a typed field for MODS. Each profile's
mapping in `cli/input/mapping` maps the CSV's keys onto those names (the identity for
Simple Dublin Core). The rules of a specification (required elements, cardinality,
Meemoo's Dutch entry) stay in the library, so findings about a whole description name
the element; the input specification's key table pairs each key with its element.
The document half of the input comes from the definition too: `input.Read` takes an
`input.Document` built from `DocumentName` and `Model`.

## Alternatives rejected

- **"Standard" or "world" for the concept.** Meemoo's dc+schema.org is a data model
  Meemoo designed, not a universal standard; "world" says nothing to a new reader.
- **Other names for the mapping**: "vocabulary" (a controlled list in this field), "row
  format", "statement format", "entry format", "CSV format" (named by syntax; the type
  never sees CSV), "crosswalk" (implies two schemas; under eark nothing is crossed),
  "adapter" (names a pattern, not what the type does).
- **Keeping the label on `MetsDeclaration` and overwriting it from the model.** It left a
  field a profile could set and see ignored.
- **Moving the Meemoo profile out of the library.** The key spelling would move with
  it, and a system that automates ingest into Meemoo would lose its library route.
- **Translating element names back to CSV keys in the CLI's findings.** More code and a
  second method on `Mapper`, only to keep the old wording.

## Consequences

- The library's API changed throughout (`MetadataModel`, `Definition.Model`,
  `DocumentName`, `EncodedDescription`, element-keyed `meemoo.Terms`). The project was
  not in use, so there are no compatibility names.
- A profile written outside this module implements `ModelType` and `ModelTypeVersion`
  and keys its flat terms by element names.
- A new Meemoo element is a row in `profiles/meemoo/elements.go`, a key in
  `mapping.Meemoo` and a line in the input specification; a test pins that every
  element the mapping names is one the library accepts.
- A term the mapper refuses is kept in place as written, so indexes still name rows,
  and `Read` reports one finding per term.
- ADR-0007's dmdSec typing is data on the model, not on the profile. ADR-0011's
  closed table now holds elements, and its keys live in the CLI. ADR-0021's "in a flat
  world the key is the model, so the profile package keeps its key table" no longer
  holds for Meemoo. ADR-0023's `cli/input/vocabulary` is `cli/input/mapping`.
