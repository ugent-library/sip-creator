[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)

# SIP Creator

Create Submission Information Packages (SIP) based on [Meemoo's SIP Specification](https://developer.meemoo.be/docs/diginstroom/sip/)

:warning: **This is an experimental package** :warning:

## Features

* Implements [Meemoo SIP Specification v2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/).
* Supports the `basic` profile.
* Supports file format characterization via [Siegfried](https://github.com/richardlehane/siegfried).

## Requisites

* [Siegfried](https://github.com/richardlehane/siegfried) is installed and working on your system.
* For the validation loop: [Docker](https://www.docker.com/) (runs the commons-ip validator and the report server) and `jq`.

## Configuration

Create a `.env` file (start from `.env.example`) if non exists. All environment variables are documented in [CONFIG.md](CONFIG.md), which is generated from the config struct — regenerate it with
`go generate ./services` rather than editing it by hand.

**Siegfried**

Configure Siegfried like this:

```
SIP_FILE_FORMAT_NAME="siegfried"
SIP_FILE_FORMAT_COMMAND="/location-of-sf-binary"
SIP_FILE_FORMAT_ARGS="-hash md5 -json"
```

## How to use

Assuming you have data in a `./tmp/basic` directory which you want to convert into a SIP package
stored in a `basic-uuid` directory:

```
go build -o bin/sip-creator .
./bin/sip-creator create --profile basic ./tmp/basic basic-uuid
```

This writes the package directory `basic-uuid/uuid-<uuid>/` and zips it (uncompressed) to
`basic-uuid/uuid-<uuid>.zip` — the zip is the deliverable meemoo ingests.

## Validation

`./build.sh` is the local CI loop: it rebuilds, regenerates the sample SIP, validates the
zip against E-ARK CSIP with [commons-ip](https://github.com/keeps/commons-ip) (dockerized,
release jar pinned), prints every FAILED check with its messages, and exits non-zero if
the package is not `VALID`.

Each run's validation reports are published to `reports/runs/<timestamp>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs):

```
docker compose up -d reports
open http://localhost:8080
```

`./scripts/validate.sh <sip.zip|sip-dir>...` validates any package standalone — including
an unzipped package directory when debugging structure. See
[ADR-0005](docs/decisions/0005-dockerized-validation-and-html-reporting.md) for the design.

## Input

The input needs to adhere to these requirements:

**Basic profile**

* There must be exactly one `dc+schema.json` file.
* The metadata is serialized in `JSON-LD`. 
  * The `@context` property refers to these two ontologies:
    * `dcterms`: `http://purl.org/dc/terms/`
    * `schema`: `http://schema.org/`
* There must be one or more representation directories named `representation_N`
  (`representation_1`, `representation_2`, …), each holding at least one essence file.
  Directories not matching that name pattern are currently skipped silently.

## Documentation

[docs/](docs/) holds the project documentation: [docs/sip-creator-design.md](docs/sip-creator-design.md)
describes the system as it is today, [docs/decisions/](docs/decisions/) records why key
choices were made, and [docs/TODO.md](docs/TODO.md) is the live backlog. Start with
[docs/README.md](docs/README.md) for how it all fits together.