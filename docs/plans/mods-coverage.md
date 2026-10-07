# Plan: MODS 3.7 coverage for the eark/mods profile, in tiers

*Note, 2026-10-07: `eark/mods` is `ugent/bibliographic` and `earkmods.Record` is
`mods.Record` in `profiles/mods` since
[ADR-0034](../decisions/0034-a-profile-is-a-content-type.md); the body keeps the old
names.*

*Status: **parked** (drafted 2026-09-30). Its precondition is met: the
[descriptive-model plan](../archive/descriptive-model.md) shipped the
supplied document route on 2026-10-01, so coverage can grow tier by tier
while a record the model cannot yet say travels as a `mods.xml`.
[ADR-0022](../decisions/0022-reference-implementation-bring-your-own-profile.md)
records why: the eark/mods profile is plain E-ARK with a MODS 3.7 writer,
and as a reference implementation the writer speaks MODS. Update this
line as steps land.*

*Note, 2026-10-06: [ADR-0033](../decisions/0033-ugent-first-profiles-of-your-own.md)
drops the reference-implementation role, and with it this plan's premise
that MODS's own words are needed because other institutions would
otherwise translate UGent's twice. Before the plan resumes, decide again
whether the model speaks MODS (`TitleInfo`, `ShelfLocator`) or UGent's
catalogue words (call number, barcode). The typed constants for values a
caller chooses stand either way.*

*Note, 2026-10-07: [ADR-0034](../decisions/0034-a-profile-is-a-content-type.md) answers
the question: the model keeps UGent's catalogue words. It is UGent's application profile
of MODS, written by `ugent/bibliographic`, which replaces `eark/mods`. Before the plan
resumes, its tiers are read again against that.*

## Context

`mods.Record` is typed by field since ADR-0021, but its three fields
are UGent's application profile of MODS: a catalogue identifier whose
`type` the profile decides, titles, and holdings named in a librarian's
words (call number, barcode, enumeration). The template maps them onto
`mods:identifier`, `mods:titleInfo` and `mods:location/holdingSimple`.
For other institutions that is one translation too many, and it decides
attribute values a caller should choose.

MODS 3.7 has twenty top-level elements. The schema is permissive about
order (every top-level element is a repeatable choice) and enumerates
several attribute values (`typeOfResource`, name `type`, date `encoding`,
`point` and `qualifier`, `issuance`), which gives typed constant sets a
principled source. Two elements break the pattern: `relatedItem` holds a
whole MODS record, and `extension` holds arbitrary XML.

## Design decisions (agreed 2026-09-30)

1. **The model speaks MODS.** One type per top-level element, named after
   the element, with the subelements and attributes implementers actually
   use, at the level of the DLF Aquifer MODS guidelines rather than every
   leaf of the schema. Field names are the schema's words (`TitleInfo`,
   `NamePart`, `ShelfLocator`, `EnumerationAndChronology`); the field docs
   say what each is in plain words.
2. **Enumerated attributes are typed constant sets** drawn from the
   schema's enumerations or, where MODS only suggests a list (identifier
   `type`), from that list, with a zero value that means the common case.
   The caller chooses; the set is closed (ADR-0011, ADR-0022).
3. **The common attributes** `authority`, `authorityURI`, `valueURI`,
   `xml:lang` and `displayLabel` are in, on the elements that take them:
   they are what a record with controlled vocabularies needs.
   `altRepGroup`, `ID`, `script`, `transliteration` and `usage` wait for
   a caller who needs them.
4. **`relatedItem` is recursive**: a type holding a `type` attribute and a
   `*Record`, rendered by the record's own named template calling itself.
5. **`extension` stays out.** Arbitrary XML is what the supplied `mods.xml`
   route is for (ADR-0021).
6. **Validation splits as today.** Each element type validates what the
   schema and the profile say about it alone (a constant in its set,
   text that is not blank, a `titleInfo` with at least one child); the
   record validates the cross rules (the identity, no barcode twice). The
   encoders trust the result.
7. **The template stays one sub-template per element**, rendered in a
   fixed order; MODS assigns no meaning to order, and a fixed order keeps
   the output stable run to run. The document for today's test record
   stays byte for byte, pinned by the golden test, while its fields move
   to MODS words.
8. **The CSV vocabulary is a flat subset** and grows one key at a time
   (`creator`, `issued`, ...), each key deciding which element and which
   constants it fills; nothing obliges the rows to say what the model can.

## Tiers

Each tier ends with `go test ./...` green, the golden test passing, both
DC profiles VALID and identical to their reference copies, the eark/mods
fixture VALID with the xmllint pass clean (once the descriptive-model plan
has captured it), and the README's library example and the design doc's
descriptive bullet current. One commit per box, proposed in chat first.

### Tier 1: the identifier's type, and the text elements

