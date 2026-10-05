# Plan: the input reader walks, then decodes

*Status: **proposed** (2026-10-02, revised 2026-10-05). Nothing implemented yet.*

This plan splits `input.Read` into two steps: a walk that finds the files of an input
folder, and a decode step that reads what is in them. It came up in the review of the
[metadata-model plan](../archive/metadata-model.md), when `Read` started taking a mapper and a
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

Only the description step needs the mapper and the document format, but because it
runs inside the walk the walker carries both for the whole read. A reader of
`walker.go` meets the profile's mapping in a file about folders, and has to read the
decoders to learn which folder rules hold. A test of a folder rule also has to pass a
mapper and a document it never uses, though that costs little today.

The parsers are already pure ([ADR-0023](../decisions/0023-cli-input-one-package.md)):
`parseTerms` and `parseRepresentationRows` take bytes and return values and errors. What
mixes the two jobs is the orchestration around them.

## The rule

> **The walker finds files; decoders read them.**

The walk looks at names, kinds (file, folder, link) and places, never at contents. Every
step that opens a file to interpret it is a decoder, and the decoders run after the
walk, on what it found.

The reserved names include the profile's document name: a `dc.xml` under eark is the
description, not content, while under basic it is content like any other file. Knowing
that name is part of the walk, not an exception to the rule.

## Design

### The walk

`walk()` returns the source package as far as the folder's structure fills it, and an
inventory of the files the decoders read:

```go
// inventory lists the files of an input folder that a decoder reads after
// the walk. An empty path means the folder has no such file. reps has an
// entry per folder under representations/; a flat folder has none, because
// its description.csv describes the package.
type inventory struct {
	pkg                descriptionFiles            // the package level
	reps               map[string]descriptionFiles // by representation name
	representationsCSV string
	sidecar            string
}

// descriptionFiles are the files one level may describe itself with.
type descriptionFiles struct {
	rows     string // description.csv
	document string // the profile's supplied document
}
```

Representations are keyed by name, not by position, because applying
representations.csv reorders them. The names are unique: `readDir` already reports two
names that are the same after NFC normalization.

Two rules about which files exist, not what they say, belong to the walk:

- **representations.csv requires a representations/ folder.** In a flat folder the walk
  reports the file and leaves `inventory.representationsCSV` empty. Recording it
  would run `applyRepresentations` against the single flat representation and add
  findings ("there is no folder representations/…", "representations/… is not
  listed") that the folder does not deserve and that today's reader does not report.
- **One description per level.** When a level has both description.csv and the
  profile's supplied document, the walk reports it (the message moves from
  `description` unchanged) and records neither file. `build.SourcePackage` has one
  description per level, so only the reader can check this rule.
- **A description at the package level.** When the package level has neither file,
  the walk reports it (`violateMissingDescription`, unchanged). The case where both
  are present is not a missing description: the walk has already reported it, and
  reports nothing more.

The last rule repeats a library rule: `SourcePackage.Validate` refuses a package
without a description ("no descriptive metadata supplied"). The reader reports it too,
naming the input specification, because `check` runs `Read` and never the library's
`Validate`; without it, `check` would pass a folder that `create` then refuses with a
message that names no file. The rule holds under every profile
([ADR-0025](../decisions/0025-every-package-carries-a-description.md)).

All three rules sit in one walk helper that takes a level's `descriptionFiles` as the
walk found them and returns what the inventory records. The decoder then sees at most
one file per level and never reports on presence.

### The per-read state

ADR-0023 keeps one struct per read for the input root and the violations so far, so
that about twenty functions can name paths in their messages without passing the root
around. That stays. It is renamed from `folderWalker` to `folderReader`, because it
now serves the walk and the decoders. It loses the `mapper` and `documentFormat`
fields and keeps `documentName`:

```go
type folderReader struct {
	root       string     // all messages and report keys are relative to it
	violations Violations // the findings so far
	// documentName is the file name reserved for the profile's supplied
	// descriptive document at both levels; empty under a profile that
	// takes rows only.
	documentName string
}
```

