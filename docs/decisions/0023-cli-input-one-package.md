# 0023 — `cli/input` stays one package: a folder walker with pure parsers

Status: **Accepted** (2026-10-02, when the [cli cleanup
plan](../archive/cli-cleanup.md) shipped). **Revised in part by [ADR-0024](0024-the-metadata-model.md)** (2026-10-02): `cli/input/vocabulary` is `cli/input/mapping`, and `Read` takes a mapper and the profile's document. **Superseded in part by [ADR-0027](0027-input-reader-walks-then-decodes.md)** (2026-10-05): the per-read state is `folderReader`, which walks the folder and then decodes what it found; the mapper and the document format are parameters of the description decoder.

## Context

`cli/input` reads an input folder into a `build.SourcePackage` and reports every
violation of the input specification at once. It does several things: it walks the
folder and its reserved names, parses the CSV of description.csv and
representations.csv, decodes a description from rows or from a supplied document,
decodes the Siegfried report, and collects the violations. A TODO item (raised
2026-10-01) asked to split it into one package per concern, because "input" says none
of that and the package grew with every plan.

Before the cleanup, every file in it worked on one shared value, then called
`directory`: the input root and the list of violations so far. The CSV decoders
recorded violations into it directly, so they could not be used or tested without a
folder on disk.

## Decision

Keep `cli/input` one package. Inside it:

- **One entry point**, `input.Read(root, vocabulary)`. It makes one `folderWalker` per
  call: the state of one read (the input root, the violations so far, the profile's
  vocabulary and document format), which walks the folder into the source package.
- **The content of a CSV file is judged by pure functions** that take the file's bytes
  and return values and errors (`parseStatements`, `parseRepresentationRows`, and
  `newCSVReader` for the rules every CSV in the input folder shares). The walker only
  reads the file from disk and adds the file name and line to each finding.

## Alternatives rejected

- **One package per concern** (folder walk, description.csv, representations.csv). The
  packages would share the per-read state, so its violation list and the helper that
  makes paths presentable would have to be exported: more API, no clearer code. The
  pure parsers give the separation that mattered (CSV content testable without a
  folder) without it, and could move to their own package later without further
  changes.
- **No per-read struct**: passing the root and the violation list to every function,
  or having every function return its findings. About twenty functions need the root
  for their messages; returning findings works for the parsers, which are leaves, but
  in the recursive walk every level would merge its children's findings.
- **An exported `Reader` holding the vocabulary**, as before the cleanup. Both commands
  built one and used it once, on the next line; it was a layer with no job.

## Consequences

- A new input file with its own syntax gets a pure parser next to the existing ones,
  tested with a byte slice; its folder-level handling goes on the walker.
- The violation list stays unexported. A program that builds a `build.SourcePackage`
  from its own store does not use `cli/input` at all.
- The package name "input" still says little about what is in it. What the
  `cli/input/vocabulary` package beside it should be called is open in the
  [metadata-model plan](../archive/metadata-model.md).
