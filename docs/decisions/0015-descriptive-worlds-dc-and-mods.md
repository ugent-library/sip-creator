# 0015 — Descriptive worlds, each with its own terms type: meemoo dc+schema, Simple DC, MODS

Status: **Proposed** (drafted 2026-09-22; the decisions were agreed in
review on 2026-09-15 with the [eark-mods plan](../plans/eark-mods.md).
Becomes Accepted when that plan's S3 ships.) **Superseded in part by
[ADR-0018](0018-engine-and-profile-packages.md)** (2026-09-28): the worlds
live in profile packages under `profiles/` (`profiles/meemoo`,
`profiles/eark`) rather than under `encoders/`, and the descriptive
standard is an exported interface on the engine that each profile package
implements, closed by the registry rather than by an unexported field.
The worlds themselves, and everything below about them, stand.

## Context

The tool emits one descriptive document per profile: meemoo's
`dc+schema` document for `basic`, a simple Dublin Core document for `eark`.
Both come from one model in `encoders/metadata`: a flat list of terms, each
naming the one Dublin Core element it emits (`dcterms:title`), read from one
closed vocabulary table ([ADR-0011](0011-closed-descriptive-vocabulary.md)).
That table is meemoo's vocabulary, and the simple DC document is derived
from it by the DCMI dumb-down, so an eark operator's `artmedium`,
`artform` or `rightsholder` row vanishes from the output without a word.
That model also reaches into the rest of the code. `sip.Entity.Description`,
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
MODS profile would bring its own table; the open question was how much of the
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
type, a closed vocabulary table, the validation of those terms, the
templates that render them, and the list of bundled XSDs its document
points at, which a profile concatenates into what its packages ship. There are three, none importing another:
`encoders/dcschema` for meemoo's `dc+schema` document (Dublin Core terms
plus schema.org properties, EDTF typing, meemoo's cardinality and language
rules), `encoders/dc` for Simple Dublin Core (the fifteen elements, no
rules beyond validity), and `encoders/mods`. Meemoo's document and Simple
DC share only the Go shape of a term, so the dumb-down that derived one
from the other goes: an eark operator's unknown key is refused, not
dropped. (Revised 2026-09-23; the 2026-09-15 review had one DC world
carrying both templates.)

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

