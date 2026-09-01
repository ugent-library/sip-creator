[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

# SIP Creator

A command-line tool and Go library for building Submission Information Packages (SIPs):
your content files plus descriptive metadata, rolled into a standards-conformant
[E-ARK SIP](https://earksip.dilcis.eu/), ready for ingest into any E-ARK compliant archival repositories. 

E-ARK Profiles specialize the output for a particular archive. This project implements the 
Meemoo profiles building SIPs conforming to
[Meemoo's SIP Specification](https://developer.meemoo.be/docs/diginstroom/sip/) for
ingest into the Flemish heritage archive.

:warning: **This is an experimental package** :warning:

## Features

* Implements two profiles, selected with `--profile`: `eark` builds a plain
  [E-ARK SIP](https://earksip.dilcis.eu/) 2.2.0 for E-ARK-conformant repositories;
  `basic` builds a SIP conforming to the
  [Meemoo SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/),
  built on E-ARK SIP 2.0.4, for ingest into the Flemish heritage archive.
* Builds a complete package from a plain input folder: your content files plus a simple
  `metadata.csv`, out comes a SIP with generated METS and PREMIS metadata and natively
  computed checksums.
* Validates an input folder before building (`check`), reporting every violation at once.
* Optional PRONOM format identification based on a pre-computed
  [Siegfried](https://github.com/richardlehane/siegfried) report (see Format characterization).

## Installation

This project requires [Go](https://go.dev/dl/) 1.27 or later; there are no prebuilt
binaries while the tool is experimental.

Install the CLI directly from the module (the binary lands in
`$(go env GOPATH)/bin` as `sip-creator`; make sure that directory is on your `PATH`):

```sh
go install github.com/ugent-library/sip-creator@latest
```

Or build from a clone, which is what the examples in this README assume
(they invoke `./bin/sip-creator`; with `go install`, invoke `sip-creator` instead):

```sh
git clone https://github.com/ugent-library/sip-creator.git
cd sip-creator
go build -o bin/sip-creator .
```

## How to use

### As a command-line tool

Assuming you have data in a `./your-input` directory (prepared as described under Input
below) which you want to convert into a SIP package stored in a `sip-out` directory
(`create` also needs the submitting organization set in the environment, see
Configuration below):

```
./bin/sip-creator create --profile eark ./your-input sip-out
```

This writes the package directory `sip-out/uuid-<uuid>/` and zips it (uncompressed) to
`sip-out/uuid-<uuid>.zip`.

Further flags:

* `--content-category` sets the package's content category (`mets/@TYPE`,
  CSIP vocabulary), overriding `SIP_CONTENT_CATEGORY` and the profile default.
* `--status` sets the record status (SIP3 vocabulary: `new`, `supplement`,
  `replacement`, `test`, `version`, `delete`; omitted means `new`). A status
  that updates an earlier package requires `--updates <identifier>`: the
  original package's identifier is reused as this package's identifier
  (`mets/@OBJID`).
* `--no-zip` to skip zipping when the package directory itself is what you need.

**Delivering to an E-ARK-conformant repository (eark profile):** the zip is the
deliverable; ingest it directly.

**Delivering to Meemoo (basic profile):** meemoo's transfer format wraps the SIP in a
BagIt bag — an envelope this tool deliberately does not produce. Use `--no-zip` to
create a package directory. Then, bag the package *directory* with a reference BagIt implementation.
Finally, follow meemoo's transfer instructions:

```
./bin/sip-creator create --profile basic --no-zip ./your-input sip-out
bagit.py --md5 sip-out/uuid-<uuid>/
```

### As a Go library

The input folder is a CLI convention; the library takes the same material as plain Go
values, and no environment variables are involved: the submitter is data on the profile,
the destination and logger are configuration. Resolve a profile, attach the submitter,
and hand `Build` your descriptive terms and content files:

```go
import (
	"log/slog"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/profiles"
)

def, _ := profiles.Get("eark")
// The second argument is the meemoo OR-id: required by meemoo profiles,
// ignored by plain E-ARK ones.
def, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "")
if err != nil {
	// ...
}

builder := profiles.New(&profiles.Config{
	Destination: "./out",
	Logger:      slog.Default(),
})

pkg, err := builder.Build(def, &profiles.Input{
	Descriptive: metadata.Terms{
		{Element: "dcterms:identifier", Value: "inv.2024.001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Correspondentie 1914-1918"},
		{Element: "dcterms:description", Lang: "nl", Value: "Brieven uit de collectie."},
		{Element: "dcterms:created", Value: "1914/1918"},
	},
	Representations: []profiles.SourceRepresentation{{
		Label: "master",
		Files: []profiles.SourceFile{
			{Source: "/data/scans/page-001.tif", Path: "page-001.tif"},
		},
	}},
})
```

`Build` validates the input against the profile's rules, then writes the complete
package directory under `Destination` and returns the built package. Zipping is a
separate step (the `archive` package). The full API is on
[pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator); the domain model
and build lifecycle are described in
[docs/sip-creator-design.md](docs/sip-creator-design.md).

## Input

One folder is one package. The smallest valid input is a `metadata.csv` plus your content
files, flat in one folder (they become the package's single representation):

```
your-input/
├── metadata.csv
├── scan-001.tif
└── scan-002.tif
```

When the content comes in multiple versions (for instance, a preservation master and an
access copy), each version gets its own folder under `representations/`,
and the optional extras slot in per package or per representation:

```
your-input/
├── metadata.csv              required: descriptive metadata for the package
├── siegfried.json            optional: characterization sidecar (see Format characterization)
├── documentation/            optional: context material about the package
│   └── README.txt
├── premis/                   optional: received preservation XML, passed through as-is
│   └── vendor-events.xml
└── representations/
    ├── master/
    │   ├── scan-001.tif
    │   ├── scan-002.tif
    │   ├── metadata.csv      optional: terms that apply to this version only
    │   ├── documentation/    optional
    │   │   └── notes.txt
    │   └── premis/           optional
    │       └── scanner-events.xml
    └── access/
        ├── scan-001.jpg
        └── scan-002.jpg
```

`metadata.csv` is a two-column `key,value` file with a header row. Keys come from a 
closed vocabulary of Dublin Core terms (the full key
table is in the [input specification](docs/input-spec.md)). Repeat a key for multiple
values, and tag a value's language in square brackets where it matters:

```csv
key,value
identifier,BIB.FA.XXXX.XXX
title[nl],Fotoalbum Gent 1913
description[nl],Album met 48 zwart-witfoto's van de Gentse binnenstad
created,1913
creator,Onbekend
subject[nl],stadsgezichten
subject[nl],wereldtentoonstellingen
spatial[nl],Gent
extent[nl],48 foto's
rights[nl],publiek domein
```

`identifier` and `title` are always required; meemoo profiles also require `description`
and `created`, and a Dutch (`[nl]`) entry wherever a language-tagged key is used. An
unknown key is an error: a typo must not silently drop metadata.

In short:

* **`metadata.csv`** (required): the descriptive metadata, see the example
  above.
* **Content**: either flat in the folder (one representation, named after the
  input folder itself), or one folder per version under
  `representations/<your-name>/`. Names are free-form (letters, digits,
  `._-`) and are used as-is: your folder name becomes the representation's
  folder name inside the generated SIP and its human-readable name in the
  metadata.
* **Optional**: `documentation/` (context material; also per representation —
  recommended: validators flag its absence as a SHOULD-level warning),
  `premis/` (preservation XML received from a vendor, passed through as-is;
  also per representation), a per-representation `metadata.csv` (e.g. a
  license that differs between master and access copy), and the
  `siegfried.json` characterization sidecar (see Format characterization).

Validate a folder without building anything (no configuration needed):

```
./bin/sip-creator check ./your-input
```

It reports every violation at once, in plain language.

## Configuration

Configuration is read from the environment (a `.env` file is loaded when present — start
from `.env.example`). All environment variables are documented in
[CONFIG.md](CONFIG.md).

**Submitting organization** (required for `create`)

Every package's METS names the organization submitting it, so `SIP_SUBMITTER_NAME` is
required for **every** profile — including `eark`, which builds with the name alone.
Meemoo profiles (`basic`) additionally require `SIP_SUBMITTER_OR_ID` — the organization's
identifier in [Meemoo's organization register](https://developer.meemoo.be/) — which is
emitted as the agent's `IDENTIFICATIONCODE` note (Meemoo SIP 1.2); other profiles ignore
it. A build refuses to run when a value its profile requires is missing, rather than
emitting a package that would be rejected at ingest:

```
SIP_SUBMITTER_NAME="Universiteitsbibliotheek Gent"
SIP_SUBMITTER_OR_ID="OR-a1b2c3d"
```

**Format characterization** (optional, input — not configuration)

Format info comes from a pre-computed
[Siegfried](https://github.com/richardlehane/siegfried) report placed next to your
input; the tool itself never runs Siegfried. Install Siegfried if you want format info
in your packages, generate the report **from the input root**, and the build picks it
up by name:

```sh
cd ./your-input && sf -hash md5 -json . > siegfried.json
```

Without a `siegfried.json` the build succeeds with no format info (`premis:format` is a
SHOULD; checksums and sizes are always computed natively). When the sidecar is present it
is strictly verified: a malformed report, an essence file missing from it, a report made
without `-hash md5`, or a file changed since the report was generated aborts the build.

## Development

This section is for developing SIP Creator itself.

`go test ./...` runs the Go test suite.

The scripts below require [Docker](https://www.docker.com/) (runs the commons-ip
validator and the report server) and `jq`; `build.sh` also needs `sf`
([Siegfried](https://github.com/richardlehane/siegfried)) on your `PATH`, because it
regenerates the input fixture's `siegfried.json` sidecar before building.

`./build.sh [profile]` (default `basic`) is the local CI loop: it rebuilds, regenerates
the sample SIP from `tmp/<profile>`, validates the zip with
[commons-ip](https://github.com/keeps/commons-ip) (dockerized, release jar pinned),
prints every FAILED check with its messages, and exits non-zero if the package is not
`VALID`. Each profile validates against the supported E-ARK spec version: `basic`
(meemoo 1.2) against 2.0.4, `eark` against 2.2.0. Both are expected to report `VALID`.

Each run's validation reports are published to `reports/runs/<timestamp>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs):

```
docker compose up -d reports
open http://localhost:8080
```

`./scripts/validate.sh <sip.zip|sip-dir>...` validates any package standalone — including
an unzipped package directory when debugging structure.

## Documentation

[docs/](docs/) holds the project documentation: [docs/sip-creator-design.md](docs/sip-creator-design.md)
describes the system as it is today, [docs/decisions/](docs/decisions/) records why key
choices were made, and [docs/TODO.md](docs/TODO.md) is the live backlog. Start with
[docs/README.md](docs/README.md) for how it all fits together.

## License

[Apache 2.0](LICENSE).