# 0021 — The descriptive model follows its standard; the CSV owns its vocabulary; a supplied document is a second description

Status: **Proposed** (drafted 2026-09-30; agreed in review the same day
with the [descriptive-model plan](../plans/descriptive-model.md). Becomes
Accepted when that plan ships.) Supersedes
[ADR-0017](0017-supplied-descriptive-document-deferred.md). Revises in part
[ADR-0011](0011-closed-descriptive-vocabulary.md),
[ADR-0015](0015-descriptive-worlds-dc-and-mods.md) and
[ADR-0016](0016-descriptive-input-rows-or-supplied-document.md); each
carries a dated note.

## Context

A description is a flat list of statements in the library: `sip.Term` is a
key, a language tag and a value, and every profile's description is a list
of those with a table mapping each key onto the element it emits
(ADR-0011, ADR-0015). The CLI decodes `description.csv` without knowing the
profile and hands the list to `Definition.NewDescription`, a constructor the
profile provides as data (ADR-0018, note of 2026-09-29).

That model is faithful for meemoo's dc+schema document and for Simple
Dublin Core: both are a sequence of elements with `xml:lang` and an
occasional `xsi:type`. MODS is a tree. A name carries parts and a role, a
subject combines a topic and a place, a title has a type and a non-sorting
part. ADR-0015 conceded the limit: one key emits one complete element with
fixed attributes, and nothing can express several parts under one parent.
The record's physical copies were the first thing that did not fit, so the
MODS description became terms plus a second list, and carrying that list
through a reader that knows no profile called for a neutral item shape in
`sip/` and a widened constructor (the eark-mods plan's S4). Every further
nested element would repeat that exercise. The flat statement is the
two-column CSV's shape living inside the library.

A review on 2026-09-30 of how other tools handle flat input against a rich
schema found one pattern: the model is nested where the standard is nested,
the transport is flat where the source is flat, and the mapping between the
two belongs to the transport and is usually declared as data. commons-ip
takes descriptive metadata as a finished file. Archivematica takes flat
Dublin Core columns or supplied XML. Darwin Core Archive joins several flat
tables on an identifier. Catmandu builds nested records from rows with
path expressions and renders them through templates. RODA's metadata
templates declare their fields and validate the result against an XSD.

## Decision

**The model follows its standard.** Where the standard is a tree, the
profile's description is a typed struct: `earkmods.Record` holds its
identifier, its titles, its items and, as they are needed, its names and
dates as fields, with attribute values as typed constants. Where the
standard is a flat list, the description stays a list of `sip.Term`:
`meemoo.Terms` and `eark.Terms`. `sip.Description`, with `Validate` and
`ValidateRequired`, stays the one interface the engine speaks, and the
templates walk whatever shape the world has.

**The CSV owns its syntax, its vocabulary and its mapping, on the CLI
side.** One adapter per profile under `cli/input` imports its profile
package and builds the profile's description from decoded rows, reporting
each finding with the row's line. The generic reader imports no profile
package; the adapters do. In a flat world the key is the model, so the
profile package keeps its key table for the template and the adapter's
mapping is the identity. In a tree world the adapter holds the key table
and maps each key onto a field of the model. `build.Definition` carries no
constructor for a transport: `NewDescription` goes, and the CLI pairs each
profile with its adapter in its own table.

**Three kinds of rules, three homes.** File rules (the header, `key[lang]`,
an unknown key, a second row for a single-valued key) are the CSV's and
live in the CLI. Document rules the profile's specification states (a
required identifier and title, cardinality, one identifier, meemoo's Dutch
entry, no empty value) live in the model's `Validate`, because a library
caller must satisfy them too and the tool must not render a document it
knows the specification rejects. Schema validity and meaning stay external
([ADR-0003](0003-validation-stays-external.md)). The CSV route is narrow
and exact: it builds the model or refuses with a line number, and it never
approximates. ADR-0011's closed vocabulary stands for the CSV.