**Each profile names its descriptive standard directly; profiles stay what
operators type.** A profile's registry entry carries, next to its values,
the one behavioral choice it makes: an unexported `descriptive` value from
the closed set in `profiles/descriptive.go` (meemoo's `dc+schema` document,
simple DC, and MODS once it exists), each a check that the input's
descriptions have the standard's type plus the encoder that writes the
document. The profiles are `basic` (dc+schema), `eark` (simple DC, name
kept) and the new `eark-mods` (MODS), whose registry entry copies `eark`
with `MDTYPE="MODS"`, `MDTYPEVERSION="3.7"`, `mods.xml` as the document
name, and identifier and title required. The `Family` constant of
[ADR-0007](0007-profile-families-share-one-writer.md) is retired (decided
2026-09-23, revising the 2026-09-15 review): with one profile per standard
it named the same thing twice, and nothing consumed the data-only
`Definition` it protected. What ADR-0007 anticipated as an internal struct
of choices exists, but the profile holds it directly instead of resolving
it from a constant. There is still one writer, and what meemoo alone needs
(the submitter's OR-id) is a plain data flag on the profile.

**The domain model speaks to descriptive metadata through a small
interface.** `sip/` declares `Description` with `LocalIdentifier() string`,
`Validate() error` and `ValidateRequired(keys ...string) error`, the
last so a profile's required keys are checked without the profile
knowing the terms type (added 2026-09-23 with the split of the DC
worlds). All three terms types implement it; `Entity.Description`,
`Representation.Description` and `Input.Descriptive` take the interface, and
`sip/` imports no encoder package. At the start of a build, before validation
and before any disk write, the profile's descriptive standard asserts that
the input's concrete type is its own; a mismatch is a build error (the rule itself is recorded
in [ADR-0016](0016-descriptive-input-rows-or-supplied-document.md)). The
identifier swap the meemoo document needs stays a method on the meemoo
terms type, because only that world has it. Required elements become
plain vocabulary keys every table resolves: the identity every package
states (`identifier`, `title`) is checked once in `Input.Validate`, and a
`Definition` lists only what its own spec adds (`description` and
`created` for meemoo's basic profile; nothing for plain E-ARK). Decided
2026-09-24: the meemoo table had carried a required column that one
profile read while the other spelled its keys in a literal, and the CLI
spelled the identity rule a third time in element names. (Revised
2026-09-28, on review of S2's result: the interface is `Validate` and
`ValidateRequired()`. What a package-level description must state is
each world's own list, identifier and title plus what its spec adds
(description and created for meemoo's basic profile), checked in
`Input.Validate` and by `check`, so a `Definition` names no descriptive
elements and no world resolves CSV keys for the library; the identifier
swap, with the reading of the local identifier it needs, moved behind
the meemoo standard's value in
`profiles/descriptive.go`; and the encoders no longer validate,
`Input.Validate` being the one contract before a write.)

## Alternatives rejected

- **One Dublin Core world carrying both the meemoo and the Simple DC
  template** (the first draft of this ADR). The two documents differ in
  table, rules and template and share only the Go shape of a term, and
  the dumb-down between them is a lossy mapping of the kind this ADR
  rejects for MODS. Keeping them together also made the eark profile's
  input vocabulary meemoo's, which no eark consumer asked for.
- **One neutral term type, keyed by the plain vocabulary key, shared by all
  worlds.** It would have put the CSV and the library on one key language.
  But a DC term produces one element and a MODS term produces a subtree; one
  type for both hides that difference, and every method on it (validate,
  resolve, render) would need the profile passed in to find the right table.
- **A key grammar: qualifiers or paths parsed out of the key string.**
  Qualifiers (`name[role=aut]`, `title[type=alternative]`) checked against a
  closed list are a closed vocabulary spelled differently. They emit exactly
  what a row emits, move the meaning from an operator word (`promotor`) into
  MODS internals, and give the bracket a second meaning next to the
  `title[nl]` language suffix. Paths (`name.namePart`, `name.role.roleTerm`)
  would be needed for real structure, and a flat row list cannot group them:
  two name parts and two roles give no way to say which role belongs to
  which name. Both cost a parser and a grammar section in the input
  specification for nothing a row cannot say. Records with structure were
  to travel as supplied documents instead (ADR-0016); that route is
  deferred ([ADR-0017](0017-supplied-descriptive-document-deferred.md)),
  and the flat rows serve the flat columns the first consumers hold.
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
- **A family constant between the profile and its behavior** (ADR-0007's
  `Family`, resolved at build time to a struct of choices). It kept
  `Definition` a struct of plain values, but nothing serializes or compares
  a definition, and embedding systems take definitions from the registry
  rather than building them. With `basic`, `eark` and `eark-mods` each in a
  standard of their own, the constant mapped one to one onto the profile,
  and a reader of a registry entry had to open a second file to learn what
  the profile writes. The behavior field on the profile is unexported, so
  the set of standards stays closed and no caller can hand `Definition` an
  encoder of its own. (Superseded 2026-09-28 by ADR-0018: the field is the
  exported `build.DescriptionEncoder`, implemented by each profile
  package, and the registry closes the set.)

## Consequences

- Outside the items table, the flat MODS model stops at one value per
  element. It cannot express several parts under one parent: a given and a
  family name part on one name, a name with two roles, a subject heading
  combining a topic and a place, an affiliation next to a name. That matches the source data (flat
  columns). Anything richer has no route into the package until the table
  grows a row for its shape or the supplied-document route of ADR-0016
  returns; that route is deferred
  ([ADR-0017](0017-supplied-descriptive-document-deferred.md), 2026-09-28).
- Adding a MODS element is a table row, a template fragment when the shape
  is new, and an input-specification line: the same rule as ADR-0011. The
  attribute values still open (the identifier `type` for the MMS ID, the
  relator code per role) are decided by the owner of the repository side and
  live in the table only.
- Library callers change their import from `encoders/metadata` to
  `encoders/dcschema` (meemoo profiles) or `encoders/dc` (plain E-ARK);
  the commits that split it record the break. A caller who hands terms of
  one world to a profile of another gets an error before any write.
- The eark input vocabulary is Simple DC's fifteen elements, no longer
  meemoo's keys: `abstract` becomes `description`, `license` becomes
  `rights`, and `artmedium` is an unknown key rather than a silently
  dropped one.
- `sip/` depends on no encoder. Adding a descriptive standard no longer
  touches the domain model; `profiles/descriptive.go` is the one place that
  lists the standards a profile can pick from.
- The output of `basic` and `eark` does not change. The rename, the
  interface and the descriptive standard on the profile are a refactor,
  checked with scripts/reference-diff.sh against the reference copies.
- The MODS description type is a struct holding terms and items, not a
  slice of terms. It still implements the `Description` interface, so the
  difference stays inside the MODS package; the CLI transports the items as
  `items.csv` (ADR-0016). A new column on the items table costs what a new
  key costs: a table entry, a template line, an input-specification line.
- Three profiles today. Another profile writing an existing standard is one
  registry entry naming an existing `descriptive` value; another standard is
  a new encoder package and one value in `profiles/descriptive.go`.
  ADR-0007 is superseded in the one respect of the `Family` constant; its
  one-writer rule and fork triggers stand.
