# 0029 — The eark profiles offer the E-ARK SIP as CSIP defines it: the description and the representation minimum are profile rules

Status: **Accepted** (2026-10-05). Supersedes
[ADR-0025](0025-every-package-carries-a-description.md). Extends
[ADR-0026](0026-profile-rules-on-the-definition.md): a minimum number of
representations joins the maximum. **Note, 2026-10-06:** implementation waits for a
UGent case. Neither the package without descriptive metadata (`eark/none`) nor the
metadata-only package is needed within UGent now
([ADR-0033](0033-ugent-first-profiles-of-your-own.md)); the decision stands, and steps 2
to 5 and 8 of the [plan](../plans/profile-rules-and-names.md) are parked until a case
arrives. Until then the engine keeps its two rules: every package carries a package-level
description and at least one representation. **Note, 2026-10-07:** that plan is retired
([archive](../archive/profile-rules-and-names.md)). The representation minimum lands as
step 7 of the [ugent-profiles plan](../plans/ugent-profiles.md); `eark/none` is dropped,
because `ugent/basic` requires a description; a profile without descriptive metadata is
the empty case of the model list in the parked
[entities-and-descriptions plan](../plans/entities-and-descriptions.md). The table's names
change with ADR-0034.

## Context

Two rules held under every profile, in `SourcePackage.Validate`: a package carries a
package-level description, and a package has at least one representation. ADR-0025
recorded the first as the tool's own policy, stricter than CSIP, "for a case no user of
the tool has asked for", and noted that an institution whose profile allows a package
without a description would need a change in `build`. The second was never recorded as
a decision.

CSIP 2.2.0 makes both conditional:

- CSIP17, `mets/dmdSec`, 0..n SHOULD: "Must be used if descriptive metadata for the
  package content is available."
- CSIPSTR7: "If descriptive metadata are available, they SHOULD be included in
  sub-folder `descriptive`."
- CSIP88, the structMap division `LABEL="Metadata"`, 1..1 MUST. CSIP92, its `@DMDID`,
  0..1 SHOULD: "All dmdSecs with `@STATUS='CURRENT'` SHOULD be referenced by their
  identifier."
- CSIP58, `mets/fileSec`, 0..1 SHOULD: "In the case that a package only contains
  metadata updates, i.e. exclusively metadata files, then no file references need to be
  added to this section."
- CSIP105, one structMap division per representation, 0..n SHOULD. CSIPSTR9: the
  package folder SHOULD include a folder named `representations`.

E-ARK SIP 2.2.0 §3.3 "does not change or extend any of the requirements already defined
by the Common Specification" for `dmdSec`, and its introduction says: "A package with
zero representations means that it only contains metadata. This is a special type of
Information Package that enables Producers to deliver updates to the metadata to
previously ingested packages."

Meemoo SIP 1.2's basic profile requires both: "exactly one metadata file
`dc+schema.xml`" and "The IE MUST be represented by exactly one representation."

The eark profiles exist to produce the E-ARK SIP without Meemoo's layer. The user asked
for the E-ARK SIP as the specification defines it: a package that carries essence and no
descriptive metadata, because the description lives in a system of record keyed on the
package identifier; and a package that carries a description and no essence, the
metadata-only package the E-ARK SIP introduction describes, to update the metadata of a
package ingested earlier.

## Decision

Both rules become values on `build.Definition`, as ADR-0026 did with the profile's other
rules on the package's shape, and `Definition.ValidateSource` checks them.

**A definition without a metadata model carries no descriptive metadata.** `Model ==
nil` says so. `ValidateSource` refuses a description at the package level and at the
representation level. The assembler creates no descriptive file node and lists no
descriptive XSD, the writer writes no document, and the package METS has no `dmdSec`
and no `DMDID` on its Metadata division, which stays because CSIP88 requires it. With a
model, a package-level description is required as before, but the rule is now
`ValidateSource`'s; `SourcePackage.Validate` validates a description only when one is
present. Under such a profile the input reader reports a `description.csv` at any level
as a violation, and the profile has no mapper, because it takes no rows.

**`MinRepresentations` is the least number of representations a package needs; zero
allows a package without any.** `SourcePackage.Validate` no longer requires a
representation; a representation that exists must still have files. The input reader
reads a folder without content files, or with an empty `representations/` folder, as
zero representations and leaves the verdict to the profile, as ADR-0026 and ADR-0027
have it: the reader judges files and names, the definition judges the package's shape. A
metadata-only package has a fileSec with the Schemas group, and the Documentation group
when there is documentation, and a structMap without representation divisions. The tool
does not tie zero representations to an update status, because CSIP does not; the usual
case is an update, with `--status` and `--updates`.

| profile | descriptive metadata | representations |
|---|---|---|
| `meemoo/basic` | Meemoo `dc+schema.xml`, required | exactly one (minimum 1, maximum 1) |
| `eark/none` | none | at least one |
| `eark/dc` | Simple Dublin Core, required | any number, including none |
| `eark/mods` | MODS 3.7, required | any number, including none |

`eark/none` needs a representation because a package with no description and no content
is not a package. The names are [ADR-0030](0030-profile-names-by-family.md)'s.

The `representations/` and `metadata/descriptive/` directories are created in every
package, empty when unused, as `metadata/preservation/` already is. CSIPSTR7 and
CSIPSTR9 recommend the folders only when there is something to put in them, but
commons-ip 2.11.2, which `build.sh` validates with, warns when either is missing
(`StructValidator.validateCSIPSTR7` and `validateCSIPSTR9`), and the samples are held
to zero warnings.

## Alternatives rejected

- **A boolean next to the model saying the profile carries no description.** A model
  that writes a document next to a flag saying there is none contradicts itself. The
  absence of the model is the fact.
- **A boolean `AllowMetadataOnly` instead of a number.** The minimum reads with the
  maximum, and Meemoo's "exactly one" is 1 and 1.
- **Requiring an update status for a package with zero representations.** The E-ARK SIP
  introduction names updates as the purpose, but CSIP58 permits the package without a
  condition. A program that registers an intellectual entity before its content arrives
  can build a new metadata-only package.
- **Leaving out the empty `representations/` and `metadata/descriptive/` directories.**
  The specification allows it; the validator warns, and an empty directory costs
  nothing.
- **Keeping the rules in the engine** and letting institutions fork `build`, ADR-0025's
  consequence. The eark profiles are this tool's own, so the rules had to move for them
  anyway.

## Consequences

- `SourcePackage.Validate` alone no longer finds a missing description or a missing
  representation. `Definition.ValidateSource` does, so `check` reports both without
  building.
- A program using the library with its own profile
  ([ADR-0022](0022-reference-implementation-bring-your-own-profile.md)) states both
  rules on its definition. The zero values are permissive: no model means no
  description, no minimum means a package may be metadata only. A profile that needs a
  description or a representation says so.
- `meemoo/basic` accepts what it accepted: a description in its model, exactly one
  representation. Its package PREMIS always has a representation to relate to.
- A metadata-only update needs `--status` and `--updates` as every update does; the
  identifier of the earlier package becomes this package's `OBJID`.
- The reader's findings "the folder contains no content files" and "representations/
  contains no representation folders" are gone. A profile that needs a representation
  reports "needs at least 1 representation(s)".
- Whether commons-ip reports a metadata-only package and a package without a `dmdSec`
  as VALID is checked with `build.sh` when each ships, and recorded in
  [TODO.md](../TODO.md) under the validator status.
