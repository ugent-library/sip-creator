# Plan: the check command reports what a folder holds and every problem in it

*Status: **in progress** (drafted 2026-10-06). Step 1 landed in 2cf8b79; step 2 follows.
The plan builds on 4c93b57; line numbers refer to that tree. Update this line as steps
land.*

## Context

`check --profile <name> <src>` validates an input folder against the
[input specification](../input-spec.md) and the profile's rules. For a valid folder it
prints one line:

```
OK: 1 representation(s), 1 content file(s), 1 documentation file(s)
```

That line tells the operator neither what the tool read from the folder nor which
defaults it will apply. Four further gaps:

- **Check reads nothing inside received PREMIS files.** A file under `premis/` that is
  not even well-formed XML passes check and fails create, which checks it during
  assembly (`build/assemble.go:271`). For the other supplied files check already reads
  the contents: `siegfried.json` must be a siegfried report
  (`characterization/characterization.go:71`), and `dc.xml` and `mods.xml` must be
  well-formed XML with the profile's root element (`cli/input/decode.go:228`).
- **Check passes a `siegfried.json` that create refuses.** When a report is supplied,
  create requires an entry for every content file (`build/assemble.go:301`). Check
  decodes the report but does not look up the files in it. A report generated from the
  wrong folder, or before files were added, therefore passes check and fails create.
- **A `dc+schema.xml` in a basic folder gets no clear message.** The basic profile takes
  no supplied document, because its document must carry the identifier the build mints.
  Beside a `representations/` folder, check reports the file as content in the wrong
  place. In a flat folder, check accepts it as a second essence file without a message.
- **Findings arrive in two shapes.** Input violations print one per line on stderr
  (`cli/cli.go:37`). An error from `Definition.ValidateSource` is returned as it is, so its
  joined lines print through cobra's error output with a different prefix.

Decided in chat on 2026-10-06:

1. The check is about the input rules of this tool, the input specification and the
   profile's rules. It is not a CSIP validator. CSIP and XSD conformance of the written
   package stays with the commons-ip validation in `build.sh`
   ([ADR-0003](../decisions/0003-validation-stays-external.md)).
2. Check confirms that a received PREMIS file is well-formed XML, and nothing else. It
   does not compare checksums: whether `siegfried.json` matches the files on disk stays
   a check of create, which hashes the files anyway.
3. The basic profile takes no supplied descriptive document. A `dc+schema.xml` in a
   basic folder is an error with a message that says so.
4. The audience is the operator who prepares a folder and builds SIPs from it.
5. The report shows findings and an inventory of the folder. It does not list every rule
   with a pass status.
6. Findings keep their current wording. They do not refer to sections of the input
   specification.
7. The inventory shows file sizes. A list of files stops after 25 entries, with an
   ellipsis line that says how many more there are. A `--full` flag lists every file.
8. Descriptive values longer than a maximum length are cut off with an ellipsis.
9. The inventory has a section for optional inputs the folder does not supply. Its
   heading makes clear that these are optional.
10. The inventory lists the files the reader skipped, by their path in the folder.
11. The inventory opens with the total size of the folder's files, in human-readable
    units, and the number of files.
12. The profile's rules run only on a folder the reader read without violations.
13. Check exits with status 1 when the folder has problems and 2 when it cannot check the
    folder at all.
14. Test folders for the report live under `cli/testdata/`, the Go convention.
15. When a folder supplies `siegfried.json`, check reports each content file that has no
    entry in it as an error, as create does. This needs no hashing.
16. The report is an unexported type `checkReport` in `cli/`. The input reader hands it
    the facts the source package lacks in a separate small value, `input.ReadDetails`.
    The walker's internal `inventory` record stays private and unchanged.
17. Steps 1 to 3 close gaps in the input rules and come first, each as its own step.

`go test ./...` passes after every step. No step changes an output file, so
`./scripts/reference-diff.sh` stays clean throughout.

**Relation to [profile-rules-and-names.md](profile-rules-and-names.md).** That plan is in
progress. It renames the profiles (`meemoo/basic`, `eark/none`, `eark/dc`, `eark/mods`),
adds a profile without descriptive metadata and allows zero representations under
`eark/dc` and `eark/mods`. The inventory in step 5 must therefore handle a package without
a description and a package without representations. Whichever plan lands second updates
the example output in this plan and in the input specification.

## Steps

### Step 1. Check confirms received PREMIS files are well-formed XML

`Fixed: check reports a file under premis/ that is not well-formed XML`

- `cli/input/walker.go` `collectPremisFiles`: open each collected file and read it with
  `xmldoc.Root`, the one XML reader. A failure is a violation that names the file, such
  as `premis/received.xml: not well-formed XML ...`. Nothing else in the file is
  checked: not the root element, not the namespace.
- `build/assemble.go:271` keeps its `premis:premis` root check unchanged. Create still
  refuses a well-formed file that is not PREMIS, as it does today.
- Tests: `cli/input/walker_test.go`, a malformed file under package-level and
  representation-level `premis/` each give a violation, and a well-formed non-PREMIS
  file gives none. `cli/check_cmd_test.go`, a folder with a malformed PREMIS file fails
  check.
