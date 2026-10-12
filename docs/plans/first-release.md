# Plan: the first release, v0.1.0

*Status: **drafted** 2026-10-12. The decisions below were taken in chat on 2026-10-12.
Each step starts on an explicit go in chat. Update this line as steps land, and move the
plan to `archive/` once the tag exists.*

## Context

### Where the project stands

The code is ready for a first version and the repository is not. `main` holds 360 commits
and no tag. `go test ./...`, `go vet ./...` and `gofmt -l .` pass. The example of every
profile reports VALID with zero warnings under commons-ip 2.11.2 ([TODO](../TODO.md),
validator status of 2026-10-07).

[TODO](../TODO.md) (parked 2026-10-06) and
[ADR-0033](../decisions/0033-ugent-first-profiles-of-your-own.md) hold a changelog, a
first tag and CI until UGent's own use calls for them, such as a version for a program
that embeds the library to pin. That use is here.

### What is already in place

- `build/software.go` writes the module version the Go toolchain stamps into the binary
  as the `SOFTWARE VERSION` note of the METS software agent. A tag `v0.1.0` makes every
  package built by `go install ...@v0.1.0` state `v0.1.0`. A build from a clone at the tag
  states the same, with `+dirty` when the tree had uncommitted changes.
- `scripts/reference-diff.sh` masks that note, so a tagged build compares clean against
  the reference copy.
- The README carries the pkg.go.dev badge, the `go install` instructions and the
  experimental warning. The repository is public, so GitHub Actions runs at no cost.

### What is missing

A changelog, a tag, CI, a way for an operator to ask the binary its version, and the docs
that say what a version means.

## Decisions (taken 2026-10-12)

1. **v0.1.0 ships `meemoo/basic` at Meemoo SIP 1.2**, as it validates today. The move to
   2.1 ([TODO](../TODO.md)) stays in the backlog and lands in a later minor version, which
   drops 1.2.
2. **Distribution is `go install`.** No prebuilt binaries and no goreleaser. A release
   pipeline and unsigned binaries are more than a handful of UGent staff need, and they
   can run `go install`.
3. **The first real ingest into UGent's RODA runs before the tag.** It is the acceptance
   test the backlog names under "Waiting on the operator".
4. **Semantic versioning at major version 0.** A minor version may change the command
   line, the library API or a profile's output. A patch version fixes a defect. The tag is
   the version the software agent states.

ADR-0036 records decisions 2 and 4 (step 5).

## Steps

Each step is one or more small commits on `main`. The commit message is proposed in chat
before each commit. No file names a downstream system. Write "a program that embeds the
library".

### Step 1. Checks and corrections

