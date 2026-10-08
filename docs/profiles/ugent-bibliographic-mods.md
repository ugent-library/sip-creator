# `ugent/bibliographic`: from catalogue record to MODS

*Mapping for the descriptive metadata of [`ugent/bibliographic`](ugent-bibliographic.md).
Settled 2026-10-08.*

The description of a `ugent/bibliographic` package comes from the UGent Library
catalogue. The tool does not read the catalogue itself. A program that builds packages
reads the catalogue's records and hands each one to the library as a `ugent.Record`,
which the library writes as `metadata/descriptive/mods.xml`.

The catalogue's records reach that program in a flattened form: each field is a list of
strings, with the MARC subfields of a field joined into one string. MARC's subfield
structure is gone by then, so the MODS record cannot split what the catalogue already
joined. A name is written as one `mods:namePart`, without the name's type or role, and a
title holds its statement of responsibility. When the catalogue delivers a field with
its parts, the mapping gains them.

## 1. Mapped fields

| what the record states | from MARC (Alma) | `ugent.Record` field | MODS 3.7 |
|---|---|---|---|
| the catalogue record's identifier, the MMS ID | 001 | `Identifier` | `mods:identifier type="local"` |
| other identifiers: ISBN, ISSN, other standard numbers, call number from the holdings | 020 $a, 022 $a, 024 $a, 852 $j | `OtherIdentifiers` | `mods:identifier`, without `type` |
| legacy and external system numbers: the Aleph `RUG01` number, Ufora, Plato, Antilope | 035 $a | `OtherIdentifiers` | `mods:identifier`, without `type` |
| title, with its statement of responsibility | 245, all subfields, joined with spaces | `Titles` | `mods:titleInfo/mods:title` |
| contributors: the main author and every other name, persons and organizations alike | 100, 110, 700, 710, 711, 720, all subfields of each, joined with spaces | `Contributors` | `mods:name/mods:namePart`, without `type` and without `mods:role` |
| a physical copy's call number | item field Z30 $3 | `Items[].CallNumber` | `mods:location/mods:holdingSimple/mods:copyInformation/mods:shelfLocator` |
| a physical copy's barcode | item field Z30 $4 | `Items[].Barcode` | `mods:copyInformation/mods:itemIdentifier type="barcode"` |
| a copy's volume or issue | none | `Items[].Enumeration` | `mods:copyInformation/mods:enumerationAndChronology` |

Notes on the rows:

- The MMS ID is the identifier that relates a package to its catalogue record, so it is
  the one identifier a package-level record must state, next to at least one title. The
  catalogue gives `unknown` for a record without a 001. That is not an MMS ID, and a
  program must not pass it on as one.
- The other identifiers carry no `type`, because the catalogue joins them into one list
  and does not say which MARC field each came from. MODS allows an identifier without a
  type. The catalogue does not deliver 035 yet, but will.
- "Contributor" is the catalogue's word for every name on the record, the main author
  included. It is not Dublin Core's narrower `contributor`, and not `ugent/basic`'s key of
  that name. MODS would tell an author from another contributor by `mods:role`, but the
  catalogue joins a name's relator term into the name, so each is written as one
  `mods:namePart`. The catalogue does not say whether a name is a person or an
  organization either, so `mods:name` carries no `type`.
- A copy states a call number, a barcode, or both. A copy that states neither is
  refused.
- The catalogue delivers no volume or issue for a copy. A program that knows it can
  still set it.
## 2. Fields left out for now

The catalogue delivers these fields too. They are out of scope until a decision brings
them in. The MODS element is the one each would most likely map to; it is not settled.

| catalogue field | from MARC | likely MODS element |
|---|---|---|
| record creation date | 008/00-05 | `mods:recordInfo/mods:recordCreationDate` |
| date of the work | 008/07-10 | `mods:originInfo/mods:dateIssued` |
| language | 008/35-37 | `mods:language/mods:languageTerm` |
| publisher | 260 $a $b | `mods:originInfo/mods:publisher` |
| physical medium | 340 | `mods:physicalDescription/mods:form` |
| notes | 500 to 599 | `mods:note` |
| rights | 506, 540 | `mods:accessCondition` |
| source | 534 $t | `mods:relatedItem type="original"` |
| subjects and classification | 050, 060, 080, 082, 600, 610, 611, 630, 650, 653 | `mods:subject`, `mods:classification` |
| UGent material type and subtype | 920 $a, 922 $a | `mods:genre` |
| links to the digital object, and their access level | AVD $u, derived from 599 $a | `mods:location/mods:url`, `mods:accessCondition` |