- Docs: `sip-creator-design.md` line 116 says check confirms received PREMIS files are
  well-formed and leaves the root element and the characterization report's checksums
  to create. `input-spec.md` section 5 says the same for the operator.

### Step 2. The basic profile refuses a supplied descriptive document

`Fixed: check and create refuse a dc+schema.xml in a basic folder with a message that basic takes no supplied document`

- `cli/input/package.go` `Read` sets the reader's document name only when the profile's
  model takes a document. The basic definition already declares `dc+schema.xml` as its
  `DocumentName`. The reader also keeps that name when the model takes no document, and
  reports a file with it, at package or representation level, as a violation:
  `dc+schema.xml: profile basic takes no supplied descriptive document; describe the
  package in description.csv`. The rule follows from the definition's data, not from the
  profile's name.
- The file is reported in a flat folder and beside a `representations/` folder alike, and
  is never taken as content.
- Tests: `cli/input/read_test.go`, a flat and a structured basic folder with
  `dc+schema.xml` each give that one violation.
- Docs: `input-spec.md` section 3, under supplying a finished document, states the rule.

### Step 3. Check requires an entry in `siegfried.json` for every content file

`Fixed: check reports content files that have no entry in a supplied siegfried.json, as create does`

- `cli/input/decode.go`: after the report decodes, look up each content file of each
  representation by its key, the path relative to the input folder. A file without an
  entry is a violation that names the file and says how to fix it:
  `siegfried.json: no entry for representations/master/image-001.jpg; regenerate it from
  the input root with: sf -hash md5 -json .` When every lookup fails, one example key
  from the report is added to the first message, as create's message does
  (`build/assemble.go:303`), because the usual cause is a report made from another
  folder.
- Check does not open or hash the files. Whether an entry's MD5 matches the file stays
  with create.
- Documentation files need no entry, as in create. Entries for files the package does not
  carry are ignored, as in create.
- `build/assemble.go` keeps its own lookup unchanged, because a program that builds a
  source package in Go never passes through the input reader. A comment at the new check
  in `cli/input` names that rule in `build` as the one it mirrors.
- Tests: `cli/input/read_test.go`, a folder whose report lacks one content file gives one
  violation naming it; a report with every key prefixed by another folder name gives a
  violation per file and an example key; a report that lacks a documentation file gives
  none. `cli/check_cmd_test.go`, such a folder fails check.
- Docs: `input-spec.md`, section 2 under `siegfried.json`, states that every content file
  needs an entry. `sip-creator-design.md` line 116 says check verifies the entries exist
  and create verifies their checksums.

### Step 4. One list of findings

`Changed: check prints every finding from the folder and the profile as one list`

- Collect findings from both sources, input violations and `ValidateSource`, into one
  list of plain lines. Joined errors are split into their parts, as `flatten` in
  `cli/input/description.go:96` already does for descriptions.
- `ValidateSource` runs only when `input.Read` returned no violations. On a partly read
  folder it would give misleading findings, such as a missing description that is only
  missing because its CSV failed to parse.
- Layout on failure. The two messages are illustrative; the real ones keep their
  current wording:

  ```
  2 problems in examples/basic

    description.csv, line 7: "created" must be a date in YYYY-MM-DD form, got "15/01/2026"
    premis/received.xml: not well-formed XML: unexpected EOF

  FAILED: fix the problems above and run check again.
  ```

- The report goes to stdout, so an operator can redirect it to a file. Cobra's final
  error line on stderr stays short: `examples/basic: 2 problem(s) found`.
- Exit statuses of check:

  | status | meaning |
  |---|---|
  | 0 | the folder meets the input specification and the profile's rules |
  | 1 | the folder has problems; the report lists them |
  | 2 | check could not check the folder: a wrong path, a file given instead of a folder, an unknown profile, a usage error |

  `cli.Run` exits through `cobra.CheckErr` today, which always exits with 1
  (`cli/cli.go:53`). Check returns a distinct error type for "problems found", and
  `Run` maps that type to 1 and every other error to 2.
- `create` keeps its current error output and exits with 1 on every failure, as today.
- Docs: `README.md` lists the exit statuses of check.

### Step 5. The inventory

`Added: check lists what the input folder holds: descriptive metadata, representations, files with their sizes, and the optional inputs it does not supply`

Layout for a valid folder:

```
Input folder: examples/basic
Profile:      basic
Contents:     4 files, 243.9 kB

Descriptive metadata (description.csv)
  identifier        example-0001
  title[nl]         Voorbeeldfoto
  title[en]         Example photograph
  description[nl]   Een voorbeeldpakket met één afbeelding.
  description[en]   An example package with one image.
  created           2026-01-15
  creator           Example Studio

Representations
  master   label "master", type "master"
    Content files: 1, 240 kB
      image-001.jpg                     240 kB   fmt/43   image/jpeg
    Documentation: 1 file
      capture-notes.txt                 1.2 kB

Package documentation: 1 file
  README.txt                            2.4 kB

Optional inputs not supplied
  representations.csv   labels and types are the folder names
  siegfried.json        files carry no format information
  premis/               no received preservation files

Skipped
  .DS_Store
  representations/master/.DS_Store

OK: the folder meets the input specification for profile basic.
```

