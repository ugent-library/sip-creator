# 0027 — The input reader walks the folder, then decodes the files it found

Status: **Accepted** (2026-10-05, when the [walk-and-decode
plan](../archive/input-walk-and-decode.md) shipped). Supersedes the part of
[ADR-0023](0023-cli-input-one-package.md) that describes the per-read state; the rest
of ADR-0023 (one package, pure parsers) stands.

## Context

ADR-0023 kept one struct per read, `folderWalker`, holding the input root, the
violations so far, and the profile's mapper and document format. The walker found the
files of the folder and, as it met each one, read it: the description of each level,
the Siegfried report, representations.csv. Only the description step needed the mapper
and the format, but because it ran inside the walk, the walker carried both for the
whole read, and a reader of `walker.go` met the profile's mapping in a file about
folders.

## Decision

> **The walker finds files; decoders read them.**

The walk looks at names, kinds (file, folder, link) and places, never at contents. Every
step that opens a file to interpret it is a decoder, and the decoders run after the
walk, on what it found.

- **The per-read state** is `folderReader`: the input root, the violations so far, and
  the profile's document name. The reserved names include that name (a `dc.xml` under
  eark is the description, under basic it is content), so knowing it is part of the
  walk. `Read` sets it only when the profile's metadata model takes supplied documents.
- **The walk** (`walker.go`) returns the source package as far as the folder's
  structure fills it, and an inventory of the files the decoders read: each level's
  description files, representations.csv and the sidecar. It applies the rules about
  which files exist: representations.csv requires a representations/ folder, a level
  has at most one description file, and the package level has one. The last repeats
  `SourcePackage.Validate` ([ADR-0025](0025-every-package-carries-a-description.md)) so
  that `check` reports it.
- **The decoders** (`decode.go`) run in the order of the input specification: the
  sidecar and representations.csv, then each level's description. The mapper and the
  document format are parameters of the description decoder, not fields of the
  per-read state.

## Alternatives rejected

- **Pass `input.DocumentSpec` to the walk and the decoder, each asserting the format.**
  The check that the profile takes a document would exist twice; if the two ever
  disagreed, the walk would reserve a name for which the decoder holds a nil format.
- **A `descriptionDecoder` struct holding the mapper, the format and the name.** It puts
  the profile's values where the rule says they belong, but it is a new type for three
  call sites, and the walk still needs the name from somewhere. Worth revisiting if more
  decoding comes to depend on the profile.
- **Leave the presence rules in the description decoder.** When both description files
  are present the walk records neither, and the decoder would then add a second finding
  that the package has no description.

## Consequences

- A folder rule can be tested on the walk alone, with no mapper and no profile.
- Findings come in a fixed order: the folder's structure first, then the sidecar and
  representations.csv, then descriptions. The set of findings is the same as before.
- A profile's rules on the shape of the package are not the reader's:
  `Definition.ValidateSource` holds them, and `check` runs it after `Read`
  ([ADR-0026](0026-profile-rules-on-the-definition.md)).
