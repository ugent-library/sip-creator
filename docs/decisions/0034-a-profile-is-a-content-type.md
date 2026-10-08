# 0034 — A profile is a content type its owner defines: ugent/basic and ugent/bibliographic replace eark/dc and eark/mods

Status: **Accepted** (2026-10-07). Supersedes
[ADR-0030](0030-profile-names-by-family.md). Supersedes in part
[ADR-0018](0018-engine-and-profile-packages.md): a package under `profiles/` belongs to
an owner and holds the definitions of that owner's profiles with the metadata models they
use. The [ugent-profiles plan](../archive/ugent-profiles.md) carries the implementation.
Decision 3 was revised the same day, before the work left its branch: it first gave
each model a package named after its standard (see Alternatives rejected).

## Context

The registry's names answer two different questions. `meemoo/basic` names a content
profile that Meemoo defines. `eark/dc` and `eark/mods` name a descriptive standard,
because the eark profiles had no content type to name: they declare
`CONTENTINFORMATIONTYPE="MIXED"`, and ADR-0030 put the choice of model in the name
instead. Every new profile would have to be forced onto one reading or the other.

The field has a word for what a profile is. CSIP calls it the content information type:
a package declares one in `csip:CONTENTINFORMATIONTYPE`, and a specification per type
(ERMS, SIARD, geospatial data, eHealth, archival information, 3D product models) fixes
its structure and metadata. None of them covers digitised library material. Meemoo SIP
1.2 defines content profiles (basic, bibliographic, material artwork), each declared by
a value in `csip:OTHERCONTENTINFORMATIONTYPE` and each fixing the descriptive standard,
the representation rules and the PREMIS rules. The descriptive standard follows from
the content type; it is not what the profile is.

UGent Library packages two content types for its own RODA instance:

- **basic**: digital-born or digitised resources the library has not necessarily
  accessioned or catalogued, such as an internal database dump or a deposited set of
  files. A short Simple Dublin Core record describes them.
- **bibliographic**: resources the library has accessioned and catalogued as its
  holdings: books, manuscripts, plans, maps, pictures, notated music, artworks. The
  catalogue record is the description's source, written as MODS 3.7, and its record
  identifier (the Alma MMS ID) identifies the intellectual entity.

Both names are also Meemoo's, with other meanings. Meemoo's basic is one media file in
one representation; UGent's basic is anything uncatalogued. Meemoo's bibliographic is
paged text with TIFF, ALTO and page order; UGent's criterion is whether the library
catalogued the item, so a map or a photograph of a painting is bibliographic too.

The MODS record `eark/mods` writes is already UGent's application profile of MODS: a
catalogue identifier, titles, and copies with call number, barcode and enumeration.
[ADR-0033](0033-ugent-first-profiles-of-your-own.md) left open whether that model keeps
the library's words or takes MODS's own.

The input reader treats a folder without `representations/` as one representation named
after the folder. Loose files do not say which kind of copy they are, and a closed set
of representation names would make the operator name the input folder itself
`preservation`, `access` or `archival`.

## Decision

**A profile is a content type, named `<owner>/<content type>`.** The owner is whoever
defines the rules: `meemoo` or `ugent`. The profiles are `meemoo/basic`, `ugent/basic`
and `ugent/bibliographic`. The specification a package conforms to beyond CSIP (Meemoo
SIP 1.2, or the E-ARK SIP 2.2.0) is a property of the profile, declared in
`mets/@PROFILE`, not part of its name. `eark/dc` and `eark/mods` retire; their
definitions become `ugent.Basic` and `ugent.Bibliographic` in `profiles/ugent`.

**Each UGent profile declares its content type.** The package METS carries
`CONTENTINFORMATIONTYPE="OTHER"` and the profile's name, `ugent/basic` or
`ugent/bibliographic`, in `OTHERCONTENTINFORMATIONTYPE`. CSIP6 asks only that this value
state the content information type. The value carries no version; it gains one when a
change to a profile's rules would refuse a package ingested earlier. The representation
METS keeps the declaration of [ADR-0013](0013-representation-type-from-label.md).