Rules for the content:

- **Descriptive metadata** lists the rows of `description.csv` as the producer wrote
  them: key with its language tag, then the value. A value longer than 60 characters is
  cut at 60 and ends in `…`. Line breaks inside a value print as spaces. A supplied
  document (`dc.xml`, `mods.xml`) prints as its file name and size, without its
  contents. A representation-level description prints under its representation in the
  same form.
- **Files** print with their size in decimal units (B, kB, MB, GB), as macOS Finder
  shows them. With a `siegfried.json`, each content file also shows the PRONOM
  identifier and MIME type its entry records. The inventory reads these from the
  report; it does not compare checksums.
- **Long lists** stop after 25 files with a line `… and 1,975 more files (use --full to
  list all)`. The limit applies per list: content, documentation, received PREMIS.
- **Optional inputs not supplied** names each optional input the folder lacks and the
  default the tool applies instead. The candidates are `representations.csv`,
  `siegfried.json`, a package-level `documentation/` and `premis/`.
- **Contents** counts every file the package will take from the folder: content,
  documentation, received PREMIS, `description.csv` and a supplied document. It does not
  count `siegfried.json` or `representations.csv`, which the package does not carry.
  The size uses the same units as the file lines.
- **Skipped** lists, by path relative to the input folder, each file the reader leaves
  out without a violation: the operating system files `.DS_Store`, `Thumbs.db`,
  `desktop.ini` and names starting with `._` (`cli/input/walker.go:373`). The section is
  left out when nothing was skipped.
- On a failed check the inventory follows the findings, built from the partial source
  package that `input.Read` returns. Parts the reader could not read are left out.

Implementation:

- The report needs three facts the source package does not carry. `input.Read` returns
  them next to the source package as a new exported type, `ReadDetails`, documented
  field by field:

  ```go
  // ReadDetails holds what a read of an input folder found beyond the source
  // package: the descriptive rows as the producer wrote them, which optional
  // files the folder supplied, and what the reader left out. It describes the
  // folder for an operator; the build does not use it.
  type ReadDetails struct {
      // PackageRows are the rows of the package-level description.csv in file
      // order, with Key as the row spells it, before the profile's mapping.
      // Empty when the package level has no description.csv.
      PackageRows []sip.Term
      // RepresentationRows are the rows of each representation's
      // description.csv, by representation name. A representation without
      // one has no entry.
      RepresentationRows map[string][]sip.Term
      // RepresentationsCSV reports whether the folder supplies
      // representations.csv.
      RepresentationsCSV bool
      // Skipped lists the files the reader left out without a violation,
      // operating system files such as .DS_Store, by slash path relative to
      // the input folder, in walk order.
      Skipped []string
  }
  ```

  The signature becomes
  `func Read(root string, mapper Mapper, documentSpec DocumentSpec) (*build.SourcePackage, ReadDetails, error)`.
  `create` ignores the details. Whether `siegfried.json`, a package-level
  `documentation/` or `premis/` was supplied follows from the source package, so
  `ReadDetails` does not repeat it. The walker's internal `inventory` record stays
  private and unchanged. `readDir` records each skipped file where it now drops it. The
  fields may still change while the step is written; the doc comments are the contract.
- The report is an unexported type `checkReport` in its own file in `cli/`, next to
  `check_cmd.go`. It holds the findings and the inventory, is built from the source
  package and the `ReadDetails`, and prints itself. Nothing moves into `build`, so the
  library API is unchanged.
- `--full` is a flag on `check` only.
- Test folder `cli/testdata/full/`: a folder with every optional input, which the
  examples leave out: `representations.csv`, `siegfried.json` with entries for its
  files, package-level and representation-level `premis/` and `documentation/`, and a
  `.DS_Store`. The go command ignores `testdata` folders when it builds and lists
  packages, and a test runs in its package folder, so the test reads it as
  `testdata/full`. The examples stay minimal.
- Tests in `cli/check_cmd_test.go`: the three examples print an inventory that names
  their files; `cli/testdata/full` prints formats, received PREMIS files, the totals
  and the skipped `.DS_Store`; a folder with 30 content files prints 25 and the
  ellipsis line, and all 30 with `--full`; a long title is cut with `…`; a folder without
  `siegfried.json` lists it under optional inputs. Folders built in a test, such as the
  one with 30 files, stay in `t.TempDir()`.
- Docs: `README.md` usage section shows a report. `sip-creator-design.md` line 116
  describes the report. `input-spec.md` mentions that check lists the folder's
  contents.

### Step 6. Examples and wrap-up

`Changed: the examples' README shows the check report for each example`

- `examples/README.md`: the command and the report for one example.
- Move this plan to `archive/`. Whether the report layout deserves an ADR is decided
  then; it probably does not, because the layout is CLI output, not a library contract.

## Open questions

None. Decisions made while writing a step go into this plan before the step's commit.
