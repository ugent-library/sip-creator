# 0015 — Two descriptive worlds, DC and MODS, each with its own terms type

Status: **Proposed** (drafted 2026-09-22; the decisions were agreed in
review on 2026-09-15 with the [eark-mods plan](../plans/eark-mods.md).
Becomes Accepted when that plan's S3 ships.)

## Context

The tool emits one descriptive document per profile family: meemoo's
`dc+schema` document for `basic`, a simple Dublin Core document for `eark`.
Both come from one model in `encoders/metadata`: a flat list of terms, each
naming the one Dublin Core element it emits (`dcterms:title`), read from one
closed vocabulary table ([ADR-0011](0011-closed-descriptive-vocabulary.md)).
That model reaches into the rest of the code. `sip.Entity.Description`,
`sip.Representation.Description` and `profiles.Input.Descriptive` are all
typed `metadata.Terms`, so the domain model imports an encoder package, and
the identity rules (`LocalIdentifier`, the identifier swap, the required
elements on a `Definition`) name `dcterms:identifier` literally.

Bibliographic material at UGent Library needs the plain E-ARK output to carry
[MODS 3.7](https://www.loc.gov/standards/mods/) instead, so the repository
that ingests it can index title, creator and dates from a bibliographic
record. MODS does not fit the flat model: a title is `titleInfo/title`, a
name carries parts and a role, a date may have a start and an end point. A
term that names one element cannot say that. ADR-0011 already noted that a
MODS family would bring its own table; the open question was how much of the
existing model the two standards share.

The first consumers are systems that automate ingest workflows. They hold
descriptive metadata as flat columns (an identifier, a title, ...) and
construct `profiles.Input` directly, so the library API comes first and the
CLI follows.

One package describes one bibliographic record, and a record has physical
copies: each a call number, usually a barcode, and for journals,
periodicals and newspapers a volume or issue designation. Several copies
per record are normal. A flat list of statements cannot say which barcode
belongs to which call number.

## Decision

**Each descriptive standard is its own world, and the worlds share no term
type.** A world is everything the tool knows about one standard: a terms
type, a closed vocabulary table, the validation of those terms, and the
templates that render them. `encoders/metadata` is renamed to `encoders/dc`
and keeps both existing templates (meemoo `dc+schema`, simple DC).
`encoders/mods` is the second world. Neither imports the other.

**In the MODS world, one key emits one complete element with fixed
attributes.** The MODS table maps a plain key to a template fragment plus the
attribute values that fragment needs: `identifier` becomes `mods:identifier`
with a fixed `type`, `title` becomes `mods:titleInfo/mods:title`. Operators
and library callers set no attributes, as today (ADR-0011). The meaning
lives in the key, so roles, title types and identifier types are rows, not
key syntax: `author` and `promotor` share one `name` fragment and differ in
the relator code, `alternative` emits `titleInfo type="alternative"`, `isbn`
and `doi` emit `identifier` with their own `type`. Names come in personal
and corporate rows, because the library's records name faculties, research
departments and competence centres as agents next to people. Dates are
single EDTF values with a fixed `encoding="edtf"`, so a range is a value,
not a shape. The table starts with `identifier` and `title`, the two columns
known to be present in every record. Each further row is data: a table
entry, a template fragment when the shape is new, and a line in the input
specification.

**A record's physical copies are a second flat table, one row per copy.**
The MODS description carries, next to its terms, a list of items. Each item
is a call number (required), a barcode (optional, unique across the items)
and a volume or issue designation (optional). The template renders the list
as one `location/holdingSimple` with one complete `copyInformation` per
item: the call number as `shelfLocator`, the designation as
`enumerationAndChronology`, the barcode as `itemIdentifier type="barcode"`.
Items are described at package level only. A representation stays what
CSIP says it is, a version of the same content such as a master next to an
access copy; a copy or a volume is never a representation, and no item
information goes into a representation's descriptive document.

**Families name a world plus an encoding; profiles stay what operators
type.** The families are `meemoo` (DC world, `dc+schema` document), `eark-dc`
(DC world, simple DC document) and `eark-mods` (MODS world). The profiles are
`basic` (meemoo), `eark` (eark-dc, name kept) and the new `eark-mods`, whose
registry entry copies `eark` with `MDTYPE="MODS"`, `MDTYPEVERSION="3.7"`,
`mods.xml` as the document name, and identifier and title required. This is
the promotion trigger recorded in
[ADR-0007](0007-profile-families-share-one-writer.md): the `Family` constant
stays the data on `Definition` and resolves at build time to an internal
struct of the family's choices (which world's terms type, which encoder,
whether a supplied document is accepted). There is still one writer.

**The domain model speaks to descriptive metadata through a small
interface.** `sip/` declares `Description` with `LocalIdentifier() string`
and `Validate() error`. Both terms types implement it; `Entity.Description`,
`Representation.Description` and `Input.Descriptive` take the interface, and
`sip/` imports no encoder package. At the start of a build, before validation
and before any disk write, the family asserts that the input's concrete type
is its own world's; a mismatch is a build error (the rule itself is recorded
in [ADR-0016](0016-descriptive-input-rows-or-supplied-document.md)). The
identifier swap the meemoo document needs stays a method on the DC terms
type, because only that world has it. Required elements on a `Definition`
become plain vocabulary keys (`identifier`, `title`), which both tables
resolve.

