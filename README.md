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

Install the command-line tool directly from the module (the binary lands in
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

To use the library in your own Go program:

```sh
go get github.com/ugent-library/sip-creator
```

## Command-line tool

The `sip-creator` command builds a package from an input folder you prepare, using the
library underneath. It reads the submitting organization from the environment.

### Creating a package

Assuming you have data in a `./your-input` directory (prepared as described under
[Input folder](#input-folder)) which you want to convert into a SIP package stored in a
`sip-out` directory (with the submitting organization set, see
[Configuration](#configuration)):

```
./bin/sip-creator create --profile eark ./your-input sip-out
```

This writes the package directory `sip-out/uuid-<uuid>/` and zips it (uncompressed) to
`sip-out/uuid-<uuid>.zip`. To try it before preparing your own input, build one of the
[example folders](examples/): `./bin/sip-creator create --profile eark examples/eark sip-out`.

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

What you deliver depends on the profile: the zip for `eark` and `eark-mods`, a bagged
package directory for `basic` (see [Profiles](#profiles)).

### Checking an input folder

To check a folder without building anything, pass the same profile. `check` reports
every violation at once, in plain language. It does not need the `SIP_SUBMITTER_*`
variables, but like every command it reads `.env` when present and stops on a malformed
one:

```
./bin/sip-creator check --profile eark ./your-input
```

### Configuration

Configuration is read from the environment. A `.env` file is loaded when present; start
from `.env.example`. All environment variables are documented in
[CONFIG.md](CONFIG.md).

Every package's METS names the organization submitting it, so `create` requires
`SIP_SUBMITTER_NAME` for every profile. `basic` also requires `SIP_SUBMITTER_OR_ID`, the
organization's identifier in [meemoo's organization register](https://developer.meemoo.be/),
emitted as the agent's `IDENTIFICATIONCODE` note. A build refuses to run when a value its
profile requires is missing, rather than emitting a package that would be rejected at
ingest:

```
SIP_SUBMITTER_NAME="Example Organization"
SIP_SUBMITTER_OR_ID="OR-a1b2c3d"
```

### Input folder

One folder is one package. The smallest valid input is a `description.csv` plus your
content files, which become the package's single representation:

```
your-input/
├── description.csv
├── scan-001.tif
└── scan-002.tif
```

When the content comes in several versions, such as a preservation master and an access
copy, each version gets its own folder under `representations/`:

```
your-input/
├── description.csv
├── representations.csv
├── siegfried.json
├── documentation/
├── premis/
└── representations/
    ├── master/
    │   ├── scan-001.tif
    │   ├── description.csv
    │   ├── documentation/
    │   └── premis/
    └── access/
        └── scan-001.jpg
```

| Name | Required | What it holds | Rules |
|---|---|---|---|
| `description.csv` | yes, or the profile's document | descriptive metadata as `key,value` rows | [§3](docs/input-spec.md#3-descriptive-metadata-descriptioncsv-or-a-supplied-document) |
| `dc.xml`, `mods.xml` | instead of `description.csv`, where the profile accepts one | a finished descriptive document | [§3](docs/input-spec.md#supplying-a-finished-document-eark-and-eark-mods) |
| `representations/<name>/` | no | one folder per version of the content | [§2](docs/input-spec.md#2-content-files-and-representations) |
| `representations.csv` | no | a label and type per representation folder | [§2](docs/input-spec.md#representationscsv-labels-and-types-optional) |
| `documentation/` | no, recommended | context material; validators warn without it | [§4](docs/input-spec.md#4-documentation) |
| `premis/` | no | received preservation XML, copied as it is | [§5](docs/input-spec.md#5-received-preservation-files-premis) |
| `siegfried.json` | no | a format characterization report | [below](#format-characterization) |

Everything else is content. A representation folder can hold its own `description.csv`
(or document), `documentation/` and `premis/`, about that version only. Representation
folder names may use letters, digits and `._-`; in the simple case the representation is
named after the input folder.

[examples/](examples/) has a complete input folder for each profile.

#### `description.csv`

A two-column file with a `key,value` header row. Repeat a key for more values, and add a
language in square brackets where it matters (`title[nl]`). An unknown key is an error, so
a typo cannot silently drop metadata. Which keys exist and which are required depends on
the profile (see [Profiles](#profiles)). The smallest valid file per profile:

`eark`:

```csv
key,value
identifier,example-0001
title,Example photograph
```

`eark-mods`:

```csv
key,value
identifier,example-0001
title[en],Example book
```

`basic`:

```csv
key,value
identifier,example-0001
title[nl],Voorbeeldfoto
description[nl],Een voorbeeldpakket met één afbeelding.
created,2026-01-15
```

A finished document replaces the file at the same level. The tool checks only that it is
XML with the root element the profile expects, and copies it as it is.

#### `representations.csv`

Gives each representation folder a display label and a type. `directory` is required; an
empty `label` means the folder name, an empty `type` means the label:

```csv
directory,label,type
master,Master scan (TIFF),archival
access,Access copy (JPEG),access
```

When the file is present, every folder must have a row and every row must match a
folder, so no content can silently drop out of the package.

#### Format characterization

The tool adds format info (PRONOM identifiers) from a Siegfried report in
`siegfried.json`; it never runs Siegfried itself. Generate the report from the input
root, capturing it before writing so `sf` does not scan its own half-written output:

```sh
cd ./your-input && report="$(sf -hash md5 -json .)" && printf '%s\n' "$report" > siegfried.json
```

Without the report, the package has no format info; checksums and sizes are always
computed. With it, the build stops when the report is malformed, made without
`-hash md5`, misses a content file, or no longer matches a file's checksum.

## Go library

The command-line tool is one program built on this library. In your own program you
supply as Go values what the tool reads from the input folder and the environment, and
the library reads no environment variables:

* the submitting organization is added to the profile with `WithSubmitter`;
* the destination directory and the logger go in `build.Config`;
* the descriptive metadata and the content files go in `build.SourcePackage`.

The full API is on [pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator);
the domain model and build lifecycle are described in
[docs/sip-creator-design.md](docs/sip-creator-design.md).

### Building a package

Resolve a profile, attach the submitter, and hand `Build` your descriptive terms and
content files:

```go
import (
	"log/slog"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
)

def, _ := profiles.Get("eark")
// The second argument is the meemoo OR-id, used by basic only.
def, err := def.WithSubmitter("Example Organization", "")
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
		{Key: "identifier", Value: "example-0001"},
		{Key: "title", Value: "Example photograph"},
		{Key: "description", Value: "An example package with one image."},
		{Key: "date", Value: "2026-01-15"},
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
change per package go in the `SourcePackage`. Besides the description and the
representations, it takes `Documentation` and `Premis` files, the library's form of the
input folder's `documentation/` and `premis/`.

### Descriptive metadata

The `eark-mods` profile takes an `earkmods.Record` instead of a list of terms: an
identifier, titles, and physical copies as items, each a call number with an optional
barcode and an optional volume or issue designation:

```go
import "github.com/ugent-library/sip-creator/profiles/earkmods"

def, _ := profiles.Get("eark-mods")
// WithSubmitter and build.New as above.

pkg, err := builder.Build(&build.SourcePackage{
	Description: earkmods.Record{
		Identifier: "example-0001",
		Titles: []earkmods.Title{
			{Value: "Example book", Lang: "en"},
		},
		Items: []earkmods.Item{
			{CallNumber: "EX.0001", Barcode: "0000000001"},
			{CallNumber: "EX.0002", Enumeration: "vol. 2"},
		},
	},
	// Representations as above.
})
```

A record that already exists as a document travels as a file: profiles that accept a
finished document (see [Profiles](#profiles)) take a `build.DescriptiveDocument` in place
of terms or a record. The library checks and copies it the same way the command-line
tool does (see [Input folder](#input-folder)):

```go
pkg, err := builder.Build(&build.SourcePackage{
	Description: build.DescriptiveDocument{Source: "/data/records/example-0001/mods.xml"},
	// Representations as above.
})
```

### Representation labels and types

Besides the required `Name` (the directory under `representations/`), each
representation takes two optional fields. `Label` is the display name, emitted
as the representation METS `mets/@LABEL`; empty means the `Name`. `Type` is the
representation's type; empty means the `Label`. Whether the type reaches the METS
depends on the profile (see [Profiles](#profiles)). These are the fields the
command-line tool fills from `representations.csv`.

### Updating an earlier package

The example above builds a new package with the profile's METS values. Three fields
override them per package: `PackageIdentifier` and `RecordStatus` for an update of an
earlier package, and `ContentCategory`:

```go
import "github.com/ugent-library/sip-creator/sip"

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

### Format characterization

Set `Characterization` on the `SourcePackage` to add format info. Decode a Siegfried
report with `characterization.DecodeSiegfried`; the library verifies it as strictly as
the command-line tool does (see [Format characterization](#format-characterization)).

### Bringing your own profile

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

## Contributing

Tests, validation scripts and conventions for working on SIP Creator itself are in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Documentation

[docs/](docs/) holds the project documentation: [docs/sip-creator-design.md](docs/sip-creator-design.md)
describes the system as it is today, [docs/decisions/](docs/decisions/) records why key
choices were made, and [docs/TODO.md](docs/TODO.md) is the live backlog. Start with
[docs/README.md](docs/README.md) for how it all fits together.

## License

[Apache 2.0](LICENSE).