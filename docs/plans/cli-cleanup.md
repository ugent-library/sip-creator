# Plan: cleaning up the cli package

*Status: **proposed** (2026-10-01). Nothing implemented yet.*

This plan collects a review of `cli/` and `cli/input/` (including
`cli/input/vocabulary/`) into small, separate steps. None of the steps changes a
generated package: every step must leave `go test ./...`, the commons-ip validation
in `build.sh` and the structural comparison in `scripts/reference-diff.sh` passing
with no change to the reference copy. Some steps change the wording of problems the
CLI reports or the stream they go to; those are called out per step.

## Decision: keep `cli/input` one package

`cli/input` is about 750 lines of non-test code with one job: read a folder into a
`build.SourcePackage` and collect every problem found on the way. Every file in it
shares the `*directory` value (the problem list and the display paths). Splitting it
into packages per component (folder walk, description.csv, representations.csv)
would mean exporting that collector and the path helper, which adds API surface
without making the code clearer.

The better move is to make the two CSV decoders pure functions that take an
`io.Reader` and return values plus errors (Phase 2). That makes them testable without
a folder on disk, and if a separate package is ever wanted, the pure parsers can move
without further changes.

## Phase 1: CLI behavior fixes

Small changes that fix behavior an operator can see.

### 1.1 `check` no longer reads configuration

`Run()` loads `.env` and parses the environment config before any command runs
([cli.go](../../cli/cli.go)). A malformed `.env` therefore makes `sip-creator check`
fail, although [ADR-0010](../decisions/0010-config-over-self-describing-input.md)
says check needs no configuration. Load `.env` and the config in `create` only.

### 1.2 Logs go to stderr

`newLogger` writes to `os.Stdout`. Anyone who pipes `create`, or reads the `OK: …`
line `check` prints to stdout from a script, gets log lines mixed in. Write logs to
`os.Stderr`.

### 1.3 One way to print violations

`check` prints one line per violation plus a count; `create` wraps the joined list in
`input folder … does not conform to the input specification:`. Add one
`reportViolations(cmd, src, err) error` and use it in both commands. This changes what
`create` prints for a bad folder.

### 1.4 Shorter `create`

`createCmd.RunE` resolves the profile, fills the submitter, pairs `--status` with
`--updates`, picks the content category, reads the folder, builds and zips. Pull out:

- `recordStatusFromFlags(cmd) (sip.RecordStatus, string, error)` for the pairing rule;
- `contentCategory(cmd) string` for the flag, then env, then profile order.

Construct the zipper only when `--no-zip` is not set. Rename `flagStatus` to
`statusText`.

### 1.5 Later, if wanted: no globals

`cfg`, `logger` and `rootCmd` are package globals and commands register in `init()`.
A `newRootCmd(cfg *config, logger *slog.Logger) *cobra.Command` would let tests run a
command in-process with their own config. Worth doing once tests for 1.4's flag rules
are wanted; not before.

## Phase 2: pure CSV parsers

### 2.1 Parse description.csv without `*directory`

`decodeDescription` ([description.go](../../cli/input/description.go)) handles CSV
syntax, calls the vocabulary, runs `Validate`/`ValidateRequired`, and maps errors back
to lines. `parseKey` takes a file name and line only to report through `d.violate`.
Target shape:

```go
func parseStatements(r io.Reader) ([]Statement, []error) // header, two columns, key[lang]
func parseKey(raw string) (key, lang string, err error)
```

`directory` opens the file (the UTF-8 and BOM rules in `openCSV` stay), calls the
parser, passes the statements to the vocabulary, and adds the file name to each error.

### 2.2 One error shape for a finding about a row

Vocabularies report `*StatementError{Line}`; the description's rules report
`*sip.TermError{Index}`, which the decoder turns into a line through
`statements[te.Index].Line` with a bounds check. `TermError` belongs to the library and
stays. Convert it in one helper next to the decoder (`lineOf(err, statements)`) so the
reporting loop handles one shape.

### 2.3 Parse representations.csv without `*directory`

Same treatment for `decodeRepresentations`
([representations.go](../../cli/input/representations.go)):

```go
func parseRepresentationRows(r io.Reader) ([]repRow, []error)
```

Matching rows to folders (`applyRepresentations`) stays on `directory`, because it
needs the folders read.

### 2.4 Tests

Move the syntax cases in `description_test.go` and `representations_test.go` to call
the parsers with a `strings.Reader`. Keep the folder-based tests that check how
problems are reported with the file name.

## Phase 3: less repetition in the folder walk

### 3.1 One table of reserved names

`read()` and `readRepresentation()` in [directory.go](../../cli/input/directory.go)
repeat the same "X is a folder; the reserved name is for …" checks. The two levels also
word the same problem differently: the top level prints the entry name, the
representation level prints `d.rel(src)` (which gives the same text at the top level).
Replace both with a table and one check:

```go
type reserved struct {
	dir   bool   // whether the name must be a folder
	holds string // what the name is reserved for, for the message
}

var reservedNames = map[string]reserved{ ... }
```

Representation folders use the subset of names that applies to them. Messages become
the same at both levels.

### 3.2 One way to collect content files

