# 0016 — Descriptive input is rows or a supplied document, and the profile fixes the standard

Status: **Accepted in part** (2026-09-28): the profile fixes the
descriptive standard. **Superseded in part** (2026-09-28, later the same
day): the rows file is `description.csv` under every profile and the
`--profile` flag, on `check` as on `create`, says which vocabulary it is
in; the names by standard (`dcschema.csv` and `dc.csv`, shipped in the
[eark-mods plan](../archive/eark-mods.md)'s S2, and the planned `mods.csv`)
and the one-standard-per-folder rule are withdrawn, see the note on the
naming paragraph below. **Superseded in part by
[ADR-0017](0017-supplied-descriptive-document-deferred.md)** (2026-09-28):
the supplied-document route (`dc.xml`, `mods.xml`, the `DescriptiveDocument`
input, its structural checks and the no-identity rule) is deferred; the
paragraphs below that describe it record the design for when it returns.
(Drafted 2026-09-23; agreed in review on 2026-09-15, the items file on
2026-09-23.) **Revived in part by
[ADR-0021](0021-descriptive-model-follows-its-standard.md)** (2026-09-30):
the supplied document returns for the two eark profiles (`dc.xml`,
`mods.xml`, one per level) as a second description type behind
`sip.Description`, not as a path next to the terms; the structural check
and the no-identity rule below stand, the essence-path mechanics do not.
`items.csv` is withdrawn: the rows carry flat statements only, and a
record's copies reach a package through the library's record or a
supplied `mods.xml`. The CLI no longer hands rows to a `NewDescription`
on the profile: an adapter per profile on the CLI side builds the
description. **Note, 2026-10-06:** the supplied document is
`build.EncodedDescription` (2026-10-02); `DescriptiveDocument` below is its
earlier name.

## Context

Descriptive metadata reaches the tool one way today: as terms, decoded from
the input folder's `metadata.csv` or constructed by a library caller, which
the writer serializes into the profile's document. That covers records that
fit the closed table ([ADR-0011](0011-closed-descriptive-vocabulary.md)). It
does not cover a record that already exists as a document: a catalogue
export, or a MODS record with structure the flat rows cannot say
([ADR-0015](0015-descriptive-worlds-dc-and-mods.md)). The input
specification has listed operator-supplied descriptive XML as deferred
since ADR-0011 chose against it for the first version.

With two descriptive standards the input can also be of the wrong kind:
MODS terms handed to a profile that emits Dublin Core, or a `mods.xml` next
to a profile whose METS declares `MDTYPE="DC"`. Something has to say which
standard a build speaks. And the folder has to say what it contains before
any profile is known, because `check` validates a folder without
configuration ([ADR-0010](0010-config-over-self-describing-input.md)).

A precedent exists for packaging XML the tool did not write. Received PREMIS
files are copied into the package as received, after one check: well-formed
XML whose root is `premis:premis` in the PREMIS 3 namespace. Schema
validation stays external ([ADR-0003](0003-validation-stays-external.md)).

## Decision

**The profile fixes the descriptive standard; the input supplies either
terms to encode in it or a finished document of it.** Terms of the other
standard, or a document of the other standard, are a build error before
any disk write. Meemoo profiles accept terms only: their document needs the
entity identifier swapped in and the meemoo namespace, which only the
encoder produces.

**A supplied document travels the essence path.** `profiles.Input` and
`SourceRepresentation` carry a `DescriptiveDocument` source path next to
`Descriptive`: exactly one of the two at package level, at most one per
representation. The assembler declares the descriptive `*sip.File` with
`Source` set, and the writer copies it as it copies essence, with fixity
computed by the store during the copy. A file without `Source` is generated
as before. The METS `dmdSec` types the document from the profile
declaration either way.

**A supplied document gets structural checks only, in process, with the
standard library.** Well-formed XML, the expected root element and
namespace: `simpledc` without namespace for Dublin Core, the shape the eark
template emits; `mods` in `http://www.loc.gov/mods/v3` carrying
`version="3.7"` for MODS. Nothing else. No XSD validation in the tool;
build.sh runs xmllint over every emitted `mods.xml` against the bundled
schema as the acceptance check.

**No identity checks on a supplied document.** Rows must carry `identifier`
and `title` at package level; a supplied document is trusted for its
content. Nothing in the plain E-ARK profiles reads the identifier back out
of the descriptive metadata: they emit no PREMIS, lift no local identifier
and swap none, so nothing later in the build depends on it.

**The CLI names files by standard.** `dcschema.csv` (meemoo's dc+schema
vocabulary), `dc.csv` (Simple Dublin Core) and `mods.csv` hold rows;
`dc.xml` and `mods.xml` hold supplied documents, and there is no
`dcschema.xml` because meemoo profiles take rows only; all at the input
root and inside each representation directory. `items.csv` holds a MODS
record's physical copies, at the root only and next to `mods.csv` only.
One source per level, one standard per folder: files of two different
standards anywhere in one folder are a violation. `metadata.csv` is
withdrawn; its presence is a violation that tells the operator to rename
it to `dcschema.csv` or `dc.csv`. The meemoo file is named after what its
rows are, mirroring the emitted `dc+schema.xml`, rather than after the
organisation.
The folder stays self-describing, so `check` keeps taking no configuration.
A mismatch between the folder's standard and the chosen profile surfaces at
`create`.

(Superseded 2026-09-28, later the same day: the rows file is
`description.csv` under every profile, and `check` takes the same
`--profile` flag as `create`, which says which vocabulary the rows are
in. The name had stated the profile a second time next to the flag,
reconciled by the encoder's type check at `create`, and `check` could not
report the mismatch. ADR-0010 puts the profile on the command line,
"flags, not folder contents", and its config-independence rule is about
installation settings, which a per-run flag is not: `check` still needs
no `.env`. The former names are not reserved and not checked for: a
`dcschema.csv` left in a folder is content like any other file, and the
missing `description.csv` is what `check` reports. `items.csv` for MODS
stays as planned, allowed only under the profile that reads it.
The CLI hands decoded rows to the profile's encoder, whose `NewDescription`
builds the description, so `cli/input` imports no profile package.)

## Alternatives rejected

- **Let the input decide the standard and the profile follow.** One profile
  would emit two different documents depending on what it was given, and
  `MDTYPE` and `MDTYPEVERSION` would stop being profile data
  ([ADR-0007](0007-profile-families-share-one-writer.md)).
- **XSD validation in process.** Needs a cgo binding to libxml2 or running
  an external tool from the build, both against the project's dependency
  rules, and it duplicates the acceptance check ADR-0003 keeps external.
- **Enforce identifier and title inside a supplied document.** Two XML
  shapes to parse for one check the repository performs on ingest anyway,
  with a wrong result on the edge: a document whose title sits in an
  element the check does not know would be refused for no gain.
- **Keep `metadata.csv` and let the profile decide what its keys mean.**
  `check` would need a profile flag to know which table to validate
  against, breaking ADR-0010's rule that a folder's conformance never
  depends on settings.
- **One merged key space for both standards**, so that `metadata.csv` can
  stay. Reintroduces the silent lossy mapping ADR-0011 removed: one key
  would mean an element in one profile and a subtree in another.

## Consequences

- A record the flat rows cannot express has a route into the package: the
  producer prepares the document and the tool packages it unchanged. The
  input specification's deferred item on operator-supplied descriptive XML
  is enacted for Dublin Core and MODS.
- Each standard has one accepted document shape, the one the tool emits
  itself, so consumers see the same shape whichever route produced it. A
  Dublin Core export in another wrapper (`oai_dc:dc`) must be rewrapped by
  the producer.
- Whether a supplied document is valid against its schema is checked by
  build.sh, not by the tool. A supplied `mods.xml` that is well-formed with
  the right root but invalid MODS is packaged, and the repository's own
  validation catches it, as for received PREMIS.
- Every existing input folder must rename `metadata.csv` to `dcschema.csv`
  (meemoo) or `dc.csv` (plain E-ARK), and an eark folder's rows move to
  Simple DC keys. The violation message says so; fixtures and the input
  specification change in the same steps.
- `check` tells an operator that a folder mixes standards or lacks a
  descriptive source, but not that it mismatches a profile; that is the
  first thing `create` reports.
- Library callers hand either a `Description` or a document path per
  level. Both, or neither at package level, is an input validation error
  before assembly.