`Read` resolves the profile's document once: it sets `documentName` only when
`document.Model` implements `build.DocumentFormat`, and keeps the format in a local
variable for the decode step. A non-empty `documentName` then means the profile takes a
supplied document, so `isDocumentName` and `violateMissingDescription` test
`documentName == ""` instead of a nil format.

Walk methods stay in `walker.go`, `violateMissingDescription` among them. The
decoders' orchestration (`description`, `decodeDescription`, `readDocument`,
`decodeSidecar`, `applyRepresentations`) moves to a new `decode.go`, next to the pure parsers in
`description.go`, `representations.go` and `csv.go`. `document.go` keeps the
`Document` type.

### The decode step

After the walk, `Read` runs the decoders on the skeleton and the inventory, in the
order of the input specification:

1. **§2:** the sidecar (`decodeSidecar`) and representations.csv
   (`applyRepresentations`), each only when the walk found the file. The checks for an
   empty path sit in `Read`, so each decoder reads a file that exists and `Read` shows
   the order.
2. **§3:** the description of each level (`description`), from the one file the walk
   recorded, if any: rows are parsed, mapped and validated (with `ValidateRequired`
   at the package level), a supplied document's root is checked.

```go
	source, inv := r.walk()
	if inv.sidecar != "" {
		source.Characterization = r.decodeSidecar(inv.sidecar)
	}
	if inv.representationsCSV != "" {
		source.Representations = r.applyRepresentations(inv.representationsCSV, source.Representations)
	}
	source.Description = r.description(inv.pkg, true, mapper, format)
	for i := range source.Representations {
		rep := &source.Representations[i]
		rep.Description = r.description(inv.reps[rep.Name], false, mapper, format)
	}
```

The mapper and the document format are parameters of the description decoder, not
fields of the per-read state: `description` passes the mapper to `decodeDescription`
and the format to `readDocument`. The format is never nil in `readDocument`: the walk
only finds a document when `documentName` is set, and `Read` sets the name only
together with the format. `readDocument` says so in one comment line.

### What does not change

- `input.Read(root, mapper, document)`, its arguments and its result.
- `input.Document` and the three places that build it from a `build.Definition`.
- Every violation message, word for word.
- The pure parsers and their tests.

What changes for an operator is the order of the findings: structural findings first,
then the sidecar and representations.csv, then descriptions. Today they come in walk
order, mixed. No test depends on that order (the `cli/input` tests check for a message
or a count, not a position), and the CLI prints the list as it comes.

## Steps

1. Introduce `inventory` and `descriptionFiles`; the walk records the description
   files, representations.csv and the sidecar there instead of decoding them, and
   reports the three rules about which files exist (above); `Read`
   resolves the document format and calls the decoders after the walk. Rename
   `folderWalker` to `folderReader`, drop its `mapper` and `documentFormat` fields, pass
   the mapper to `decodeDescription` and the format to `readDocument`, and move the
   decoder orchestration to `decode.go`, taking `readDocument` out of `document.go`.
2. Add a walk test that runs without a mapper or a document format: a folder rule
   (for example content beside representations/) reported by
   `(&folderReader{root: dir}).walk()` alone.
3. Update the design doc's description of `cli/input` and CLAUDE.md's "System shape"
   step 1.

One change, reviewed as one; step 2 is its test.

## Acceptance

`go test ./...` passes with no existing test changed; one walk test is added. No test
constructs the per-read state, so the rename touches none. The commons-ip validation in `build.sh` (all three profiles VALID) and the
structural comparison in `scripts/reference-diff.sh` pass with no change to the
reference copies: no generated package changes. `sip-creator check` on the three
examples and on a folder with findings of every kind (structure, sidecar,
representations.csv, description) reports the same set of messages as before.

## When this plan ships

Write an ADR that supersedes the part of ADR-0023 describing the per-read state
("`folderWalker` ... the profile's vocabulary and document format, which walks the
folder into the source package") with the rule above, including that the reserved
names the walk knows include the profile's document name, and keeps the rest of
ADR-0023 (one package, pure parsers). Move this plan to `docs/archive/`.