- [ ] **The identifier's type as the caller's choice.** `Identifier{Value,
      Type IdentifierType}` with constants from the list MODS suggests for
      `identifier/@type` (`local`, `isbn`, `issn`, `doi`, `uri`, `lccn`,
      `hdl`, the exact set settled when the box starts); the zero value
      means `local`, so a caller who states nothing gets today's document;
      `mmsIDType` goes. Whether a record carries one identifier or several
      (an ISBN next to the local one), which turns the field into
      `Identifiers []Identifier` and the identity rule into "exactly one
      of type local", is decided in chat when the box starts.
- [ ] **Text elements with a few attributes.** `typeOfResource` (its
      enumeration), `genre` (authority), `abstract` (`xml:lang`),
      `tableOfContents`, `targetAudience` (authority), `note` (`type`),
      `classification` (authority, edition), `accessCondition` (`type`),
      `language` (`languageTerm` with `type` and authority).
- [ ] **Titles in MODS words.** `Title` becomes `TitleInfo` with `type`,
      `nonSort`, `title`, `subTitle`, `partNumber` and `partName`;
      `Titles []Title` becomes `TitleInfos []TitleInfo`.

### Tier 2: the structured elements

- [ ] **`name`**: `type` (personal, corporate, conference, family),
      `namePart` by `type`, `role` with `roleTerm` (`type` code or text,
      `authority`, `marcrelator` as the default), `affiliation`,
      `nameIdentifier`, `displayForm`.
- [ ] **`originInfo`**: `place` with `placeTerm`, `publisher`, the date
      family (`dateIssued`, `dateCreated`, `dateCaptured`, `dateModified`,
      `copyrightDate`, `dateOther`) with `encoding`, `point`, `qualifier`
      and `keyDate`, `edition`, `issuance`, `frequency`.
- [ ] **`subject`**: `topic`, `geographic`, `temporal`, `name`,
      `titleInfo`, `genre`, `occupation`, with `authority` on the subject.
- [ ] **`location`** in full: `physicalLocation`, `shelfLocator`, `url`
      (with `usage` and `access`), and `holdingSimple/copyInformation`
      (`subLocation`, `shelfLocator`, `enumerationAndChronology`,
      `itemIdentifier` with `type`, `note`). `Items []Item` becomes the
      `Location` field's copies; the ingest system's call follows.
- [ ] **`physicalDescription`**: `form` (authority), `extent`,
      `digitalOrigin` (enumeration), `internetMediaType`, `note`.
- [ ] **`part`**: `type`, `order`, `detail` (`type`, `number`, `caption`),
      `extent` (`unit`, `start`, `end`), `date`, `text`.
- [ ] **`recordInfo`**: `recordContentSource`, `recordCreationDate`,
      `recordChangeDate`, `recordIdentifier` (`source`), `recordOrigin`,
      `languageOfCataloging`, `descriptionStandard`.

### Tier 3: the recursive element and the closing sweep

- [ ] **`relatedItem`**: `type` (its enumeration) and a nested `*Record`,
      rendered recursively.
- [ ] **The common attributes sweep**: `authority`, `authorityURI`,
      `valueURI`, `xml:lang` and `displayLabel` on every element that
      takes them, where a tier left one out.
- [ ] **Closing docs**: the design doc's descriptive bullet (the writer
      covers MODS 3.7's top-level elements; `extension` and the deferred
      attributes travel as a supplied document), README, `CLAUDE.md`,
      ADR-0022 to Accepted if S5 of the descriptive-model plan has not
      already done so, this plan archived.

## Size, for the record

Counted on 2026-09-30 against the current package, where three elements
cost about 300 lines of code and 260 of tests, half of the code being the
field docs the project requires.

| part | lines, rough |
|---|---|
| model: element types, subelements, field docs | 550 to 650 |
| typed constant sets | 100 |
| validation | 150 to 200 |
| template | 250 |
| tests | 600 to 800 |
| total | about 1,700 to 2,000, half of it tests |

Tier 1 plus `titleInfo`, `name`, `originInfo`, `subject` and `location`
is perhaps 700 lines including tests and covers most bibliographic
records; the rest follows element by element behind the same golden test.

## Open questions

- **One identifier or several.** Decided when tier 1 starts; it changes
  the identity rule (`ValidateRequired`) and the eark/mods vocabulary's
  `identifier` key.
- **The constant sets' exact members**, per attribute: which identifier
  types, which relator authority, whether `dateOther` needs its own
  `type`. Settled per box against the schema and the MODS user guidelines.
- **The CSV keys for new elements**: which flat keys the eark/mods
  vocabulary gains (`creator` as a personal name with role `aut`?
  `issued` as `dateIssued` with `encoding="edtf"`?), each a decision of
  the vocabulary, made when the repository's index fields are known.
- **The ingest system's migration** from `Items` and `CallNumber` to the
  MODS-named fields: the commit that renames them records the break.