`collectFiles(dir)` is `walkContent(dir, dir, …)`; `readFlatRepresentation` and the
default branch of `readRepresentation` both loop over entries calling
`walkContent`/`newFile`. One `collectEntries(base string, entries []os.DirEntry)
[]build.SourceFile` covers all three.

### 3.3 One type for the supplied document

`Eark` and `EarkMods` each implement `DocumentName` and `CheckDocument` as one-line
calls into `Definition.DescriptiveName` and the encoder's
`DescriptiveDocumentChecker`. The reader taking no profile definition is deliberate
and stays. Replace the copies with one unexported type in `cli/input/vocabulary`
holding the definition, embedded in both vocabularies.

## Phase 4: names

One commit per file or concept, so each diff is easy to read.

| Now | Problem | New name |
|---|---|---|
| `directory` | A read in progress with collected problems, not a directory | `folder` |
| `d.violate(...)` | Reads as if the code is breaking a rule | `d.addViolation(...)` |
| `d.rel(p)` | Returns "the input folder" for the root: a display name, not a relative path | `d.display(p)` |
| `description, document, repsDir, repsCSV` in `read()` | Paths named like the things they point at | `descriptionPath`, `documentPath`, `representationsDir`, `representationsCSV` |
| `description(rows, document string, packageLevel bool)` | `rows` is a path; the bool is unclear at the call site | `rowsPath`, `documentPath`; a level type or two wrappers |
| `m := d.read()` in `Read` | Left over from an older name | `source` |
| `name := e.Name()` in `readRepresentation` | Shadows the `name` parameter | `entry := e.Name()` |
| `CheckDocument`, `checkDocument` | CLAUDE.md: a function returning an explanatory error is `Validate…` | `ValidateDocumentRoot` |
| `DocumentVocabulary` | Not a vocabulary; it says the profile accepts a finished document | `SuppliedDocument` |
| `decodeRepresentations`, `applyRepresentations` | Don't say they are about representations.csv | `parseRepresentationsCSV`, `orderByRepresentationsCSV` |
| `repRow.kind`, `repRow.dir` | `kind` stands in for `type`; `dir` is short | `typ`, `directory` |
| `placement.repeat cardinality`, `placement.lang` | `repeat` holds a count rule; `lang` is a yes/no | `occurs`, `takesLang` |
| `decodeSidecar`, `sidecarName` | The file is the Siegfried report | `decodeCharacterization`, `siegfriedName` |
| `def` in `create` and `resolveProfile` | Too short for a central value | `definition` |

Renaming the exported `DocumentVocabulary` and `CheckDocument` means updating
CLAUDE.md ("System shape", step 1) and `sip-creator-design.md` in the same commit.

## Phase 5: comments

Apply the CLAUDE.md rules ("comment the thing, not its callers"; as short as it can
be). Do this per file, together with or right after that file's Phase 4 renames.

- `Reader` ([package.go](../../cli/input/package.go)) talks about `check` and
  `create`. Proposed: *"Reader reads input folders under one profile's vocabulary and
  reports the library's own rules with file and line."*
- `checkCmd` ([check_cmd.go](../../cli/check_cmd.go)): seven lines down to two:
  *"checkCmd validates an input folder without building. Checks on file contents
  (received PREMIS, the characterization report) run only in create."*
- `decodeDescription`: the paragraph on who decides what goes away with Phase 2; one
  sentence per parser.
- `readDir`: keep the three rules (symbolic links, OS files, names that collide after
  NFC) and "os.ReadDir sorts by name, so traversal order is stable". Drop the mention
  of `scripts/reference-diff.sh`, which names a caller.
- `Vocabulary.Description` (9 lines), `DocumentVocabulary` (8) and `EarkMods` (11):
  describe only the contract, not what the reader does with it.
- "(collect-all)" in directory.go and representations.go: write "so later problems in
  the folder are still reported".
- `New validates nothing: Read refuses a nil vocabulary.`: remove the nil check (a nil
  interface panics on first use, which is fine for a programming error), or move it
  into `New` with an error return. Either way the comment goes.
- `vocabulary.go`: rewrap the overlong line in the `checkDocument` comment.
- `directory` struct: drop the sentence that repeats the `document` field's comment.

## Phase 6: small cleanups

- `newFile`: the fallbacks for a failed `filepath.Rel` cannot happen for paths under
  the root, and if they did they would put an absolute path in `Key`. Remove them.
- Use `errors.AsType` in `description.go`, as `check_cmd.go` does.
- Test files: `builder_test.go` tests that a Go-built source package matches the
  folder; rename it `source_package_test.go`. Move the `TestRepresentationCSV…` tests
  out of `description_test.go` into `representations_test.go`, and the one test in
  `directory_test.go` into `read_test.go`.

## Order

1. Phase 1.1 to 1.3: small and visible to operators.
2. Phase 2: the largest gain in readability and testing.
3. Phase 3.
4. Phases 4 and 5 together, file by file.
5. Phase 6 at any point.

Phase 1.4 can go anywhere; 1.5 only when someone wants tests for the CLI flags.

## When this plan ships

Update `sip-creator-design.md` and CLAUDE.md for the renamed exported names and the
parser functions. Consider a short ADR for keeping `cli/input` one package, since the
question is likely to come up again. Then move this plan to `docs/archive/`.
