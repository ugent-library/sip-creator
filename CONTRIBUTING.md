# Contributing to SIP Creator

This guide is for working on SIP Creator itself. For using the tool or the library, see
the [README](README.md).

## Requirements

* [Go](https://go.dev/dl/) 1.27 or later, for building and for `go test ./...`.
* For `build.sh` and the validation scripts:
  * `jq`;
  * [Docker](https://www.docker.com/), for the commons-ip validator and the report server;
  * `sf` ([Siegfried](https://github.com/richardlehane/siegfried)) on your `PATH`,
    because `build.sh` generates a `siegfried.json` for the input before building;
  * `xmllint` (part of libxml2, present on macOS and most Linux systems), which checks
    every `mods.xml` in an `eark-mods` package against the bundled MODS schema.

To run commons-ip on a local Java instead of in Docker, set `CSIP_CMD` to the command,
using the jar version that `docker/validator/Dockerfile` pins:

```sh
CSIP_CMD="java -jar commons-ip2-cli-2.11.2.jar" ./build.sh eark
```

## Building

```sh
go build -o bin/sip-creator .
./bin/sip-creator create --profile basic examples/basic basic-uuid
```

`bin/` and `<profile>-uuid/` are ignored by git.

## Tests

```sh
go test ./...
```

The Go tests need none of the tools above: no Docker, no `sf`, no `.env`. They include
a build of every folder in [examples/](examples/), so a change that makes an example
invalid fails the tests.

## Validating generated packages

`./build.sh [profile] [input]` is the local CI loop. The profile defaults to `basic` and
the input to [`examples/<profile>`](examples/); pass another input folder to validate
your own. It:

1. rebuilds the binary;
2. copies the input to `tmp/build/<profile>`, generates its `siegfried.json` there, and
   builds the package into `<profile>-uuid/`, so the input folder itself is never
   changed;
3. validates the zip with [commons-ip](https://github.com/keeps/commons-ip), printing
   every failed check with its messages;
4. runs `xmllint` over every `mods.xml` in the package against the MODS 3.7 schema the
   package ships, offline through the XML catalog in `scripts/schema-catalog.xml`,
   because commons-ip does not validate the descriptive documents the METS points at.

It exits non-zero when the package is not `VALID` or a `mods.xml` is not valid MODS. Each
profile validates against its E-ARK spec version: `basic` (Meemoo 1.2) against 2.0.4,
`eark` and `eark-mods` against 2.2.0. All three are expected to report `VALID`.

Each run's reports are written to `reports/runs/<timestamp>-<profile>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs):

```sh
docker compose up -d reports
open http://localhost:8080
```

`./scripts/validate.sh [-o report-dir] <sip.zip|sip-dir>...` validates any package on
its own, including an unzipped package directory when debugging structure.

## Comparing against the reference copy

```sh
./scripts/reference-diff.sh tmp/reference/<pkg> <profile>-uuid/uuid-<uuid>
```

compares a generated package with a reference copy kept in `tmp/reference/`, after
replacing the values that change on every run or per environment: UUIDs, timestamps,
the checksums and sizes of generated XML files, and the submitting organization. The
script's header lists what it normalizes and what it never does. A refactor must
produce no difference. When a change alters the output on purpose, replace the
reference copy and record why in the commit message and in `tmp/reference/README.md`.
`tmp/` is local and not tracked in git. The reference copies contain format info, so
build the package to compare with `build.sh`, which generates `siegfried.json`.

## Documentation

Update the docs in the same change as the code:

* a new or changed environment variable: run `go generate ./cli` to regenerate
  [CONFIG.md](CONFIG.md), which is never edited by hand;
* changed usage or input requirements: [README.md](README.md) and
  [docs/input-spec.md](docs/input-spec.md);
* changed library usage: the runnable examples in
  [build/example_test.go](build/example_test.go), which pkg.go.dev shows;
* a changed design: [docs/sip-creator-design.md](docs/sip-creator-design.md), and an ADR
  in [docs/decisions/](docs/decisions/) when the change records a decision;
* a renamed or removed exported symbol: a dated note under the Status line of every ADR
  whose decision text names it, giving the current name. ADRs are frozen, so the note is
  how a reader who greps for the old name finds the new one;
* a resolved item: remove it from [docs/TODO.md](docs/TODO.md).

[docs/README.md](docs/README.md) explains which document answers which question.

## Commits

Start the message with the kind of change, capitalized and followed by a colon:
`Added:`, `Changed:`, `Fixed:` or `Removed:`. Keep commits small and focused. Never
commit `.env`, `tmp/` or generated packages.
