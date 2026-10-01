[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

# SIP Creator

A command-line tool and Go library for building Submission Information Packages (SIPs):
your content files plus descriptive metadata, rolled into a standards-conformant
[E-ARK SIP](https://earksip.dilcis.eu/), ready for ingest into any E-ARK-conformant repository.

E-ARK profiles specialize the output for a particular archive. This project implements
three: two plain E-ARK profiles for E-ARK-conformant repositories, with Simple Dublin Core
or MODS 3.7 as the descriptive metadata, and meemoo's `basic` profile, building SIPs
conforming to [Meemoo's SIP Specification](https://developer.meemoo.be/docs/diginstroom/sip/)
for ingest into the Flemish heritage archive. Descriptive metadata reaches the tool as
flat rows in a CSV or, under the E-ARK profiles, as a finished document.

The library is written to be usable by other institutions as it stands, as a reference
implementation of E-ARK SIP packaging: the profiles in this repository are reference
implementations, and an institution with its own descriptive standard brings its own
profile (see [Bringing your own profile](#bringing-your-own-profile)). UGent Library's
usage in the examples is the example, not the rule.

:warning: **This is an experimental package** :warning:

## Features

* Implements three profiles, selected with `--profile`: `eark` builds a plain
  [E-ARK SIP](https://earksip.dilcis.eu/) 2.2.0 for E-ARK-conformant repositories with
  Simple Dublin Core descriptive metadata; `eark-mods` builds the same package with
  [MODS 3.7](https://www.loc.gov/standards/mods/) descriptive metadata for bibliographic
  records; `basic` builds a SIP conforming to the
  [Meemoo SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/),
  built on E-ARK SIP 2.0.4, for ingest into the Flemish heritage archive.
* Builds a complete package from a plain input folder: your content files plus a simple
  descriptive rows file (`description.csv`) or, under the eark profiles, a finished
  `dc.xml` or `mods.xml`, out comes a SIP with generated METS and PREMIS metadata and
  natively computed checksums.
* Validates an input folder before building (`check`, with the same `--profile` as
  `create`), reporting every violation at once.
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

**Delivering to an E-ARK-conformant repository (eark and eark-mods profiles):** the zip
is the deliverable; ingest it directly.

**Delivering to Meemoo (basic profile):** meemoo's transfer format wraps the SIP in a
BagIt bag, an envelope this tool deliberately does not produce. Use `--no-zip` to
create a package directory, bag that *directory* with a reference BagIt
implementation, and then follow meemoo's transfer instructions:

```
./bin/sip-creator create --profile basic --no-zip ./your-input sip-out
bagit.py --md5 sip-out/uuid-<uuid>/
```

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
// The second argument is the meemoo OR-id: required by the basic
// profile, ignored by the plain E-ARK ones.
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
	// Each profile package owns its description type: the eark profile
	// writes Simple Dublin Core from eark.Terms, the basic profile meemoo's
	// dc+schema document from meemoo.Terms (both lists of sip.Term, keyed
	// by the plain keys of the input specification's tables), and the
	// eark-mods profile MODS 3.7 from an earkmods.Record (below).
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

The `eark-mods` profile takes a bibliographic record instead of a list of terms. MODS is
a tree, so the record is typed by field: its identifier, its titles, and the library's
physical copies of it as items, each a call number with an optional barcode and an
optional volume or issue designation. Items belong on the package-level record, because
a representation is a version of the content, never a copy. The library does not refuse
items on a representation's record; it writes them to that representation's `mods.xml`:

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

A record richer than the fields, or one that already exists as a document, travels as a
supplied file: the two eark profiles accept a `build.DescriptiveDocument` in place of
terms or a record; the `basic` profile takes terms only. What the tool checks and how it
copies the file is described under [Input](#input):

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
representation's type; empty means the `Label`.

What `Type` does depends on the profile:

* The **eark and eark-mods profiles** declare each representation's resolved type in that
  representation's METS content typing (`TYPE="Other"` with `csip:OTHERTYPE`,
  and `csip:CONTENTINFORMATIONTYPE="OTHER"` with
  `csip:OTHERCONTENTINFORMATIONTYPE`). Ingest systems read one of those pairs
  as the representation's type: RODA v5.7.0 and later shows the value in the
  Type column of the AIP's representations.
* The **basic profile ignores `Type`**: meemoo SIP 1.2 fixes every METS
  content typing to `OTHER` plus the profile URI
  (`https://data.hetarchief.be/id/sip/1.2/basic`), so the spec leaves no
  attribute for a producer-chosen type. `Label` still becomes `mets/@LABEL`.

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
to `check` and `create` says which vocabulary the rows are in: under `basic` the keys
come from meemoo's closed vocabulary of Dublin Core terms plus two schema.org
properties; under `eark` they are the fifteen Simple Dublin Core elements; under
`eark-mods` they are the MODS keys, `identifier` and `title` (the tables are in the
[input specification](docs/input-spec.md)). Repeat a key for multiple values, and tag
a value's language in square brackets where it matters. This example is for `basic`:
`created`, `spatial` and `extent` are meemoo keys, and `eark` refuses them as unknown
(its Simple Dublin Core elements are `date`, `coverage` and `format`):

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

`identifier` and `title` are always required. Under `basic`, `description` and `created`
are required too (meemoo's basic content profile), as is a Dutch (`[nl]`) entry wherever
a language-tagged key is used; `check` reports all of these. An unknown key is an error: a
typo must not silently drop metadata.

Under `eark` and `eark-mods` a finished document can stand in for the rows: `dc.xml` (a
`simpledc` document) or `mods.xml` (a `mods:mods` document declaring version 3.7), at the
top level or inside a representation folder, one or the other per level. This is the
route for a record the flat rows cannot say, such as a MODS record with its physical
copies. The tool checks that the file parses as XML with that root element and
copies it into the package as it is; validity against the schema stays with the
validators downstream. Under `basic` there is no document route, because meemoo's
document must carry the identifier the tool mints.

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
required for **every** profile, including `eark`, which builds with the name alone.
The meemoo profile (`basic`) also requires `SIP_SUBMITTER_OR_ID`, the organization's
identifier in [Meemoo's organization register](https://developer.meemoo.be/). It is
emitted as the agent's `IDENTIFICATIONCODE` note (Meemoo SIP 1.2); the other profiles
ignore it. A build refuses to run when a value its profile requires is missing, rather than
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