**A model lives with its owner's profiles.** `profiles/ugent` holds both definitions and
the two models they use: the Simple Dublin Core model (`ugent.Terms`) for `ugent/basic`
and the MODS 3.7 model (`ugent.Record`) for `ugent/bibliographic`, as Meemoo's
`dc+schema` model lives in `profiles/meemoo`. Both models are UGent's: each requires
what its profile page requires (an identifier and a title), and the MODS record is
UGent's application profile of MODS, named in the library's catalogue words (call
number, barcode, enumeration), which settles the question
[ADR-0033](0033-ugent-first-profiles-of-your-own.md) left open. A model moves to a
package of its own when a profile of another owner needs it.

**Every UGent package carries a description; representations are zero or more.** There
is no UGent profile without descriptive metadata: a package that nothing describes is
what basic exists to avoid. A package with a description and no representations is
allowed under both profiles, as a new package or as an update, as
[ADR-0029](0029-eark-profiles-offer-csip-as-written.md) decided for `eark/dc` and
`eark/mods`. ADR-0029's rule that a definition without a model carries no descriptive
metadata stands as architecture, with no profile that uses it.

**Essence always sits in a representation folder.** Under every profile, content files
live in `representations/<name>/`, with an optional `representations.csv`. A folder
without `representations/` has no representations. The rule belongs to the input
specification and is the same for all profiles, so `cli/input` stays unaware of them.

## Alternatives rejected

- **Keep the eark names and add UGent's rules to them.** The rules belong to UGent's
  content types, not to the E-ARK SIP, and the names would still say only the standard.
- **One eark profile with a flag for the standard.** The operator would choose two
  things where one is enough, and the content type would still have no name.
- **Name the UGent profiles after their standard** (`ugent/dc`, `ugent/mods`). The
  standard follows from the content type. A rule that depends on whether the library
  catalogued an item would have no name to belong to, and RODA would show `mods` as
  the AIP type.
- **A versioned URI as the content type**, as Meemoo declares. A URI needs a namespace
  that someone at the library owns and keeps, and CSIP6 does not ask for one.
- **Keep `MIXED`.** RODA would give every UGent AIP the same type, and the package would
  not say which rules it was built under.
- **Keep the flat input folder and let the representation names refuse it.** The
  operator would have to name the input folder after a kind of copy, and the message
  would point at the folder's name instead of at the missing `representations/`.
- **A package per descriptive standard** (`profiles/simpledc`, `profiles/mods`), which
  profiles of any owner import. Built first and dropped the same day. It put two kinds
  of package side by side under `profiles/`, owners and standards, the same mix this
  decision removes from the names. It was made for a reuse no second profile has:
  each model has one user, and a Meemoo bibliographic profile would follow Meemoo's
  MODS rules, not UGent's record. And the packages were not standard-neutral: they
  carried UGent's required fields and catalogue words.

## Consequences

- `--profile eark/dc` and `eark/mods` report an unknown profile. A program that resolves
  a profile by name uses the new names; one that holds a definition or builds a
  description imports `profiles/ugent`: `earkdc.Terms` is `ugent.Terms` and
  `earkmods.Record` is `ugent.Record`.
- The package METS of a UGent package changes once: `OTHER` and the profile's name
  replace `MIXED`. The commons-ip validation in `build.sh` confirms that the package
  stays valid when the change ships.
- RODA sets the AIP type from that value because commons-ip reads
  `CONTENTINFORMATIONTYPE` and `OTHERCONTENTINFORMATIONTYPE` as the package's content
  type. ADR-0013 records the same dependency for representation types, and the open
  commons-ip issue that would change it.
- An operator with a flat input folder moves its content into
  `representations/<name>/`, under `meemoo/basic` as well. The examples already use
  `representations/`.
- `ugent/basic` and `ugent/bibliographic` share their words with Meemoo's profiles and
  mean something else; the owner in the name and the profile pages keep them apart.
- `eark/none` is not built. A profile without descriptive metadata would be the empty
  case of the model list in the parked
  [entities-and-descriptions plan](../plans/entities-and-descriptions.md).
- ADRs and `docs/archive/` keep the old names as history.
