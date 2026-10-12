---
name: validate-profiles
description: >-
  Build the example package of every registered profile with build.sh, validate it
  with commons-ip and xmllint, compare it with the reference copy in tmp/reference/,
  and report the result per profile. Use when the user asks to validate all profiles,
  check that every profile is still valid, or run the checks a change must pass.
---

# Validate profiles

Run the checks a change must pass for every registered profile, and report what
failed in plain language. The Validating generated packages section of
CONTRIBUTING.md describes what `build.sh` does and what each check covers. Read it
first.

This skill changes no files. If a check fails, report it and leave the fix to the
user's next request.

## Steps

1. Run `go test ./...`. If it fails, report the failing tests and stop: a package
   built from failing code says nothing.
2. List the registered profiles. `go run . create --help` prints them in the
   description of the `--profile` flag.
3. For each profile, run `./build.sh <profile>` on its own and keep its exit status.
   A non-zero exit means the package is not `VALID` or a descriptive document failed
   its schema. Exit status 2 means the run did not get that far: a missing tool, an
   unknown profile or a missing input folder.
4. Read each run's output. Most lines are the builder's `level=INFO` log. The lines
   that matter:
   - `== <zip>: VALID (errors=… warnings=…)` or `INVALID`: the commons-ip result.
   - `FAILED [MUST|SHOULD|MAY] <check id>: <name>` and the messages under it: each
     failed CSIP check.
   - `<file> validates` or `<file> fails to validate`, with the xmllint errors
     above it: the descriptive documents.
   - `report: reports/runs/<run>`: the folder with the full report.

   The xmllint lines "Schemas parser warning … Skipping import of schema located at
   'http://www.w3.org/2001/03/xml.xsd'" come from the schemas the package ships and
   are not a result. Ignore them.
5. For each profile with a reference copy in `tmp/reference/<profile>/pkg`, run
   `./scripts/reference-diff.sh tmp/reference/<profile>/pkg tmp/output/<profile>/uuid-*/`.
   A profile without a reference copy gets "no reference copy" in the report. Don't decide whether a difference is intended. Show it, and
   let the user decide.

## Expected results

Every profile's example package reports `VALID` with `errors=0 warnings=0`. Any
failed check is a finding, at any level. The exception: a package without
representations under a UGent profile reports two SHOULD-level warnings, CSIPSTR11
and CSIPSTR13, which docs/TODO.md explains. The examples all have representations,
so these warnings appear only when the user passed another input folder.

## Report

Report in chat, one line per profile, then the details of what failed:

- the commons-ip result with its error and warning counts
- whether every descriptive document validated
- whether the reference copy matched, differed, or does not exist
- the report folder under `reports/runs/`

For a failure, quote the failed check's ID and messages, or the xmllint error, and
name the file. Don't guess at a cause you haven't checked in the code.
