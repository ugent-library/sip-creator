# Contributing to SIP Creator

This guide is for working on SIP Creator itself. For using the tool or the library, see
the [README](README.md).

## Requirements

* [Go](https://go.dev/dl/) 1.27 or later, for building and for `go test ./...`.
* For `build.sh` and the validation scripts:
  * `jq`;
  * [Docker](https://www.docker.com/), for the commons-ip validator and the report server;
  * `sf` ([Siegfried](https://github.com/richardlehane/siegfried)) on your `PATH`,
    because `build.sh` regenerates the input fixture's `siegfried.json` before building;
  * `xmllint` (part of libxml2, present on macOS and most Linux systems), which checks
    every `mods.xml` in an `eark-mods` package against the bundled MODS schema.

To run commons-ip on a local Java instead of in Docker, set `CSIP_CMD` to the command,
using the jar version that `docker/validator/Dockerfile` pins:

```sh
CSIP_CMD="java -jar commons-ip2-cli-2.11.2.jar" ./build.sh eark
```

## Tests

```sh
go test ./...
```

The Go tests need none of the tools above: no Docker, no `sf`, no `.env`.

## Validating generated packages

`./build.sh [profile]` (default `basic`) is the local CI loop. It:

1. rebuilds the binary;
2. regenerates the sample SIP from `tmp/<profile>`;
3. validates the zip with [commons-ip](https://github.com/keeps/commons-ip), printing
   every failed check with its messages;
4. runs `xmllint` over every `mods.xml` in the package against the MODS 3.7 schema the
   package ships, offline through the XML catalog in `scripts/schema-catalog.xml`,
   because commons-ip does not validate the descriptive documents the METS points at.

It exits non-zero when the package is not `VALID` or a `mods.xml` is not valid MODS. Each
profile validates against its E-ARK spec version: `basic` (meemoo 1.2) against 2.0.4,
`eark` and `eark-mods` against 2.2.0. All three are expected to report `VALID`.

Each run's reports are written to `reports/runs/<timestamp>-<profile>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs):

```sh
docker compose up -d reports
open http://localhost:8080
```

`./scripts/validate.sh <sip.zip|sip-dir>...` validates any package on its own, including
an unzipped package directory when debugging structure.

## Documentation

Update the docs in the same change as the code:

* a new or changed environment variable: run `go generate ./cli` to regenerate
  [CONFIG.md](CONFIG.md), which is never edited by hand;
* changed usage or input requirements: [README.md](README.md) and
  [docs/input-spec.md](docs/input-spec.md);
* a changed design: [docs/sip-creator-design.md](docs/sip-creator-design.md), and an ADR
  in [docs/decisions/](docs/decisions/) when the change records a decision;
* a resolved item: remove it from [docs/TODO.md](docs/TODO.md).

[docs/README.md](docs/README.md) explains which document answers which question.

## Commits

Start the message with the kind of change, capitalized and followed by a colon:
`Added:`, `Changed:`, `Fixed:` or `Removed:`. Keep commits small and focused. Never
commit `.env`, `tmp/` or generated packages.
