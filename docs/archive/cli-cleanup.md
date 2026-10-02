# Plan: cleaning up the cli package

*Status: **shipped** (2026-10-02). Phases 1 to 6 done, with the steps marked dropped left out; 1.5 parked. The decision to keep `cli/input` one package is [ADR-0023](../decisions/0023-cli-input-one-package.md).*

This plan collects a review of `cli/` and `cli/input/` (including
`cli/input/vocabulary/`) into small, separate steps. None of the steps changes a
generated package: every step must leave `go test ./...`, the commons-ip validation
in `build.sh` and the structural comparison in `scripts/reference-diff.sh` passing
with no change to the reference copy. Some steps change the wording of problems the
CLI reports or the stream they go to; those are called out per step.

## Decision: keep `cli/input` one package

*Recorded in [ADR-0023](../decisions/0023-cli-input-one-package.md).*

`cli/input` is about 750 lines of non-test code with one job: read a folder into a
`build.SourcePackage` and collect every problem found on the way. Every file in it
shares the `*folderWalker` value (the problem list and the display paths). Splitting it
into packages per component (folder walk, description.csv, representations.csv)
would mean exporting that collector and the path helper, which adds API surface
without making the code clearer.

The better move is to make the two CSV decoders pure functions that take the file's
content and return values plus errors (Phase 2). That makes them testable without
a folder on disk, and if a separate package is ever wanted, the pure parsers can move
without further changes.

## Phase 1: CLI behavior fixes

Small changes that fix behavior an operator can see.

### 1.1 `check` no longer reads configuration

*Done (2026-10-01).* `Run()` loads `.env` and parses the environment config before any command runs
([cli.go](../../cli/cli.go)). A malformed `.env` therefore makes `sip-creator check`
fail, although [ADR-0010](../decisions/0010-config-over-self-describing-input.md)
says check needs no configuration. Load `.env` and the config in `create` only.

### 1.2 Logs go to stderr

*Done (2026-10-01).* `newLogger` writes to `os.Stdout`. Anyone who pipes `create`, or reads the `OK: …`
line `check` prints to stdout from a script, gets log lines mixed in. Write logs to
`os.Stderr`.

### 1.3 One way to print violations

*Done (2026-10-01).* `check` prints one line per violation plus a count; `create` wraps the joined list in
`input folder … does not conform to the input specification:`. Add one
`reportViolations(cmd, src, err) error` and use it in both commands. This changes what
`create` prints for a bad folder.

### 1.4 Shorter `create`

*Done (2026-10-01).*

`createCmd.RunE` resolves the profile, fills the submitter, pairs `--status` with
`--updates`, picks the content category, reads the folder, builds and zips. Pull out
`recordStatusFromFlags(cmd) (sip.RecordStatus, string, error)` for the pairing rule,
the one piece of flag policy in `create`. The content category stays inline: four
lines with one caller read better in place than behind a function.

Construct the zipper only when `--no-zip` is not set. Rename `flagStatus` to
`statusText`.

### 1.5 Later, if wanted: no globals

`logger` and `rootCmd` are package globals and commands register in `init()` (the
config stopped being one in 1.1: `create` loads it). A
`newRootCmd(logger *slog.Logger) *cobra.Command` would let tests run a command
in-process. Worth doing once tests for 1.4's flag rules
are wanted; not before.

## Phase 2: pure CSV parsers

*Done (2026-10-01), except 2.2, which was dropped.*

### 2.1 Parse description.csv without the folder walker

*Done.* `decodeDescription` ([description.go](../../cli/input/description.go)) handled
CSV syntax, called the vocabulary, ran `Validate`/`ValidateRequired`, and mapped
errors back to lines. `parseKey` took a file name and line only to report through
`violate`. Now the folder walker only reads the file (`os.ReadFile`); everything
about its content, encoding included, belongs to the parser:

```go
func newCSVReader(data []byte) (*csv.Reader, error)                // csv.go: BOM, UTF-8, any row width
func parseStatements(data []byte) ([]Statement, []error, error)    // header, two columns, key[lang]
func parseKey(raw string) (key, lang string, err error)
```

`parseStatements` returns its findings in the rows separately from an error meaning
the content cannot be read as CSV at all. The decoder stops on the second, so content
that is not UTF-8 is one violation instead of also tripping the required-keys rules.

