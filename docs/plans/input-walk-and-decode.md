# Plan: the input reader walks, then decodes

*Status: **proposed** (2026-10-02). Nothing implemented yet.*

This plan splits `input.Read` into two steps: a walk that finds the files of an input
folder, and a decode step that reads what is in them. It came up in the review of the
[metadata-model plan](metadata-model.md), when `Read` started taking a mapper and a
document only to hand them to the walker.

## Context

`input.Read(root, mapper, document)` makes one `folderWalker` and calls `read()`. The
walker does two jobs in one pass:

- **Finding files**: the reserved names, the folder rules (`expectFile`,
  `expectFolder`), symbolic links, OS artifacts and NFC collisions (`readDir`), the
  representation folders and their names, content files, documentation, and the
  received PREMIS file lists with the reserved `premis.xml` name.
- **Reading what is in them**, as it meets each file: the description of each level
  (`description`: description.csv parsed, mapped and validated, or a supplied
  document's root checked), the Siegfried report (`decodeSidecar`), and
  representations.csv (`applyRepresentations`, which reorders and labels the walked
  representations).

Only the description step needs the profile, but because it runs inside the walk the
walker carries the mapper, the document name and the document format for the whole
read. A reader of `walker.go` meets the profile in a file about folders, and a test of a
folder rule has to pass a mapper and a document it never uses.

The parsers are already pure ([ADR-0023](../decisions/0023-cli-input-one-package.md)):
`parseTerms` and `parseRepresentationRows` take bytes and return values and errors. What
mixes the two jobs is the orchestration around them.

## The rule

> **The walker finds files; decoders read them.**

The walk looks at names, kinds (file, folder, link) and places, never at contents. Every
step that opens a file to interpret it is a decoder, and the decoders run after the
walk, on what it found.

## Design

### The walk

`walk()` returns the source package as far as the folder's structure fills it, and an
inventory of the files the decoders read:

```go
// inventory lists the files of an input folder that a decoder reads after
// the walk. An empty path means the folder has no such file.
type inventory struct {
	pkg                levelSources            // the package level
	reps               map[string]levelSources // by representation name
	representationsCSV string
	sidecar            string
}

// levelSources are the description sources one level supplies.
type levelSources struct {
	rows     string // description.csv
	document string // the profile's supplied document
}
```

Representations are keyed by name, not by position, because applying
representations.csv reorders them.

The walk still needs the document's file name, to reserve it at both levels: a
`dc.xml` under eark is the description, not content. It needs nothing else from the
profile. The rule that representations.csv requires a representations/ folder stays in
the walk, because it is about which files exist, not what they say.

### The decode step

After the walk, `Read` runs the decoders on the skeleton and the inventory, in the
order of the input specification:

1. **§2:** the sidecar (`decodeSidecar`) and representations.csv
   (`applyRepresentations`).
2. **§3:** the description of each level (`description`): both sources at one level
   is a violation, neither at the package level is a violation, rows are parsed,
   mapped and validated, a supplied document's root is checked.

The mapper and the document format are parameters of the description decoder, not
fields of the per-read state.

### The per-read state

ADR-0023 keeps one struct per read for the input root and the violations so far, so
that about twenty functions can name paths in their messages without passing the root
around. That stays. It is renamed from `folderWalker` to `folderReader`, because it
now serves the walk and the decoders, and it loses the `mapper` and `documentFormat`
fields. Walk methods stay in `walker.go`; the decoders' orchestration moves to a new
`decode.go`, next to the pure parsers in `description.go`, `representations.go` and
`csv.go`.

### What does not change

- `input.Read(root, mapper, document)`, its arguments and its result.
- Every violation message, word for word.
- The pure parsers and their tests.

What changes for an operator is the order of the findings: structural findings first,
then the sidecar and representations.csv, then descriptions. Today they come in walk
order, mixed. No test depends on that order (the `cli/input` tests check for a message,
not its position), and the CLI prints the list as it comes.

## Steps

1. Introduce `inventory` and `levelSources`; the walk records the description sources,
   representations.csv and the sidecar there instead of decoding them; `Read` calls the
   decoders after the walk. Rename `folderWalker` to `folderReader`, drop its `mapper`
   and `documentFormat` fields, and move the decoder orchestration to `decode.go`.
2. Add a walk test that runs without a mapper or a document format: a folder rule
   (for example content beside representations/) reported by `walk()` alone.
3. Update the design doc's description of `cli/input` and CLAUDE.md's "System shape"
   step 1.

One change, reviewed as one; step 2 is its test.

## Acceptance

`go test ./...` passes with no test changed except for the rename and the new walk
test. The commons-ip validation in `build.sh` (all three profiles VALID) and the
structural comparison in `scripts/reference-diff.sh` pass with no change to the
reference copies: no generated package changes. `sip-creator check` on the three
examples and on a folder with findings of every kind (structure, sidecar,
representations.csv, description) reports the same set of messages as before.

## When this plan ships

Write an ADR that supersedes the part of ADR-0023 describing the per-read state
("`folderWalker` ... the profile's vocabulary and document format, which walks the
folder into the source package") with the rule above, and keeps the rest of ADR-0023
(one package, pure parsers). Move this plan to `docs/archive/`.
