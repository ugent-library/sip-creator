# 0020 — The profile is builder configuration; a package's own METS values ride on the source package

Status: **Accepted** (2026-09-29).

## Context

`Builder.Build` took two arguments, the profile definition and the source
package, on every call. The definition is chosen once and reused: one
profile per deployment, or one builder per profile in a system that serves
several. The source package varies per call. Passing both per call hid
that difference, and it hid a second one: the definition carried two values
that are per package, not per profile. The CLI set `Declaration.RecordStatus`
from `--status` and `Declaration.Type` from `--content-category` on the
definition before each build. A system building many packages through one
builder could not have said "this one is a supplement" without rebuilding
the builder.

The reader in `cli/input` had just taken the same shape, `input.New(builder)`
then `Read(root)`: construct with what is fixed, call with what varies. The
standard library does the same: `json.NewEncoder(w).Encode(v)`,
`zip.NewWriter(w).Create(name)`, an `http.Client` holding its transport while
`Do` takes the request.

## Decision

**The profile is builder configuration.** `build.Config` carries the
profile next to the destination and the logger, `build.New` refuses a
profile without a descriptive encoder, and `Builder.Build(source)` takes
the source package alone. A second profile is a second builder.

**A package's own METS values ride on the source package.** `RecordStatus`
(metsHdr/@RECORDSTATUS, SIP3 vocabulary) and `ContentCategory` (mets/@TYPE,
CSIP vocabulary) sit on `SourcePackage` next to `PackageIdentifier`, which
an update-class status already pairs with. Empty means the profile's value.
`SourcePackage.Validate` checks them with the rules the CLI already applied,
plus the half of the status-and-identifier pairing the library can hold:
an update-class status requires `PackageIdentifier`, because the link to
the earlier package is its reused `mets/@OBJID`. The reverse stays open,
since an identifier on a NEW package may be one a caller minted upstream
(the open question on minting authority in TODO.md).
The assembler takes a copy of the profile's declaration per build, applies
the package's values to it, and derives each representation's declaration
from that copy, so the graph never points into the builder's profile.

**The definition holds what is fixed for a profile.** Its encoder, its
emission flags, the descriptive file name, and the METS values every
package to that profile declares, with the deployment's submitter added by
`WithSubmitter`.

## Alternatives rejected

- **Keep `Build(def, source)` and add the per-package fields anyway.** The
  fields would have two homes, on the definition and on the source
  package, and a reader could not tell which one a build reads.
- **Leave the per-package values on the definition.** A shared builder
  could then not express a per-package status; the CLI worked only because
  it builds one package per process.
- **A builder per package, constructed with a definition mutated for it.**
  It keeps per-package facts on a type documented as a profile, and it
  makes the builder's own lifetime a per-package one for no gain.

## Consequences

- Library callers change: `build.New` returns an error and takes the
  profile, `Build` takes one argument, and per-package status and category
  move from the definition's declaration to the source package. The README
  example shows the shape.
- `Builder`'s fields are unexported; `New` is the only way to make one.
- Output unchanged: both profiles compare structurally identical to their
  reference copies, and a test pins that the package's values land on the
  package and representation declarations and leave the profile's alone.
- Note, later the same day: the record status became a typed vocabulary.
  `sip.RecordStatus` carries the six SIP3 values as constants, with
  `ParseRecordStatus` (text in any case onto the vocabulary), `IsValid`
  (exact membership) and `IsUpdate` (the statuses that reference an earlier
  package). The free functions `ValidateRecordStatus` and
  `IsUpdateRecordStatus` are gone. `SourcePackage.RecordStatus` and
  `MetsDeclaration.RecordStatus` carry the type, `SourcePackage.Validate`
  checks the value exactly, and the CLI parses `--status` once. The type
  lives with the declaration it feeds, the way the identifier scheme lives
  with the identifiers: a shared value's own contract sits in `sip`, and
  the lever calls it. SIP3 is a closed list, unlike the CSIP content
  category, which stays a string.