### 2.2 One error shape for a finding about a row

*Dropped.* The idea was to convert `*sip.TermError{Index}` into a line in a
`lineOf(err, statements)` helper, so the reporting loop in `decodeDescription`
handles one shape instead of two. After 2.1 that loop is a three-case switch with a
comment explaining both shapes; a helper would move the same logic out of sight
without making it simpler.

### 2.3 Parse representations.csv without the folder walker

*Done.* Same split for representations.csv
([representations.go](../../cli/input/representations.go)):

```go
func parseRepresentationRows(data []byte) ([]repRow, []error, error)
func parseRepresentationsHeader(header []string) (repColumns, error)
```

A header can have several problems at once; they are joined into the one error, and
the decoder reports each. Row findings are an unexported `*rowError{line}`;
`StatementError` stays the vocabularies' type. Matching rows to folders
(`applyRepresentations`) stays on the folder walker, because it needs the folders read.

### 2.4 Tests

*Done.* The syntax cases call the parsers with a byte slice. The folder tests keep
what needs a folder: matching rows to representation folders, the vocabulary, and how findings
are reported with the file name and line, plus a test that content that is not UTF-8
is one violation.

## Phase 3: less repetition in the folder walk

### 3.1 One check for the kind of a reserved name

*Done (2026-10-01), without the table first proposed.* `read()` and
`readRepresentation()` in [walker.go](../../cli/input/walker.go) wrote out the
same "X is a folder; the reserved name is for …" check nine times. Each `case` now
calls `expectFile(e, src, holds)` or `expectFolder(e, src, holds)`, which hold the two
messages once.

The first proposal was a table of reserved names (kind, message text, root only) with
one lookup in front of both switches. It was built and thrown away: the switches stay
anyway, because each name goes somewhere different, so every name appeared twice
(table and switch), a reader had to follow four places to see how one name is handled,
and the file did not get shorter. The wording it was meant to make consistent across
the two levels already was: at the root, the entry's display path is its name.

### 3.2 One way to collect content files

*Dropped.* The idea was one `collectEntries(base string, entries []os.DirEntry)` for
`collectFiles`, `readFlatRepresentation` and the default branch of
`readRepresentation`. `collectFiles` is already a three-line wrapper and the repeated
loop is four lines; a merged function with two path arguments to tell apart would
save little and cost the reader more.

### 3.3 One type for the supplied document

*Dropped.* The idea was one unexported type, embedded in `Eark` and `EarkMods`, for
their `DocumentName` and `CheckDocument` methods. Those are two pairs of three-line
methods; embedding would hide where the methods come from. The question of what
`DocumentVocabulary` should be called stays in Phase 4.

## Phase 4: names

### 4.1 The folder walker, and folder for the input

*Done (2026-10-01).* Three changes that came out of the question what to call the
`directory` struct, whose name clashed with `expectFolder` and the messages, which say
"folder":

- **`input.Read(root, vocabulary)` replaces `Reader` and `New`.** `Reader` held only the
  vocabulary and both callers used it once, on the next line; it was a layer with no
  job.
- **The per-read state is a `folderWalker`** in [walker.go](../../cli/input/walker.go),
  receiver `w`: it walks one input folder into a `build.SourcePackage`, collecting
  every violation on the way, and `Read` makes one per call. Rejected names: `folder`
  (the thing walked, not the walk), `reader` (reads as `io.Reader`), `walk` and
  `inspection` (too short or too abstract). A struct stays, rather than parameters or
  returned findings, because nearly every function needs the root and the findings
  list.
- **Folder for the input, directory for the package.** The representations.csv column
  is `folder` (a file with a `directory` header is refused; the project is not in use
  yet), and the input side says "folder" throughout. "Directory" stays for the package
  the tool writes and for fixed terms. The rule is in CLAUDE.md.

### 4.2 Remaining names

*Done (2026-10-01): three rows done, one moved to `docs/TODO.md`, the rest dropped.*
Each row was weighed before any diff, as with 3.1 to 3.3: a rename that makes a reader
stop and look twice is not worth doing.

Done:

- **Path variables.** `description`, `document`, `repsDir`, `repsCSV` in `read()` and
  `readRepresentation()`, and the `rows`, `document` parameters of `description(...)`,
  held paths but read like the things they point at, next to `source.Description` and
  the `description` method. Now `descriptionPath`, `documentPath`,
  `representationsPath`, `representationsCSVPath` and `rowsPath`. The `packageLevel`
  bool stays: two call sites, each in a function whose level is obvious.
