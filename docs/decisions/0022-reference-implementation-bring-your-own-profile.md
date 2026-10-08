# 0022 — The library is a reference implementation others can use; an institution brings its own profile

Status: **Accepted** (2026-10-01, when the
[descriptive-model plan](../archive/descriptive-model.md)'s S5 shipped;
drafted 2026-09-30 and agreed in review the same day). **Superseded in
part by [ADR-0033](0033-ugent-first-profiles-of-your-own.md)** (2026-10-06):
the library serves UGent Library first and is no longer a reference
implementation for other institutions; the MODS model's vocabulary is
decided again when the mods-coverage plan resumes. The extension point and
the typed constants stand. **Note,
2026-10-06:** `build.DescriptionEncoder` below is `build.MetadataModel`
since 2026-10-02 ([ADR-0024](0024-the-metadata-model.md)). **Note,
2026-10-06:** a profile brings its own XSDs as well: `MetadataModel.Schemas`
returns each schema with its contents (`build.Schema`), so an XSD no longer
has to be bundled in this repository. `build.BundledSchemas` returns the
bundled ones, and `Example_ownProfile` in `build/example_test.go` builds a
package with a profile defined outside `profiles/`. The tagged release and
changelog the consequences below call for wait for the first outside
caller (docs/TODO.md). **Note, 2026-10-07:**
[ADR-0034](0034-a-profile-is-a-content-type.md) settles the MODS model's vocabulary:
it keeps the library's catalogue words, because it is UGent's application profile of
MODS, written by `ugent/bibliographic`.

## Context

Every document so far names one audience: digital-preservation staff at
UGent Library preparing SIPs for meemoo and for UGent's RODA instance.
The code follows: the MODS profile's comments speak of "an Alma MMS ID at
UGent Library", the identifier's `type` attribute is a constant the
profile decides, and the two eark profiles carry choices made for RODA
(no PREMIS, because RODA drops package PREMIS that describes no agents or
events; the representation type in the content typing, because RODA reads
it there, [ADR-0013](0013-representation-type-from-label.md)).

The library is nevertheless shaped for others. Since
[ADR-0018](0018-engine-and-profile-packages.md),
[ADR-0019](0019-build-is-the-library-face.md) and
[ADR-0020](0020-profile-is-builder-configuration.md), a profile is an
exported `build.Definition` carrying a `build.DescriptionEncoder`, the
engine imports no profile, and `build.New` takes any definition. An
institution with its own descriptive standard can already write a
profile package and build packages with it. Nothing says so, and the
in-tree profiles read as the only ones.

The review of 2026-09-30 that typed the MODS record
([ADR-0021](0021-descriptive-model-follows-its-standard.md)) asked what
the model should be if others use it. UGent's application profile of MODS
(a catalogue number, titles, holdings) is what UGent's callers hold, but
a reference implementation of "E-ARK with MODS" should speak MODS.

## Decision

**The library is a reference implementation of E-ARK SIP packaging that
other institutions can use as it stands.** Its documents are written for
them as well as for UGent: UGent's usage is the example, never the rule.

**A profile is something an institution brings.** The route is the one
the engine already has: a description type implementing
`sip.Description`, an encoder implementing `build.DescriptionEncoder`, an
exported `build.Definition`, handed to `build.New`. The registry in
`profiles/` is the CLI's list of what `--profile` can name, not the
library's limit, and the three in-tree profiles are reference
implementations of that route.

**Where a profile would decide an attribute value for the caller, the
caller decides, from a closed set.** A `type` on an identifier, a name
type, a relator role, a date encoding: each is a typed constant set
drawn from the standard's own enumeration or suggested list, with a zero
value that means the common case. The set is closed for the same reason
the CSV vocabulary is ([ADR-0011](0011-closed-descriptive-vocabulary.md)):
two callers making the same choice emit the same document.

**The eark-mods profile is plain E-ARK with a MODS 3.7 writer, and the
writer speaks MODS.** Its model covers the standard's top-level elements
in MODS's own words (`TitleInfo`, `NamePart`, `ShelfLocator`), one type
per element with the subelements and attributes implementers use, built
in tiers under the [mods-coverage plan](../archive/mods-coverage.md). What
the model cannot say travels as a supplied `mods.xml` (ADR-0021).

**The eark profiles' RODA choices are named as such.** No PREMIS and the
representation type in the content typing are the reference's choices
for RODA, not E-ARK's rules. They stay as they are until a consumer that
wants otherwise appears; the design doc says which choices they are.

## Alternatives rejected

- **Keep the UGent-only framing.** Cheapest, and it leaves a library whose
  extension point is real but invisible, and whose MODS model bakes one
  institution's identifier convention into a template.
- **Make the registry the extension point**, with profiles registered at
  run time. A library caller needs no registry: it holds its definition
  and hands it to `build.New`. The registry serves the CLI's flag, and a
  closed list there is what keeps `--profile` honest.
- **A model that speaks the caller's domain** (call number, barcode,
  catalogue number) for everyone. Those are UGent's words for MODS
  elements every library uses; another institution would translate twice.
  The MODS words are the shared ones.
- **Make the RODA choices configurable now.** Data nobody sets is
  speculation; the choices are named instead, and become data or a
  fourth profile when a non-RODA E-ARK consumer arrives.

## Consequences

- The audience sections of `CLAUDE.md`, the README and the design doc
  change; the README gains the bring-your-own-profile route with the
  three things a profile package exports.
- The MODS model grows from three fields to the standard's top-level
  elements, in tiers, under a plan of its own. The ingest system's call
  becomes more explicit (`Location.HoldingSimple.CopyInformation` where
  it said `Items`) and no less clear. Today's document stays byte for
  byte, pinned by the golden test.
- Every exported symbol weighs more once others depend on it. Breaking
  changes keep being recorded in commit messages; a tagged release and a
  changelog become due before an external caller is invited to depend on
  the module.
- The CSV vocabulary stays a flat subset of what the model can say and
  grows key by key; nothing obliges it to express every element.
- The eark profiles' RODA choices are an open question in the
  descriptive-model plan until a consumer decides them.