## Alternatives rejected

- **One neutral term type, keyed by the plain vocabulary key, shared by both
  worlds.** It would have put the CSV and the library on one key language.
  But a DC term produces one element and a MODS term produces a subtree; one
  type for both hides that difference, and every method on it (validate,
  resolve, render) would need the family passed in to find the right table.
- **A key grammar: qualifiers or paths parsed out of the key string.**
  Qualifiers (`name[role=aut]`, `title[type=alternative]`) checked against a
  closed list are a closed vocabulary spelled differently. They emit exactly
  what a row emits, move the meaning from an operator word (`promotor`) into
  MODS internals, and give the bracket a second meaning next to the
  `title[nl]` language suffix. Paths (`name.namePart`, `name.role.roleTerm`)
  would be needed for real structure, and a flat row list cannot group them:
  two name parts and two roles give no way to say which role belongs to
  which name. Both cost a parser and a grammar section in the input
  specification for nothing a row cannot say. Records with structure travel
  as supplied documents instead (ADR-0016).
- **Pairing a call number with its barcode inside `mods.csv`**, by row
  order, by indexed keys or by a delimiter inside one value. Order
  misaligns silently the moment one copy has no barcode. Indices are the
  key grammar above. A delimiter is an invented syntax that stops working
  at the third field. A row with columns says the same thing without any
  of that, and `representations.csv`
  ([ADR-0014](0014-representations-csv-strict-when-present.md)) is the
  precedent for it.
- **One representation per physical copy or volume**, so that a call number
  and a barcode pair by folder. CSIP defines a representation as a version
  of the data and metadata over its lifecycle, and E-ARK SIP speaks of
  representations of the same intellectual entity. Two volumes are parts
  and two copies are duplicates, not versions; the structural metadata
  would say "rendition" where the truth is "copy". Neither specification
  mentions bibliographic items, call numbers or barcodes: pairing them is
  a MODS concern, and MODS has `copyInformation` for it.
- **One `eark` family with the descriptive encoder as a `Definition`
  field.** The encoder is behavior, and ADR-0007 keeps behavior on the
  family and values on the profile. A profile field naming a function also
  stops `Definition` from being plain, comparable data.

## Consequences

- Outside the items table, the flat MODS model stops at one value per
  element. It cannot express several parts under one parent: a given and a
  family name part on one name, a name with two roles, a subject heading
  combining a topic and a place, an affiliation next to a name. That matches the source data (flat
  columns); anything richer arrives as a finished `mods.xml` (ADR-0016).
- Adding a MODS element is a table row, a template fragment when the shape
  is new, and an input-specification line: the same rule as ADR-0011. The
  attribute values still open (the identifier `type` for the MMS ID, the
  relator code per role) are decided by the owner of the repository side and
  live in the table only.
- Library callers change their import from `encoders/metadata` to
  `encoders/dc`; the commit that renames it records the break. A caller who
  hands terms of one world to a profile of the other gets an error before
  any write.
- `sip/` depends on no encoder. Adding a descriptive standard no longer
  touches the domain model; the family resolution is the one place that
  lists what each family does.
- The output of `basic` and `eark` does not change. The rename, the
  interface and the family struct are a refactor, checked with
  scripts/reference-diff.sh against the reference copies.
- The MODS description type is a struct holding terms and items, not a
  slice of terms. It still implements the `Description` interface, so the
  difference stays inside the MODS package; the CLI transports the items as
  `items.csv` (ADR-0016). A new column on the items table costs what a new
  key costs: a table entry, a template line, an input-specification line.
- Three families and three profiles today. Another profile in an existing
  family is still one registry entry; another world is a new encoder
  package, a family constant and one resolution case.