- **`name` shadowing in `readRepresentation`.** The parameter is now `repName`, so the
  loop's `name := e.Name()` no longer hides it. Go's `shadow` analyzer found no other
  case where one name means two things; the remaining reports are the ordinary
  `if err := …; err != nil` pattern.
- **`placement.lang`, `placement.repeat`** in `earkmods.go` are `takesLang` and
  `occurs`: `!key.lang` read like a language value next to `s.Lang`.

Done later, with the library:

- **`CheckDocument` → `Validate…`.** CLAUDE.md's naming rule applies, but
  `CheckDocument` passed through to the library's `CheckDescriptiveDocument`, which
  broke the same rule, so the two were renamed together (2026-10-02):
  `build.DescriptiveDocumentChecker` became `build.DocumentFormat` and
  `input.DocumentVocabulary` became `input.DocumentFormat`, both with
  `ValidateDocumentRoot`. "Format" is how a profile's metadata model is stored as a
  file; the [metadata-model plan](metadata-model.md) carries that naming further.

Dropped:

- **`violate` → `addViolation`, `rel` → `display`:** about 80 call sites for little
  gain; one call shows what each does.
- **`DocumentVocabulary` → `SuppliedDocument`:** reads as the document itself, next to
  `build.DescriptiveDocument`. Superseded by `DocumentFormat`, above.
- **`decodeRepresentations`, `applyRepresentations`:** they live in
  `representations.go`, and `parse…` now names the pure parsers.
- **`repRow.kind` → `typ`:** `kind` is the usual stand-in for the keyword `type`.
- **`decodeSidecar`, `sidecarName`:** "sidecar" is the project's word for the
  report (ADR-0009, CLAUDE.md, the README).
- **`def` → `definition`:** clear in context.

## Phase 5: comments

*Done (2026-10-01), in one commit: every bullet below, plus "the reader" in comments
that meant `Read`.*

Apply the CLAUDE.md rules ("comment the thing, not its callers"; as short as it can
be). Do this per file.

- `checkCmd` ([check_cmd.go](../../cli/check_cmd.go)): seven lines down to two:
  *"checkCmd validates an input folder without building. Checks on file contents
  (received PREMIS, the characterization report) run only in create."*
- `readDir`: keep the three rules (symbolic links, OS files, names that collide after
  NFC) and "os.ReadDir sorts by name, so traversal order is stable". Drop the mention
  of `scripts/reference-diff.sh`, which names a caller.
- `Vocabulary.Description` (9 lines), `DocumentVocabulary` (8) and `EarkMods` (11):
  describe only the contract, not what `Read` does with it. The same goes for the
  other comments that still say "the reader" for the reading code.
- "(collect-all)" in walker.go: write "so later problems in the folder are still
  reported".
- `vocabulary.go`: rewrap the overlong line in the `checkDocument` comment.

## Phase 6: small cleanups

*Done (2026-10-01), except moving the test out of `walker_test.go`: it tests
`isOSArtifact`, which lives in `walker.go`, so `walker_test.go` is where it belongs.
The `newFile` fallbacks became `filepath.Rel` calls whose error is ignored, with a
one-line reason.*

- `newFile`: the fallbacks for a failed `filepath.Rel` cannot happen for paths under
  the root, and if they did they would put an absolute path in `Key`. Remove them.
- Use `errors.AsType` in `description.go`, as `check_cmd.go` does.
- Test files: `builder_test.go` tests that a Go-built source package matches the
  folder; rename it `source_package_test.go`. The `TestRepresentationCSV…` tests in
  `description_test.go` test a representation's own description.csv, not
  representations.csv: rename them `TestRepresentationDescription…`. Move the one test
  in `walker_test.go` into `read_test.go`.

## Order

All phases are done; 1.5 stays parked until someone wants tests for the CLI flags.

## When this plan ships

Done (2026-10-02): `sip-creator-design.md` and CLAUDE.md describe `input.Read`, the
folder walker and the parsers;
[ADR-0023](../decisions/0023-cli-input-one-package.md) records keeping `cli/input`
one package; this plan moved to `docs/archive/`. The naming question it ended on
continues in the [metadata-model plan](metadata-model.md).
