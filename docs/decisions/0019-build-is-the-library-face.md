# 0019 — build is the library's face; sip holds the shared types and the engine's graph

Status: **Accepted** (2026-09-29). Note, later the same day: `build.Material`
became `build.SourcePackage` and `build/material.go` became
`build/source.go`, closing the `Source*` family against `sip.Package`,
`Representation` and `File`; the text below keeps the name it was written
with.

## Context

[ADR-0018](0018-engine-and-profile-packages.md) put the engine in `build/`,
each profile in its own package and the renderers in `encoders/`, and fixed
the import direction. It did not say which package is the library's API.
The refactors of 2026-09-29 kept reopening that question: the folder reader
took a whole `Definition` to reach one method, the CLI kept a model of its
own that mirrored the library's field for field, and a proposal to move the
supplied tree into `sip/` was on the table. Each answer depended on where
an embedding system is expected to look.

Two trees describe a package. `build.Material`, `SourceRepresentation` and
`SourceFile` are the package as supplied: plain values holding source paths
and container-relative logical paths. `sip.Package`, `Entity`,
`Representation` and `File` are the package as built: a graph with minted
identifiers, generated document nodes, METS-relative paths, back-references
the PREMIS templates walk, and the fixity the writer fills in as each file
lands. Every supplied field reappears on the graph, and the assembler
copies it there. The graph also carried eleven exported setters that
assigned one exported field each, all called from the assembler alone, and
a sub-entity slot nothing assembled.

## Decision

**`build` is the library's face.** Every lever an implementer pulls lives
there: the material, the definition (handed out by the registry in
`profiles/`), the config, the builder, and the encoder interface a new
profile implements. An embedding system imports `build` and one profile
package; it names a `sip` type only when it spells a shared value type.

**`sip` holds the shared domain types and the engine's graph.** The shared
types (terms and descriptions, the METS declaration and its agents,
identifiers) sit there because the face and the renderers both need them
and neither may import the other. The graph sits there because the
renderers walk it. `sip` exports what a template or a second package reads
and nothing more: no setters, no slots for features without a design. The
assembler builds the graph by assigning fields.

**Validation of supplied data lives on the lever.** `Material.Validate` and
the name and attribute rules next to it are the one check an implementer
meets before a write. The graph has no caller-facing checks; its
invariants (identifiers minted, paths declared, a mime type on every node)
are the assembler's to keep.

**The two trees stay two.** The supplied tree is a plain value, validated
before any write and usable for another build afterwards. The graph is
what the renderers need. A build turns the first into the second.

## Alternatives rejected

- **One mutable model the caller constructs and `Build` completes**, the
  shape of commons-ip (`EARKSIP`, `IPRepresentation`, `IPFile` with a
  source path; `build()` fills checksums and writes METS) and of
  Archivematica's METS library. A failed build would leave the caller's
  object half-completed, `Path` would mean container-relative before
  assembly and METS-relative after on one field, and validation would have
  to know which state the object is in.
- **No supplied model, a writer or updater API** (`archive/zip.Writer`,
  OCFL client updaters, bagit's in-place bagging). Files are written as
  they are added, so whole-package rules (name uniqueness, required
  descriptive keys) could only run after writes. Keeping "validate first,
  then write" means accumulating into a value first, which is the material
  under another name.
- **The supplied tree in `sip/`.** It would put both states of a package
  in one package, next to each other, but it moves the main lever away
  from the face, and the CLI would import `sip` for the rules it phrases
  as violations.
- **`sip` as an `internal/` package.** `Build` returns a `*sip.Package`,
  and the profiles' terms types are lists of `sip.Term`. External callers
  can use values of internal types through type inference but cannot name
  them, which makes the documented API read wrong.

## Consequences

- The eleven setters and `Entity.Entities` are gone; the assembler assigns
  fields, and `DescriptiveFiles` lists the root entity's document alone.
  Output unchanged, checked with the structural comparison against both
  reference copies.
- CLAUDE.md's rule that domain validation goes on `sip/` types as
  `Validate` methods is replaced: validation of supplied data is on
  `build.Material`.
- `Build` returns `*sip.Package`, the one place the graph reaches a
  caller. The CLI's archiver reads the location and identifier from it.
  Whether systems that automate ingest workflows need the whole graph back
  (per-file fixity, say) or a small `build` result was left to a real
  need. Decided 2026-09-29, later the same day: it stays the graph. A
  caller reads it and does not build on it, and a smaller result would
  drop the per-file fixity that an ingest system is the most likely to
  want back.
- Multi-entity packages, when a design arrives, add the slot back with the
  design ([TODO.md](../TODO.md)).