**A supplied document is a second description type, for the two eark
profiles.** `build.DescriptiveDocument` is a finished descriptive document supplied
as a file. It implements `sip.Description`: `Validate` reads it once,
requires well-formed XML and captures the root element;
`ValidateRequired` trusts it, since nothing in the eark profiles reads an
identifier back out of the descriptive metadata. The engine reads a
document's root element once and asks the profile's encoder, through the
optional `DescriptiveDocumentChecker` interface (the idiom `IdentifierSwapper` set),
whether that root is its standard's (`simpledc` without namespace for
`eark`; `mods:mods` in the MODS v3 namespace carrying `version="3.7"` for
`eark-mods`); an encoder without the interface takes no document. The
writer copies a document with the store's streamed copy, fixity computed
on the way as for essence, where it would render a model. The assembler
does not branch and the encoders never see a document. The root reader is
shared with the received-PREMIS check. The `basic` profile's encoder is no
`DescriptiveDocumentChecker`: meemoo's document must carry the entity identifier the
tool mints at build, which the swap writes in, and the tool does not edit
XML.

**The CLI takes a document per level.** `dc.xml` under `eark` and
`mods.xml` under `eark-mods`, at the input root or inside a representation
directory, exactly one of the document or `description.csv` per level.
Under `basic` a document is a violation. The rows of `description.csv`
are flat statements about the level's entity and nothing else: a record's
copies, and any other structure, reach the package through the library's
typed record or a supplied `mods.xml`. There is no second table.

## Alternatives rejected

- **Grow the flat table, one row per shape** (`author`, `promotor`,
  `namecorporate-author`). Combinatorial, and it can never express two
  parts under one parent. The item list was the first special case.
- **One level of grouping in the shared statement**, a term with parts,
  fed from numbered keys or from one table per group. The generic tree
  admits shapes no template renders, the validation table grows to match,
  and the result is still not the standard's shape. Dataverse and CKAN
  chose this compromise for forms; the library's callers construct records
  in code, where a typed field is clearer.
- **A second table for the record's copies** (`items.csv`, ADR-0015's
  twelfth decision and the eark-mods plan's S4). A one-to-many child
  table is the flat-to-rich device the CSV should not grow, and every
  later nested element would ask for the same. Copies travel in the
  library's record, as the ingest system supplies them, or in a supplied
  document. Withdrawn in the review of 2026-09-30, before it shipped.
- **A full schema binding**, structs generated from the XSD or
  `encoding/xml` struct trees. They permit everything the schema permits,
  which is what the closed vocabulary exists to prevent, and Go's
  `encoding/xml` cannot emit prefixed elements under a controlled prefix.
  [ADR-0002](0002-xml-via-text-template.md) stands: templates render, and
  the data they walk is the profile's typed model.
- **The document route as a second input path** (ADR-0016's
  `DescriptiveDocument` field at every level). Its cost was the branches
  at every level, recorded in ADR-0017. A second type behind the existing
  interface costs one optional interface with a method per eark encoder,
  one check and one copy branch in the engine, and one shared root reader.
- **Documents for meemoo.** The swap needs the entity UUID, which exists
  only at build.
- **A key grammar or delimiters inside a value.** Rejected in ADR-0015 and
  still: alignment and delimiter pitfalls every such convention documents.
- **XSD validation in process.** ADR-0003, ADR-0016.

## Consequences

- Library callers of `eark-mods` construct a `Record` by field, with a
  key typo a compile error instead of a validation finding. Callers of the
  two flat profiles are unchanged. `build.Definition.NewDescription` is
  gone and `input.New` takes the profile's adapter; the commits record
  each break.
- The output of all three profiles is unchanged, checked with
  scripts/reference-diff.sh against the reference copies; the eark-mods
  reference is captured in the plan.
- A MODS field costs a field on the record, a template line, a key in the
  adapter's table and a line in the input specification. A record richer
  than the fields is supplied as `mods.xml`; the CLI does not grow a
  grammar for it, nor a second table.
- A supplied document that is well-formed with the right root but invalid
  against its schema is packaged, and xmllint in build.sh and the
  repository's own validation catch it, as for received PREMIS. The
  package ships the standard's schema whatever the document's own
  `schemaLocation` says.
- The input specification's §3 vocabulary tables become CLI tables, and
  its deferred item on operator-supplied descriptive XML is enacted for the
  eark profiles. The rule in `CLAUDE.md` that `cli/input` imports no
  profile package narrows to the generic reader.
- ADR-0011 narrows to the CSV. ADR-0015's worlds, interface and the items
  on the record stand; its rule that one key emits one complete element is
  revised for MODS, and its `items.csv` is withdrawn. ADR-0016's document
  route returns in another shape and its `items.csv` rules go; ADR-0017 is
  superseded. ADR-0018's note placing `NewDescription` on `Definition` is
  undone.