Checks: `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .`, `go mod tidy` with no
diff, `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (a one-off run, not a
dependency), and `./build.sh` for every profile with `scripts/reference-diff.sh` clean.

Corrections:

- **The Meemoo spec status.** `CLAUDE.md` (first paragraph) and
  [sip-creator-design.md](../sip-creator-design.md) (profile table) say "the stable
  version; 2.0 and 2.1 are release candidates". The TODO entry of 2026-10-08 records that
  2.0 is stable and 2.1 a release candidate. Both follow the TODO.
- **`.gitignore`.** `.rumdl_cache/` at the repository root is not listed. Add it if git
  would track it.
- **The empty `sip.Event`** (`sip/event.go`): an exported type with no fields and no user
  in the module. A tag commits to every exported symbol. Decided when this step starts:
  remove it, or keep it as the place the events design starts from. Removing it changes
  the sentences that name it in [sip-creator-design.md](../sip-creator-design.md) (Known
  gaps), [TODO.md](../TODO.md) (the events item) and
  [development-writing-style.md](../development-writing-style.md) (its TODO example).

### Step 2. `sip-creator --version`

An operator and a script need to know which version runs, and it must be the version the
METS states. Set `Version` on the root command in `cli/cli.go`. cobra then provides
`--version`. The value comes from the function the software agent uses, `softwareVersion`
in `build/software.go`, exported because `cli` becomes its second caller and the two must
not drift. The name is discussed in chat before the change. The README (Installation)
gets a sentence on `--version`. The design doc's sentence on the software agent gains
that the CLI reports the same value.

### Step 3. CI

`.github/workflows/test.yml`, on push to `main` and on pull requests: checkout, `setup-go`
with `go-version-file: go.mod`, a `gofmt -l .` check that fails on output, `go vet ./...`,
`go test ./...`. No Java, `sf` or `.env`. CONTRIBUTING.md (Tests) says the workflow runs
the same checks on every push.

A job that runs `build.sh` is left out. It needs Java 17, `sf` with its signature file and
the commons-ip jar. It goes to TODO as a follow-up with that list.

### Step 4. `CHANGELOG.md`

[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format, with `Unreleased` and
then `0.1.0` dated at the tag. The commit prefixes `Added:`, `Changed:`, `Fixed:` and
`Removed:` are its headings, so later entries come from the log.

The 0.1.0 entry says what the version is, not what 360 commits did:

- the `create` and `check` commands and the input folder
  ([input-spec.md](../input-spec.md));
- the profiles and their targets: `meemoo/basic` at Meemoo SIP 1.2, validated against
  E-ARK 2.0.4, and `ugent/basic` and `ugent/bibliographic` at E-ARK SIP 2.2.0, the latter
  with MODS 3.7;
- the library: `build`, `profiles`, `archive`, `characterization`, and profiles of your
  own;
- the known gaps, from TODO and the design doc:
  - EDTF dates under `meemoo/basic` are not checked;
  - the ARCHIVIST agent, contact persons and the submission agreement are not written;
  - PREMIS events are not modeled;
  - essence must be files on disk, and fixity cannot be supplied;
  - the final rename can fail on Windows;
  - `meemoo/basic` moves to Meemoo 2.1 in a later version;
  - no package has been ingested by Meemoo, only validated with commons-ip.

CONTRIBUTING.md (Documentation) gains: a change an operator or a program notices gets a
line under `Unreleased`.

### Step 5. What a version means

- **README.** The experimental warning stays. Installation keeps `go install ...@latest`,
  which resolves to the newest tag once one exists, and gains a short paragraph on
  versions: decision 4, the changelog, and that every package's METS states the version
  that built it.
- **CONTRIBUTING.md.** A "Releasing" section with the procedure of step 7.
- **ADR-0036, versioned releases.** One page. Context: ADR-0033's condition is met.
  Decision: 2 and 4, and what triggers a release. Alternatives rejected: v1.0.0, while
  streams, fixity, events and Meemoo 2.1 still change the API and the output; prebuilt
  binaries; waiting for Meemoo 2.1.
- **CLAUDE.md.** The Git section gains the tag rule (annotated `vX.Y.Z`, by the procedure
  in CONTRIBUTING.md). The docs list gains CHANGELOG.md.
- **TODO.** The parked item of 2026-10-06 keeps the input-reader move only. The
  validation CI job from step 3 is added.

### Step 6. The RODA ingest

Run by the operator. Build a `ugent/basic` and a `ugent/bibliographic` package with
`build.sh` from the commit that ends step 5. Ingest them as
[development-roda-ingest.md](../development-roda-ingest.md) describes, verify what
arrived, and record the outcome in the runbook's table, which is a desk-check today. What
RODA rejects is fixed and validated before the tag. The TODO item under "Waiting on the
operator" closes when the outcome is recorded.

### Step 7. Tag and publish

Each command runs on an explicit go.

1. Move `Unreleased` to `0.1.0 - <date>` and commit.
2. `git tag -a v0.1.0 -m "SIP Creator v0.1.0"`, then `git push origin main v0.1.0`.
3. `gh release create v0.1.0 --title v0.1.0 --notes-file <the 0.1.0 section>`.
4. This plan's status line records the tag, and the plan moves to `archive/`.

## Verification

- Steps 1 to 3: the three Go checks pass locally and in the Actions run. Every profile
  reports VALID in build.sh with no reference diff. `./bin/sip-creator --version` prints
  a pseudo-version with the commit.
- After the tag, with a temporary `GOPATH` and `GOMODCACHE`:
  `go install github.com/ugent-library/sip-creator@v0.1.0`, then `sip-creator --version`
  prints `v0.1.0`. A package built from `examples/ugent/basic` carries `v0.1.0` in the
  `SOFTWARE VERSION` note of its METS.
- `go list -m -versions github.com/ugent-library/sip-creator` lists `v0.1.0` through
  proxy.golang.org (requesting the pkg.go.dev page once triggers the fetch). The
  pkg.go.dev page for `build` shows the five examples of `build/example_test.go`.

## Open questions

- Whether `sip.Event` is removed (step 1).
- The exported name of the version function (step 2).

## Stays in the backlog

`meemoo/basic` to Meemoo 2.1 (the first candidate for v0.2.0), streams and pre-computed
fixity, PREMIS events, the administrative values of ADR-0010, EDTF checking, the
input-reader move, `create` exit statuses, the validation CI job. Prebuilt binaries are
decided against (decision 2), not deferred.
