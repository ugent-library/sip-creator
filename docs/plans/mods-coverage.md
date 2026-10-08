# Plan: the ugent/bibliographic MODS record follows the catalogue

*Status: **active**. Drafted 2026-09-30 as full MODS 3.7 coverage for `eark/mods`;
rewritten 2026-10-08 after [ADR-0033](../decisions/0033-ugent-first-profiles-of-your-own.md)
and [ADR-0034](../decisions/0034-a-profile-is-a-content-type.md) made the record UGent's
application profile of MODS. Update this line as steps land.*

## Context

`ugent.Record` holds three things: the catalogue identifier, titles, and the library's
physical copies (call number, barcode, enumeration). The template writes them as
`mods:identifier type="local"`, `mods:titleInfo/mods:title` and
`mods:location/mods:holdingSimple/mods:copyInformation`.

The description of a `ugent/bibliographic` package comes from the UGent Library
catalogue, which delivers its MARC records in a flattened form: one list of strings per
field, with the subfields joined. That form decides what the MODS record can say. The
mapping we settled on, field by field, is in
[docs/profiles/ugent-bibliographic-mods.md](../profiles/ugent-bibliographic-mods.md).
This plan builds its planned rows.

The 2026-09-30 draft planned the whole of MODS 3.7 in MODS's own element names, for
institutions other than UGent. ADR-0033 dropped that audience, and the catalogue cannot
fill most of those elements, so that scope is gone.

## Design decisions (agreed 2026-10-08)

1. **The record holds what the catalogue delivers, in the catalogue's words**
   (ADR-0034). A field enters the record when the catalogue has a value for it and a
   decision brings it into scope. The fields the catalogue delivers that are out of scope
   are listed in the mapping's §2.
2. **What the catalogue joins stays joined.** A name is one string, written as one
   `mods:namePart` without `type` or `mods:role`. A title holds its statement of
   responsibility. Typed name parts, roles and the 245 $a / $c split wait until the
   catalogue delivers subfields.
3. **The MMS ID stays the one required identifier**, written as `type="local"`. The other
   identifiers are a list of strings, written without `type`, because the catalogue does
   not say which kind each is.
4. **A copy states a call number, a barcode, or both.** The call number is no longer
   required: when the catalogue has none, there is none. A copy that states neither is
   refused, because it would be written as an empty `mods:copyInformation`.
5. **Nothing outside the mapping enters the model.** A record that needs more, such as
   typed names or `mods:extension`, travels as a supplied `mods.xml` (ADR-0021).
6. **Validation splits as today.** Each field validates what it alone must hold (text
   that is not blank and that XML can carry); `Record.Validate` holds the rules across
   fields (no barcode twice); `ValidateRequired` holds the identity rule. The encoders
   trust the result.
7. **The template keeps one fixed order of elements**, so the output is the same run to
   run, and today's test record stays byte for byte, pinned by the golden test.

## Steps

Each step ends with `go test ./...` green and the golden test passing, `./build.sh`
reporting VALID for `meemoo/basic`, `ugent/basic` and `ugent/bibliographic`,
`./scripts/reference-diff.sh` clean or the reference copy updated with the reason in the
commit, and the profile page §3, the mapping page and the design doc current. One commit
per step, proposed in chat first.

- [x] **Copies without a call number.** `validateItem` accepts an item without a call
      number and refuses one that states neither a call number nor a barcode; the
      template leaves out `mods:shelfLocator` when the call number is empty. The profile
      page §3 changes "A copy MUST state its call number" to the rule in decision 4.
- [ ] **Other identifiers.** A list of identifiers next to the MMS ID, each written as a
      `mods:identifier` without `type`. The 035 $a numbers arrive through the same list
      once the catalogue delivers them; nothing in the library changes for them.
- [ ] **Names.** A list of names, each written as `mods:name/mods:namePart`. MODS sets no order
      for its top-level elements; the template writes names after the titles.
- [ ] **Closing.** The design doc's description of `ugent/bibliographic`, the README's
      library example, the mapping page's status column, and this plan archived. An ADR
      only if a decision here is not already covered by ADR-0034.

## Open questions

- **Field names.** What the new `Record` fields are called (for instance `Identifiers`
  next to `Identifier`, and `Names` or `Contributors`) is decided in chat when each step
  starts, as `CLAUDE.md` asks for names.
- **`description.csv` keys.** Whether the CSV vocabulary gains keys for the other
  identifiers and the names, or whether they stay for programs and supplied `mods.xml`
  documents, as copies do today.
- **The 035 prefix.** A 035 $a value starts with its source in parentheses, such as
  `(RUG01)`. Whether that prefix becomes the identifier's `type`, and is stripped from
  the value, is decided when the catalogue delivers 035 and the values can be seen.
- **The fields in the mapping's §2.** Each one comes into scope by its own decision.
