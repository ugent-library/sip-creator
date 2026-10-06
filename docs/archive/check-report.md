# Plan: the check command reports what a folder holds and every problem in it

*Status: **shipped** (2026-10-06). Steps 1 to 4 landed in 2cf8b79, 9977f7a, 6e07de0 and
610692e; step 5 in b7b1869 and 6eeb105, after the inventory was dropped for counts; step 6
in the commit that moved this plan here. No ADR: the report is CLI output, not a library
contract. The check command and its report are described in the
[design doc](../sip-creator-design.md); the input rules it added are in the
[input specification](../input-spec.md). Exit statuses for `create` stay open in
[TODO.md](../TODO.md).*

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
5. The report shows findings and a summary of the folder in counts. It does not list
   every rule with a pass status, and it does not list files.
6. Findings keep their current wording. They do not refer to sections of the input
   specification.
7. The summary names where the package description comes from (`description.csv`, a
   supplied document, or none), and counts representations, those with their own
   description, essence files, documentation files and PREMIS files across all levels.
   It says whether a format report was supplied. It shows no sizes, no file lists and no
   skipped files.
8. The profile's rules run only on a folder the reader read without violations.
9. Check exits with status 1 when the folder has problems and 2 when it cannot check the
   folder at all.
10. When a folder supplies `siegfried.json`, check reports each content file that has no
    entry in it as an error, as create does. This needs no hashing.
11. The report is an unexported type `checkReport` in `cli/`, built from the source
    package alone. `input.Read` keeps its signature.
12. Steps 1 to 3 close gaps in the input rules and come first, each as its own step.

Revised on 2026-10-06 after a first build of step 5: an inventory with file lists,
sizes, formats, the descriptive rows, skipped files and a `--full` flag, fed by a new
`input.ReadDetails`, was built and dropped. A check reports violations; listing a
folder's contents is work for other tools, and a list cut at 25 files with a flag for
the rest made the report harder to read, not easier. `ReadDetails` (b7b1869) is
removed again in step 5's commit.

`go test ./...` passes after every step. No step changes an output file, so
`./scripts/reference-diff.sh` stays clean throughout.

**Relation to [profile-rules-and-names.md](../plans/profile-rules-and-names.md).** That plan is in
progress. It renames the profiles (`meemoo/basic`, `eark/none`, `eark/dc`, `eark/mods`),
adds a profile without descriptive metadata and allows zero representations under
`eark/dc` and `eark/mods`. The summary in step 5 must therefore handle a package without
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
  `siegfried.json has no entry for representations/master/image-001.jpg; regenerate it
  from the input root with: sf -hash md5 -json .` When no content file has an entry, one
  line with an example path from the report replaces the line per file, because the usual
  cause is a report made from another folder, and a folder of thousands of files would
  otherwise give thousands of lines. An empty report gets its own line.
- Check does not open or hash the files. Whether an entry's MD5 matches the file stays
  with create.
- Documentation files need no entry, as in create. Entries for files the package does not
  carry are ignored, as in create.
- `build/assemble.go` keeps its own lookup unchanged, because a program that builds a
  source package in Go never passes through the input reader. A comment at the new check
  in `cli/input` names that rule in `build` as the one it mirrors.
- Tests: `cli/input/read_test.go`, a folder whose report lacks one content file gives one
  violation naming it; a report with every key prefixed by another folder name gives one
  violation with an example path; an empty report gives one violation; a report that
  lacks the documentation files gives none. `cli/check_cmd_test.go`, such a folder fails check.
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

### Step 5. The summary

`Added: check summarizes what the input folder holds in counts, below the problems and above the verdict`

Layout for the eark example:

```
Input folder:         examples/eark
Profile:              eark

Descriptive metadata: description.csv
Representations:      1 (1 with its own description)
Essence files:        1
Documentation files:  2
PREMIS files:         2
Format report:        not supplied (files carry no format information)

OK: the folder meets the input specification for profile eark.
```

- **Descriptive metadata** is `description.csv` when the package description comes from
  rows, which the build turns into a generated document; the document's name with
  `(supplied document, copied as it is)` for a `dc.xml` or `mods.xml`; `none` when the
  reader found no package description.
- **Representations** adds `(n with its own description)` when any representation has
  one.
- **Essence, documentation and PREMIS files** are totals over the package and all
  representations. PREMIS files are the received ones; the generated documents are not
  in the folder.
- **Format report** is `siegfried.json` or `not supplied (files carry no format
  information)`.
- On failure the problems come first, the summary of what the reader got follows, and
  the verdict reads "fix the problems listed at the top".
- Implementation: `checkReport` in `cli/check_report.go` gains the source package and
  prints the summary. `input.ReadDetails` and the reader's record of skipped files are
  removed, and `input.Read` returns to two values.
- Tests in `cli/check_cmd_test.go`: the eark example's report exactly; a failing folder's
  report exactly, with the summary of what was read; a supplied `dc.xml` and a supplied
  `siegfried.json` named in the summary.
- Docs: `README.md` shows a report. `sip-creator-design.md` describes the summary.
  `input-spec.md` says check counts what the folder holds.

### Step 6. Examples and wrap-up

`Changed: the examples' README shows the check report for each example`

- `examples/README.md`: the command and the report for one example.
- Move this plan to `archive/`. Whether the report layout deserves an ADR is decided
  then; it probably does not, because the layout is CLI output, not a library contract.

## Open questions

None. Decisions made while writing a step go into this plan before the step's commit.
