[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

# SIP Creator

SIP Creator packages your content files and their descriptive metadata into an
[E-ARK](https://earksip.dilcis.eu/) Submission Information Package (SIP), ready to hand
to a digital archive. It is a Go library, and a command-line tool built on it.

Archives differ in what they expect inside a SIP. A profile captures one archive's
expectations: the specification version, the descriptive metadata standard, and the
rules your input must meet. Three are included: `eark` (Dublin Core) and `eark-mods`
(MODS) for E-ARK-conformant repositories, and `basic` for meemoo, the Flemish heritage
archive (see [Profiles](#profiles)).

:warning: **This is an experimental package** :warning:

## Features

* Builds a complete package from a plain input folder: your content files plus a simple
  descriptive rows file (`description.csv`) or, where the profile allows it, a finished
  descriptive document. Out comes a SIP with generated METS and PREMIS metadata and
  natively computed checksums.
* Validates an input folder before building (`check`, with the same `--profile` as
  `create`), reporting every violation at once.
* Optional PRONOM format identification based on a pre-computed
  [Siegfried](https://github.com/richardlehane/siegfried) report (see Format characterization).
* Profiles of your own for other archives or descriptive standards (see
  [Bringing your own profile](#bringing-your-own-profile)).

## Profiles

Choose a profile with `--profile` on the command line, or with `profiles.Get` in Go.

| | `eark` | `eark-mods` | `basic` |
|---|---|---|---|
| Built for | E-ARK-conformant repositories | E-ARK-conformant repositories | meemoo (hetarchief.be) |
| Specification | E-ARK SIP 2.2.0 | E-ARK SIP 2.2.0 | meemoo SIP 1.2, on E-ARK SIP 2.0.4 |
| Descriptive standard | Simple Dublin Core | MODS 3.7 | meemoo's Dublin Core and schema.org |
| [`description.csv` keys](docs/input-spec.md#3-descriptive-metadata-descriptioncsv-or-a-supplied-document) | the 15 Dublin Core elements | `identifier`, `title` | meemoo's vocabulary |
| Required keys | `identifier`, `title` | `identifier`, `title` | `identifier`, `title`, `description`, `created` |
| [Finished document](docs/input-spec.md#supplying-a-finished-document-eark-and-eark-mods) accepted | `dc.xml` | `mods.xml` | none |
| Go description type | `eark.Terms` | `earkmods.Record` | `meemoo.Terms` |
| Submitter | name | name | name and meemoo OR-id |
| Representation type | written to the METS | written to the METS | ignored |
| You deliver | the zip | the zip | the package directory, in a BagIt bag |

### `eark`: E-ARK with Dublin Core

Builds a plain [E-ARK SIP](https://earksip.dilcis.eu/) 2.2.0 with a Simple Dublin Core
document (`dc.xml`) as its descriptive metadata. `description.csv` takes the fifteen
Dublin Core elements (`title`, `creator`, `date`, `coverage`, ...) as keys. A language
tag on a key is accepted but not written into the document.

Each representation's type goes into that representation's METS content typing
(`csip:OTHERTYPE` and `csip:OTHERCONTENTINFORMATIONTYPE`); RODA v5.7.0 and later shows it
in the Type column of the AIP's representations. The zip the tool writes is the
deliverable: ingest it as it is.

### `eark-mods`: E-ARK with MODS

The same package as `eark`, with a [MODS 3.7](https://www.loc.gov/standards/mods/)
document (`mods.xml`) for bibliographic records. Everything except the descriptive
metadata works as under `eark`.

MODS is a tree, so `description.csv` holds only `identifier` and `title`. A richer
record, such as one listing the library's physical copies of the work (call number,
barcode, volume), comes as a finished `mods.xml`, or in Go as an `earkmods.Record` with
`Items`. Physical copies belong on the package-level record, because a representation is
a version of the content, never a copy. The library does not refuse items on a
representation's record; it writes them to that representation's `mods.xml`.

### `basic`: meemoo SIP 1.2

Builds a SIP conforming to the basic content profile of the
[meemoo SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/),
for ingest into the Flemish heritage archive. `description.csv` takes meemoo's closed
vocabulary: Dublin Core terms (`created`, `spatial`, `extent`, ...) plus two schema.org
properties. On top of the four required keys, meemoo requires a Dutch value (`[nl]`)
wherever a language-tagged key is used.

There is no finished-document route: meemoo's document must carry the package identifier
the tool mints, and the tool does not edit XML. The submitter needs meemoo's OR-id as
well as a name (see [Configuration](#configuration)). A representation's type is
ignored, because meemoo SIP 1.2 fixes every METS content typing to `OTHER` plus the
profile URI; its label still becomes `mets/@LABEL`.

meemoo's transfer format wraps the SIP in a BagIt bag, which this tool does not produce.
Build a package directory with `--no-zip`, bag that directory with a reference BagIt
implementation, and follow meemoo's transfer instructions:

```
./bin/sip-creator create --profile basic --no-zip ./your-input sip-out
bagit.py --md5 sip-out/uuid-<uuid>/
```

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
  `replacement`, `test`, `version`, `delete`). Without it the METS carries no
  status, which the E-ARK SIP specification reads as `new`. A status
  that updates an earlier package requires `--updates <identifier>`: the
  original package's identifier is reused as this package's identifier
  (`mets/@OBJID`).
* `--no-zip` to skip zipping when the package directory itself is what you need.

To check a folder without building anything, pass the same profile. `check` does not
need the `SIP_SUBMITTER_*` variables, but like every command it reads `.env` when
present and stops on a malformed one:

```
./bin/sip-creator check --profile eark ./your-input
```

What you deliver depends on the profile: the zip for `eark` and `eark-mods`, a bagged
package directory for `basic` (see [Profiles](#profiles)).

### As a Go library

The input folder is a CLI convention. The library takes the same source package as plain
Go values and reads no environment variables:

* the submitting organization is added to the profile with `WithSubmitter`;
* the destination directory and the logger go in `build.Config`;
* the descriptive metadata and the content files go in `build.SourcePackage`.

Resolve a profile, attach the submitter, and hand `Build` your descriptive terms and
content files:

```go
import (
	"log/slog"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

def, _ := profiles.Get("eark")
// The second argument is the meemoo OR-id, used by basic only.
def, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "")
if err != nil {
	// ...
}

builder, err := build.New(&build.Config{
	Profile:     def,
	Destination: "./out",
	Logger:      slog.Default(),
})
if err != nil {
	// ...
}

pkg, err := builder.Build(&build.SourcePackage{
	// The description's type belongs to the profile (see Profiles).
	Description: eark.Terms{
		{Key: "identifier", Value: "inv.2024.001"},
		{Key: "title", Value: "Correspondentie 1914-1918"},
		{Key: "description", Value: "Brieven uit de collectie."},
		{Key: "date", Value: "1914/1918"},
	},
	Representations: []build.SourceRepresentation{{
		Name: "master",
		Files: []build.SourceFile{
			{Source: "/data/scans/page-001.tif", Path: "page-001.tif"},
		},
	}},
})
```

`Build` validates the source package against the profile's rules, then writes the complete
package directory under `Destination` and returns the built package. Zipping is a separate
step (the `archive` package). A builder is constructed once per profile; the values that
change per package go in the `SourcePackage`. The example above builds a new package with
the profile's METS values. Three fields override them per package: `PackageIdentifier`
and `RecordStatus` for an update of an earlier package, and `ContentCategory`:

```go
pkg, err := builder.Build(&build.SourcePackage{
	// The identifier of the package this one updates becomes this package's
	// mets/@OBJID; an update-class status requires it. RecordStatus is
	// metsHdr/@RECORDSTATUS in the SIP3 vocabulary (the sip.RecordStatus
	// constants) and ContentCategory is mets/@TYPE, the CSIP content
	// category. Empty means the profile's value, which for the status the
	// E-ARK SIP spec reads as NEW.
	PackageIdentifier: "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e",
	RecordStatus:      sip.RecordStatusReplacement,
	ContentCategory:   "Textual works – Print",
	// Description and Representations as above.
})
```

The `eark-mods` profile takes an `earkmods.Record` instead of a list of terms: an
identifier, titles, and physical copies as items, each a call number with an optional
barcode and an optional volume or issue designation:

```go
import "github.com/ugent-library/sip-creator/profiles/earkmods"

def, _ := profiles.Get("eark-mods")
// WithSubmitter and build.New as above.

pkg, err := builder.Build(&build.SourcePackage{
	Description: earkmods.Record{
		Identifier: "990001234560471",
		Titles: []earkmods.Title{
			{Value: "Correspondentie 1914-1918", Lang: "nl"},
		},
		Items: []earkmods.Item{
			{CallNumber: "BIB.HS.001", Barcode: "000012345678"},
			{CallNumber: "BIB.HS.002", Enumeration: "deel 2"},
		},
	},
	// Representations as above.
})
```

A record that already exists as a document travels as a file: profiles that accept a
finished document (see [Profiles](#profiles)) take a `build.DescriptiveDocument` in place
of terms or a record. What the tool checks and how it copies the file is described under
[Input](#input):

```go
pkg, err := builder.Build(&build.SourcePackage{
	Description: build.DescriptiveDocument{Source: "/data/records/990001234560471/mods.xml"},
	// Representations as above.
})
```

#### Bringing your own profile

The three profiles above are reference implementations of one route, and the engine
imports none of them. An institution with its own descriptive standard writes a package
with three parts:

* a description type implementing `sip.Description`, whose `Validate` and
  `ValidateRequired` are the rules of your standard;
* an encoder implementing `build.DescriptionEncoder`: the type check, the code that
  writes the document (the profiles here use `text/template`), and the list of XSDs
  the document points at;
* an exported `build.Definition` naming the encoder, the document's file name and the
  METS values (`sip.MetsDeclaration`: profile URL, content typing, `MDTYPE`). The
  encoder type itself can stay unexported.

Hand that definition to `build.New` as above. The XSDs an encoder lists must be ones
this repository bundles in `schemas/`: the build refuses any other name, so a standard
whose schema is not bundled needs its XSD added there first. The registry in `profiles/`
is the CLI's list of what `--profile` can name; a library caller's package need not join
it. When your profile has a value the caller should choose, such as the `type` of a MODS
identifier, offer a fixed set of typed constants rather than free text, so two callers
making the same choice write the same document (ADR-0022).

The full API is on [pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator);
the domain model and build lifecycle are described in
[docs/sip-creator-design.md](docs/sip-creator-design.md).

#### Representation labels and types

Besides the required `Name` (the directory under `representations/`), each
representation takes two optional fields. `Label` is the display name, emitted
as the representation METS `mets/@LABEL`; empty means the `Name`. `Type` is the
representation's type; empty means the `Label`. Whether the type reaches the METS
depends on the profile (see [Profiles](#profiles)).

On the CLI the same fields come from the optional `representations.csv`
(see [Input](#input) below).

## Input

One folder is one package. The smallest valid input is a descriptive rows file,
`description.csv`, plus your content files, flat in one folder (they become the
package's single representation):

```
your-input/
├── description.csv
├── scan-001.tif
└── scan-002.tif
```

When the content comes in multiple versions (for instance, a preservation master and an
access copy), each version gets its own folder under `representations/`,
and the optional extras slot in per package or per representation:

```
your-input/
├── description.csv           required: descriptive metadata (or dc.xml / mods.xml, see below)
├── representations.csv       optional: a label and type per representation
├── siegfried.json            optional: characterization sidecar (see Format characterization)
├── documentation/            optional: context material about the package
│   └── README.txt
├── premis/                   optional: received preservation XML, passed through as-is
│   └── vendor-events.xml
└── representations/
    ├── master/
    │   ├── scan-001.tif
    │   ├── scan-002.tif
    │   ├── description.csv   optional: terms (or a document) for this version only
    │   ├── documentation/    optional
    │   │   └── notes.txt
    │   └── premis/           optional
    │       └── scanner-events.xml
    └── access/
        ├── scan-001.jpg
        └── scan-002.jpg
```

Representation folder names may use letters, digits and `._-`, and are used as-is: the
folder name becomes the representation's folder name inside the SIP and, unless
`representations.csv` says otherwise, its label and type in the metadata. In the flat
case the representation is named after the input folder itself. A representation's own
`description.csv` describes that version only, such as a license that differs between
master and access copy. `documentation/` is recommended: validators warn (a CSIP SHOULD)
when a package has none.

The rows file is a two-column `key,value` file with a header row. The profile you pass
to `check` and `create` decides which keys are allowed and which are required (see
[Profiles](#profiles); the full key tables are in the
[input specification](docs/input-spec.md)). Repeat a key for multiple values, and tag
a value's language in square brackets where it matters. This example uses `basic` keys:

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

`check` reports a missing required key. An unknown key is an error: a typo must not
silently drop metadata.

Where the profile accepts a finished document, it can stand in for the rows, at the top
level or inside a representation folder, one or the other per level. The tool checks
that the file parses as XML with the root element the profile expects and copies it into
the package as it is; validity against the schema stays with the validators downstream.

The optional `representations.csv` gives each representation folder a display
label and a type (what an ingest system such as RODA shows as the
representation's kind) when the folder names alone don't say it. It is a
table with a `directory,label,type` header row; `directory` names a folder
under `representations/` and is required, the other two columns are optional
(an empty `label` means the folder name, an empty `type` means the label):

```csv
directory,label,type
master,Master scan (TIFF),archival
access,Access copy (JPEG),access
```

When the file is present it must be complete: every row must match a folder
and every folder must have a row, so nothing can silently drop out of the
package. The full rules are in the
[input specification](docs/input-spec.md). Run `check` (see
[As a command-line tool](#as-a-command-line-tool)) to test a folder against them: it
reports every violation at once, in plain language.

## Configuration

Configuration is read from the environment. A `.env` file is loaded when present; start
from `.env.example`. All environment variables are documented in
[CONFIG.md](CONFIG.md).

**Submitting organization** (required for `create`)

Every package's METS names the organization submitting it, so `SIP_SUBMITTER_NAME` is
required for every profile. `basic` also requires `SIP_SUBMITTER_OR_ID`, the
organization's identifier in [meemoo's organization register](https://developer.meemoo.be/),
emitted as the agent's `IDENTIFICATIONCODE` note. A build refuses to run when a value its profile requires is missing, rather than
emitting a package that would be rejected at ingest:

```
SIP_SUBMITTER_NAME="Universiteitsbibliotheek Gent"
SIP_SUBMITTER_OR_ID="OR-a1b2c3d"
```

**Format characterization** (optional; this is input, not configuration)

Format info comes from a pre-computed
[Siegfried](https://github.com/richardlehane/siegfried) report placed next to your
input; the tool itself never runs Siegfried. Install Siegfried if you want format info
in your packages, generate the report **from the input root**, and the build picks it
up by name. Capture the report before writing it, so sf never scans its own
half-written output:

```sh
cd ./your-input && report="$(sf -hash md5 -json .)" && printf '%s\n' "$report" > siegfried.json
```

Without a `siegfried.json` the build succeeds with no format info (`premis:format` is a
SHOULD; checksums and sizes are always computed natively). When the sidecar is present it
is strictly verified: a malformed report, an essence file missing from it, a report made
without `-hash md5`, or a file changed since the report was generated aborts the build.

## Development

This section is for developing SIP Creator itself.

`go test ./...` runs the Go test suite.

The scripts below require `jq`, and [Docker](https://www.docker.com/) for the commons-ip
validator and the report server. To run commons-ip on a local Java instead of in Docker,
set `CSIP_CMD` to the command, for example
`CSIP_CMD="java -jar commons-ip2-cli-2.11.2.jar" ./build.sh eark`, using the jar version
that `docker/validator/Dockerfile` pins. `build.sh` also needs `sf`
([Siegfried](https://github.com/richardlehane/siegfried)) on your `PATH`, because it
regenerates the input fixture's `siegfried.json` sidecar before building, and `xmllint`
(part of libxml2, present on macOS and most Linux systems), which checks every
`mods.xml` in an `eark-mods` package against the bundled MODS schema.

`./build.sh [profile]` (default `basic`) is the local CI loop: it rebuilds, regenerates
the sample SIP from `tmp/<profile>`, validates the zip with
[commons-ip](https://github.com/keeps/commons-ip) (dockerized, release jar pinned),
prints every FAILED check with its messages, and exits non-zero if the package is not
`VALID`. Each profile validates against the supported E-ARK spec version: `basic`
(meemoo 1.2) against 2.0.4, `eark` and `eark-mods` against 2.2.0. All three are expected
to report `VALID`. commons-ip does not validate the descriptive documents the METS points
at, so the script also runs `xmllint` over every `mods.xml` in the package against the
MODS 3.7 schema the package ships, offline through the XML catalog in
`scripts/schema-catalog.xml`, and fails the run when one is not valid.

Each run's validation reports are published to `reports/runs/<timestamp>-<profile>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs):

```
docker compose up -d reports
open http://localhost:8080
```

`./scripts/validate.sh <sip.zip|sip-dir>...` validates any package standalone, including
an unzipped package directory when debugging structure.

## Documentation

[docs/](docs/) holds the project documentation: [docs/sip-creator-design.md](docs/sip-creator-design.md)
describes the system as it is today, [docs/decisions/](docs/decisions/) records why key
choices were made, and [docs/TODO.md](docs/TODO.md) is the live backlog. Start with
[docs/README.md](docs/README.md) for how it all fits together.

## License

[Apache 2.0](LICENSE).