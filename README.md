[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

# SIP Creator

SIP Creator packages your content files and their descriptive metadata into an
[E-ARK](https://earksip.dilcis.eu/) Submission Information Package (SIP), ready
to hand to a digital archive.

:warning: **This is an experimental package** :warning:

## Features

A command-line tool that:

* builds a complete E-ARK SIP from the files in an input folder.
* takes descriptive metadata as a CSV or an XML file.
* creates a ZIP from the resulting SIP directory.
* checks an input folder before building.

A Go library that lets you:

* build E-ARK SIPs from your own Go programs and workflows.
* define profiles of your own.

Optional:

* PRONOM format identification from a pre-computed
  [Siegfried](https://github.com/richardlehane/siegfried) report.
* Skip building a zip of the SIP directory.

## Profiles

CSIP, the common part of E-ARK, says what every package must contain. A profile
adds the choices for one kind of content: the descriptive metadata standard, the
representations it allows, and the values its METS declares. Three profiles for
UGent Library's use are included. A program that uses the library can define
profiles of its own (see [Profiles of your own](#profiles-of-your-own)).

* `ugent/basic`, for UGent Library's archival repository: digital resources the
  library has not necessarily catalogued, described in Simple Dublin Core;
* `ugent/bibliographic`, for UGent Library's archival repository: digital
  resources the library has catalogued, described in MODS from the catalogue
  record;
* `meemoo/basic`, for Meemoo's archive through its "Digitale Instroom",
  digitized or digital cultural heritage resources, as catalogued by the UGent
  Library.

## Installation

This project requires [Go](https://go.dev/dl/) 1.27 or later. There are no
prebuilt binaries while the tool is experimental.

Install the command-line tool directly from the module. The binary lands in
`$(go env GOPATH)/bin` as `sip-creator`, so make sure that directory is on your
`PATH`:

```sh
go install github.com/ugent-library/sip-creator@latest
```

`sip-creator --version` prints the version you installed. Every package the tool
builds states the same version in its METS.

Versions follow [semantic versioning](https://semver.org/). While the major
version is 0, a minor version may change the command line, the library API or
the packages a profile builds. To install a given version, replace `@latest`
with its tag, such as `@v0.1.0`. [CHANGELOG.md](CHANGELOG.md) lists what each
version changed.

Or build from a clone. The examples in this README assume a clone and invoke
`./bin/sip-creator`. After `go install`, invoke `sip-creator` instead:

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

The `sip-creator` command builds a package from an input folder you prepare.
The recording below runs `check` and `create` on the `ugent/basic` example
folder, then lists the package directory and the zip they leave behind.

![sip-creator check and create on an example folder, then the package directory](docs/demo.gif)

### Creating a package

Prepare an input folder as described under [Input folder](#input-folder), and
set the submitting organization as described under
[Configuration](#configuration). To build the folder `./your-input` into a
package in the `sip-out` directory:

```text
./bin/sip-creator create --profile ugent/basic ./your-input sip-out
```

This writes the package directory `sip-out/uuid-<uuid>/` and zips it
(uncompressed) to `sip-out/uuid-<uuid>.zip`. To try it before preparing your own
input, build one of the [example folders](examples/):
`./bin/sip-creator create --profile ugent/basic examples/ugent/basic sip-out`.

Built from that example, the package directory looks like this (shown three
levels deep):

```text
sip-out/uuid-<uuid>/
├── documentation
│   └── README.txt
├── metadata
│   ├── descriptive
│   │   └── dc.xml
│   └── preservation
│       └── digitization-events.xml
├── METS.xml
├── representations
│   ├── access
│   │   ├── data
│   │   ├── metadata
│   │   └── METS.xml
│   └── archival
│       ├── data
│       ├── documentation
│       ├── metadata
│       └── METS.xml
└── schemas
    ├── DILCISExtensionMETS.xsd
    ├── DILCISExtensionSIPMETS.xsd
    ├── mets1_12.xsd
    ├── simpledc.xsd
    ├── xlink.xsd
    └── xml.xsd
```

* `METS.xml` is the package METS, the entry point of the package.
* `metadata/` holds the package's descriptive metadata (`dc.xml`, written from
  `description.csv`) and the preservation files from the input folder's
  `premis/`.
* `representations/` has one folder per representation folder of the input.
  Each has its own `METS.xml`, its content files under `data/`, and its own
  `metadata/` and `documentation/` where the input supplied them.
* `schemas/` holds the XSDs the METS documents reference.
* `documentation/` is the input folder's `documentation/`, copied as it is.

Further flags:

* `--content-category` sets the package's content category (`mets/@TYPE`,
  CSIP vocabulary), overriding `SIP_CONTENT_CATEGORY` and the profile default.
* `--status` sets the record status, one of the SIP3 vocabulary values `new`,
  `supplement`, `replacement`, `test`, `version` and `delete`. Without it the
  METS carries no status, which the E-ARK SIP specification reads as `new`.
* `--updates <identifier>` names the earlier package this one updates. A status
  that updates an earlier package requires it. The original package's identifier
  becomes this package's identifier (`mets/@OBJID`). Build the update into a
  destination that does not hold the original. The tool refuses to write into an
  existing package directory or zip with that identifier.
* `--no-zip` skips the zip, for when the package directory itself is what you
  need.

### Checking an input folder

The input folder has to follow the [input specification](docs/input-spec.md).
`check` tests a folder without building anything.

```text
./bin/sip-creator check --profile ugent/basic ./your-input
```

`check` reports whether the input folder is valid, and lists every problem it
found.

```text
1 problem in ./your-input

  premis/broken.xml: not well-formed XML: XML syntax error on line 2: unexpected EOF

Input folder:         ./your-input
Profile:              ugent/basic

Descriptive metadata: description.csv
Representations:      1 (1 with its own description)
Essence files:        1
Documentation files:  2
PREMIS files:         2
Format report:        not supplied (files carry no format information)

FAILED: fix the problems listed at the top and run check again.
```

The exit status tells a script what happened:

| status | meaning |
|---|---|
| 0 | the folder meets the input specification and the profile's rules |
| 1 | the folder has problems; the report lists them |
| 2 | check could not check the folder: a wrong path, a file instead of a folder, an unknown profile, or a missing argument |

### Configuration

Configuration is read from the environment. Start from `.env.example`.
All environment variables are documented in [CONFIG.md](CONFIG.md).

### Input folder

One folder is one package. Your content files live in a representation folder
under `representations/`, one folder per version of the content. The smallest
valid input is a `description.csv` plus one representation:

```text
your-input/
├── description.csv
└── representations/
    └── archival/
        ├── scan-001.tif
        └── scan-002.tif
```

When the content comes in several versions, such as an archival copy and an
access copy, each version gets its own representation folder.

Every name the tool reads in an input folder:

| Name | Required | What it holds | Rules |
|---|---|---|---|
| `description.csv` | yes, or the profile's document | descriptive metadata as `key,value` rows | [§3](docs/input-spec.md#3-descriptive-metadata-descriptioncsv-or-a-supplied-document) |
| `dc.xml`, `mods.xml` | instead of `description.csv`, where the profile accepts one | a finished descriptive document | [§3](docs/input-spec.md#supplying-a-finished-document-ugentbasic-and-ugentbibliographic) |
| `representations/<name>/` | under `meemoo/basic`, exactly one | one folder per version of the content | [§2](docs/input-spec.md#2-content-files-and-representations) |
| `representations.csv` | no | a display label per representation folder | [§2](docs/input-spec.md#representationscsv-labels-optional) |
| `documentation/` | no | context material | [§4](docs/input-spec.md#4-documentation) |
| `premis/` | no | received preservation XML, copied as it is | [§5](docs/input-spec.md#5-received-preservation-files-premis) |
| `siegfried.json` | no | a format characterization report | [below](#format-characterization) |

Content files live only in representation folders: anything else at the top
level is an error. `meemoo/basic` takes exactly one representation, under any
name. The UGent profiles take `preservation`, `archival` and `access`. A
representation folder can hold its own `documentation/` and `premis/`, and under
the UGent profiles its own description. Folder names may use letters, digits and
`._-`.

[examples/](examples/) has a complete input folder for each profile.

#### `description.csv`

A two-column file with a `key,value` header row. Repeat a key for more values,
and add a language in square brackets where it matters (`title[nl]`). An unknown
key is an error. Which keys exist and which are required depends on the profile.

The smallest valid file per profile:

`ugent/basic`:

```csv
key,value
identifier,example-0001
title,Example photograph
```

`ugent/bibliographic`:

```csv
key,value
identifier,example-0001
title[en],Example book
```

`meemoo/basic`:

```csv
key,value
identifier,example-0001
title[nl],Voorbeeldfoto
description[nl],Een voorbeeldpakket met één afbeelding.
created,2026-01-15
```

Instead of a `description.csv` file, you can also drop in an XML document per
profile:

* `ugent/basic`: `dc.xml`
* `ugent/bibliographic`: `mods.xml`

This project only checks that it is XML with the root element the profile
expects, and copies it as it is.

#### `representations.csv`

Optional. Gives each representation folder a display label, written into the
package's METS. An empty `label` means the folder name, and a label may hold any
text:

```csv
folder,label
archival,Scanned master copies (TIFF)
access,Access copy (JPEG)
```

When the file is present, every folder needs a row and every row must match a
folder, so no content can drop out of the package unnoticed. The rows set the
order of the representations in the package.

#### Format characterization

The tool adds format info (PRONOM identifiers) from a Siegfried report in
`siegfried.json`. Generate the report from the input folder, before building a
SIP.

```sh
cd ./your-input && report="$(sf -hash md5 -json .)" && printf '%s\n' "$report" > siegfried.json
```

Without the report, the package has no format info, and the tool computes every
checksum itself. With it, the build stops when the report is malformed, made
without `-hash md5`, or misses a content file. The report's MD5 is the checksum
the package declares for each file it covers. The tool does not re-verify the
checksum in the supplied `siegfried.json` file.

## Go library

SIP Creator is also a Go library. Your program gives it a profile, the
descriptive metadata and the paths of the content files. The library checks them
against the profile's rules and writes the package directory. Use it when your
application already manages the metadata and essence files. The command-line
tool is built on the same library.

The API and runnable examples are on
[pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator).

### Building a package

Resolve a profile, attach the submitter, and hand `Build` your descriptive terms
and content files:

```go
import (
	"log/slog"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/ugent"
)

def, ok := profiles.Get("ugent/basic")
if !ok {
	// ...
}
// The second argument is the Meemoo OR-id. Only meemoo/basic requires it.
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
	Description: ugent.Terms{
		{Key: "identifier", Value: "example-0001"},
		{Key: "title", Value: "Example photograph"},
		{Key: "description", Value: "An example package with one image."},
		{Key: "date", Value: "2026-01-15"},
	},
	Representations: []build.SourceRepresentation{{
		Name: "archival",
		Files: []build.SourceFile{
			{Source: "/data/scans/page-001.tif", Path: "page-001.tif"},
		},
	}},
})
```

* `Build` checks the source package against the profile's rules, writes the
  package directory under `Destination`, and returns the built package.
* Zipping is a separate step, in the `archive` package.
* One builder serves one profile. Reuse it for every package you build to that
  profile, and put what differs per package in the `SourcePackage`.

### More options

Three of these have a runnable example in
[build/example_test.go](build/example_test.go), named in brackets:

* **Documentation and PREMIS.** `Documentation` and `Premis`, on the
  `SourcePackage` or on a representation, are the library's form of the input
  folder's `documentation/` and `premis/`.
* **A bibliographic record.** `ugent/bibliographic` takes a `ugent.Record`
  instead of terms: identifiers, titles, contributors and physical copies
  (`ExampleBuilder_Build_mods`).
* **A finished document.** Where the profile accepts one, a
  `build.EncodedDescription` names a `dc.xml` or `mods.xml` to copy as it is
  (`ExampleEncodedDescription`).
* **Labels.** `SourceRepresentation.Label` is the display name in the
  representation's METS, and defaults to `Name`. Under the UGent profiles the
  type follows from the name and cannot be set.
* **Updates.** `PackageIdentifier`, `RecordStatus` and `ContentCategory` on the
  `SourcePackage` make it update an earlier package. Under the UGent profiles an
  update may hold no representations (`ExampleBuilder_Build_update`).
* **Format information.** Set `Characterization` to a Siegfried report decoded
  with `characterization.DecodeSiegfried`.

### Profiles of your own

A program that embeds the library can define a profile of its own: a description
type, a metadata model that writes and types the document, and a
`build.Definition`. Hand the definition to `build.New`. It needs no entry in the
registry. `Example_ownProfile` builds a package this way, and the godoc of
`build.MetadataModel` and `build.Definition` lists what each part must do. The
library is written for UGent's use first, and makes no promises to other
institutions
([ADR-0033](docs/decisions/0033-ugent-first-profiles-of-your-own.md)).

## Documentation

[CONTRIBUTING.md](CONTRIBUTING.md) has the tests, validation scripts and
conventions for working on SIP Creator itself. [docs/README.md](docs/README.md)
says which document answers which question: the design, the decisions and the
backlog.

## License

[Apache 2.0](LICENSE).
