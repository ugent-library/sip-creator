[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

# SIP Creator

SIP Creator packages your content files and their descriptive metadata into an
[E-ARK](https://earksip.dilcis.eu/) Submission Information Package (SIP), ready to hand
to a digital archive. It is a Go library, and a command-line tool built on it.

Archives differ in what they expect inside a SIP. A profile is a content type, defined
by the archive or institution that owns its rules: the specification version, the
descriptive metadata standard, and the rules your input must meet. Every profile builds
an E-ARK SIP. Three are included:

* `ugent/basic`, for UGent Library's RODA: resources the library has not necessarily
  catalogued, such as a database dump or a deposited set of files, described in Simple
  Dublin Core;
* `ugent/bibliographic`, for UGent Library's RODA: the library's catalogued holdings,
  described in MODS from the catalogue record;
* `meemoo/basic` follows Meemoo's SIP specification, with Meemoo's own mix of Dublin
  Core terms and schema.org, for ingest into Meemoo, the Flemish heritage archive.

See [Profiles](#profiles) for what each one requires.

:warning: **This is an experimental package** :warning:

## Features

* Builds a complete package from a plain input folder: your content files plus a simple
  descriptive rows file (`description.csv`) or, where the profile allows it, a finished
  descriptive document. Out comes a SIP with generated METS and PREMIS metadata and
  natively computed checksums.
* Validates an input folder before building (`check`, with the same `--profile` as
  `create`), reporting every violation of the input rules at once.
* Optional PRONOM format identification based on a pre-computed
  [Siegfried](https://github.com/richardlehane/siegfried) report (see Format characterization).
* A profile of your own, for a program that embeds the library (see
  [Profiles of your own](#profiles-of-your-own)).

## Profiles

Choose a profile with `--profile` on the command line, or with `profiles.Get` in Go.

| | `ugent/basic` | `ugent/bibliographic` | `meemoo/basic` |
|---|---|---|---|
| Use for | uncatalogued resources, into UGent's RODA | catalogued holdings, into UGent's RODA | ingest into Meemoo (hetarchief.be) |
| Specification | E-ARK SIP 2.2.0 | E-ARK SIP 2.2.0 | Meemoo SIP 1.2, on E-ARK SIP 2.0.4 |
| Descriptive standard | Simple Dublin Core | MODS 3.7 | Meemoo's Dublin Core and schema.org |
| [`description.csv` keys](docs/input-spec.md#3-descriptive-metadata-descriptioncsv-or-a-supplied-document) | the 15 Dublin Core elements | `identifier`, `title` | Meemoo's key table |
| Required keys | `identifier`, `title` | `identifier`, `title` | `identifier`, `title`, `description`, `created` |
| [Finished document](docs/input-spec.md#supplying-a-finished-document-ugentbasic-and-ugentbibliographic) accepted | `dc.xml` | `mods.xml` | none |
| Go description type | `simpledc.Terms` | `mods.Record` | `meemoo.Terms` |
| Submitter | name | name | name and Meemoo OR-id |
| Representation type | written to the METS | written to the METS | ignored |
| You deliver | the zip | the zip | the package directory, in a BagIt bag |

### `ugent/basic`: UGent Library, uncatalogued resources

The rules are written in [docs/profiles/ugent-basic.md](docs/profiles/ugent-basic.md).
Builds a plain [E-ARK SIP](https://earksip.dilcis.eu/) 2.2.0 with a Simple Dublin Core
document (`dc.xml`) as its descriptive metadata. `description.csv` takes the fifteen
Dublin Core elements (`title`, `creator`, `date`, `coverage`, ...) as keys. A language
tag on a key is accepted but not written into the document.

Each representation's type goes into that representation's METS content typing
(`csip:OTHERTYPE` and `csip:OTHERCONTENTINFORMATIONTYPE`); RODA v5.7.0 and later shows it
in the Type column of the AIP's representations. The zip the tool writes is the
deliverable: ingest it as it is.

### `ugent/bibliographic`: UGent Library, catalogued holdings

The rules are written in
[docs/profiles/ugent-bibliographic.md](docs/profiles/ugent-bibliographic.md). The same
package as `ugent/basic`, with a [MODS 3.7](https://www.loc.gov/standards/mods/) document
(`mods.xml`) written from the catalogue record. Everything except the descriptive
metadata works as under `ugent/basic`.

MODS is a tree, so `description.csv` holds only `identifier` and `title`. A richer
record, such as one listing the library's physical copies of the work (call number,
barcode, volume), comes as a finished `mods.xml`, or in Go as a `mods.Record` with
`Items`. Physical copies belong on the package-level record, because a representation is
a version of the content, never a copy. The library does not refuse items on a
representation's record; it writes them to that representation's `mods.xml`.

### `meemoo/basic`: Meemoo SIP 1.2

Builds an E-ARK SIP (2.0.4) that also conforms to the basic content profile of the
[Meemoo SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/),
for ingest into the Flemish heritage archive. `description.csv` takes Meemoo's closed
key table: Dublin Core terms (`created`, `spatial`, `extent`, ...) plus two schema.org
properties. On top of the four required keys, Meemoo requires a Dutch value (`[nl]`)
wherever a language-tagged key is used.

There is no finished-document route: Meemoo's document must carry the package identifier
the tool mints, and the tool does not edit XML. The submitter needs Meemoo's OR-id as
well as a name (see [Configuration](#configuration)). A representation's type is
ignored, because Meemoo SIP 1.2 fixes every METS content typing to `OTHER` plus the
profile URI; its label still becomes `mets/@LABEL`.

Meemoo's transfer format wraps the SIP in a BagIt bag, which this tool does not produce.
Build a package directory with `--no-zip`, bag that directory with a reference BagIt
implementation, and follow Meemoo's transfer instructions:

```
./bin/sip-creator create --profile meemoo/basic --no-zip ./your-input sip-out
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

Assuming you have data in a `./your-input` folder (prepared as described under
[Input folder](#input-folder)) which you want to convert into a SIP package stored in a
`sip-out` directory (with the submitting organization set, see
[Configuration](#configuration)):

```
./bin/sip-creator create --profile ugent/basic ./your-input sip-out
```

This writes the package directory `sip-out/uuid-<uuid>/` and zips it (uncompressed) to
`sip-out/uuid-<uuid>.zip`. To try it before preparing your own input, build one of the
[example folders](examples/): `./bin/sip-creator create --profile ugent/basic examples/ugent/basic sip-out`.

Further flags:

* `--content-category` sets the package's content category (`mets/@TYPE`,
  CSIP vocabulary), overriding `SIP_CONTENT_CATEGORY` and the profile default.
* `--status` sets the record status (SIP3 vocabulary: `new`, `supplement`,
  `replacement`, `test`, `version`, `delete`). Without it the METS carries no
  status, which the E-ARK SIP specification reads as `new`. A status
  that updates an earlier package requires `--updates <identifier>`: the
  original package's identifier is reused as this package's identifier
  (`mets/@OBJID`). Build the update into a destination that does not
  still hold the original: an existing package directory or zip with that
  identifier is refused, never written into.
* `--no-zip` to skip zipping when the package directory itself is what you need.

What you deliver depends on the profile: the zip for `ugent/basic` and `ugent/bibliographic`, a bagged
package directory for `meemoo/basic` (see [Profiles](#profiles)).

### Checking an input folder

To check a folder without building anything, pass the same profile. `check` reports
every violation of the input rules at once, in plain language. When the folder passes, it
then checks the profile's own rules, such as `meemoo/basic`'s single representation. It reads
no configuration: no `.env` and no environment variables:

```
./bin/sip-creator check --profile ugent/basic ./your-input
```

The report goes to stdout, so it can be saved to a file. It lists every problem first,
then a summary of what the tool read, then the verdict:

```
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

The summary says where the package description comes from: the rows of
`description.csv`, from which the tool generates a document, or a supplied `dc.xml` or
`mods.xml`, which it copies as it is.

The exit status tells a script what happened:

| status | meaning |
|---|---|
| 0 | the folder meets the input specification and the profile's rules |
| 1 | the folder has problems; the report lists them |
| 2 | check could not check the folder: a wrong path, a file instead of a folder, an unknown profile, or a missing argument |

### Configuration

Configuration is read from the environment. A `.env` file is loaded when present; start
from `.env.example`. All environment variables are documented in
[CONFIG.md](CONFIG.md).

Every package's METS names the organization submitting it, so `create` requires
`SIP_SUBMITTER_NAME` for every profile. `meemoo/basic` also requires `SIP_SUBMITTER_OR_ID`, the
organization's identifier in [Meemoo's organization register](https://developer.meemoo.be/),
emitted as the agent's `IDENTIFICATIONCODE` note. A build refuses to run when a value its
profile requires is missing, rather than emitting a package that would be rejected at
ingest:

```
SIP_SUBMITTER_NAME="Example Organization"
SIP_SUBMITTER_OR_ID="OR-a1b2c3d"
```

### Input folder

One folder is one package. Your content files live in a representation folder under
`representations/`, one folder per version of the content. The smallest valid input is a
`description.csv` plus one representation:

```
your-input/
├── description.csv
└── representations/
    └── archival/
        ├── scan-001.tif
        └── scan-002.tif
```

When the content comes in several versions, such as scanned master copies and an access
copy, each version gets its own folder (under the UGent profiles; Meemoo's
`meemoo/basic` profile allows one representation and no description below the package
level). Under the UGent profiles a representation folder is named `preservation`,
`archival` or `access`, and the name is also its type:

```
your-input/
├── description.csv
├── representations.csv
├── siegfried.json
├── documentation/
├── premis/
└── representations/
    ├── archival/
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
| `dc.xml`, `mods.xml` | instead of `description.csv`, where the profile accepts one | a finished descriptive document | [§3](docs/input-spec.md#supplying-a-finished-document-ugentbasic-and-ugentbibliographic) |
| `representations/<name>/` | yes | one folder per version of the content | [§2](docs/input-spec.md#2-content-files-and-representations) |
| `representations.csv` | no | a label and type per representation folder | [§2](docs/input-spec.md#representationscsv-labels-and-types-optional) |
| `documentation/` | no | context material; commons-ip warns when a representation has none | [§4](docs/input-spec.md#4-documentation) |
| `premis/` | no | received preservation XML, copied as it is | [§5](docs/input-spec.md#5-received-preservation-files-premis) |
| `siegfried.json` | no | a format characterization report | [below](#format-characterization) |

Content lives only in the representation folders: anything else at the top level is an
error. A representation folder can hold its own `documentation/` and
`premis/`, and under the UGent profiles its own `description.csv` (or document), about
that version only. Representation folder names may use letters, digits and `._-`.

[examples/](examples/) has a complete input folder for each profile.

#### `description.csv`

A two-column file with a `key,value` header row. Repeat a key for more values, and add a
language in square brackets where it matters (`title[nl]`). An unknown key is an error, so
a typo cannot silently drop metadata. Which keys exist and which are required depends on
the profile (see [Profiles](#profiles)). The smallest valid file per profile:

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

A finished document replaces the file at the same level. The tool checks only that it is
XML with the root element the profile expects, and copies it as it is.

#### `representations.csv`

Gives each representation folder a display label and a type. `folder` is required; an
empty `label` means the folder name, an empty `type` means the label. Under the UGent
profiles the type is the folder name, so leave `type` out or repeat the name:

```csv
folder,label
archival,Scanned master copies (TIFF)
access,Access copy (JPEG)
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

Without the report, the package has no format info, and the tool computes every
checksum itself. With it, the build stops when the report is malformed, made without
`-hash md5`, or misses a content file. The report's MD5 is the checksum the package
declares for each file it covers, taken as given and not checked against the file
(ADR-0032): the build is much faster, and keeping the report true to the files is up to
you. Generate it right before the build, from the folder as you will build it.

## Go library

The command-line tool is one program built on this library. In your own program you
supply as Go values what the tool reads from the input folder and the environment, and
the library reads no environment variables:

* the submitting organization is added to the profile with `WithSubmitter`;
* the destination directory and the logger go in `build.Config`; without a logger, the
  progress messages are discarded;
* the descriptive metadata and the content files go in `build.SourcePackage`.

The full API is on [pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator).
Runnable examples for each case below are in [build/example_test.go](build/example_test.go);
`go test` runs them, and pkg.go.dev shows them with the
[build package](https://pkg.go.dev/github.com/ugent-library/sip-creator/build#pkg-examples).
The domain model and build lifecycle are described in
[docs/sip-creator-design.md](docs/sip-creator-design.md).

### Building a package

Resolve a profile, attach the submitter, and hand `Build` your descriptive terms and
content files:

```go
import (
	"log/slog"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
)

def, ok := profiles.Get("ugent/basic")
if !ok {
	// ...
}
// The second argument is the Meemoo OR-id, used by basic only.
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
	Description: simpledc.Terms{
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

`Build` validates the source package against the profile's rules, then writes the complete
package directory under `Destination` and returns the built package. Zipping is a separate
step (the `archive` package). A builder is constructed once per profile; the values that
change per package go in the `SourcePackage`. Besides the description and the
representations, it takes `Documentation` and `Premis` files, the library's form of the
input folder's `documentation/` and `premis/`.

### Descriptive metadata

The `ugent/bibliographic` profile takes a `mods.Record` instead of a list of terms: an
identifier, titles, and physical copies as items, each a call number with an optional
barcode and an optional volume or issue designation (example: `ExampleBuilder_Build_mods`).

A record that already exists as a document travels as a file: profiles that accept a
finished document (see [Profiles](#profiles)) take a `build.EncodedDescription` in place
of terms or a record, naming the file in its `Source` field. The library checks and
copies it the same way the command-line tool does (see [Input folder](#input-folder);
example: `ExampleEncodedDescription`).

### Representation labels and types

Besides the required `Name` (the directory under `representations/`), each
representation takes two optional fields. `Label` is the display name, emitted
as the representation METS `mets/@LABEL`; empty means the `Name`. `Type` is the
representation's type; empty means the `Label`. Whether the type reaches the METS
depends on the profile (see [Profiles](#profiles)). These are the fields the
command-line tool fills from `representations.csv`.

### Updating an earlier package

The example above builds a new package with the profile's METS values. Three fields
on the `SourcePackage` override them per package (example: `ExampleBuilder_Build_update`):

* `PackageIdentifier`: the identifier of the package this one updates, reused as this
  package's `mets/@OBJID`. A status that updates an earlier package requires it.
* `RecordStatus`: `metsHdr/@RECORDSTATUS`, one of the `sip.RecordStatus` constants (the
  SIP3 vocabulary). Without it the METS carries no status, which the E-ARK SIP
  specification reads as new.
* `ContentCategory`: `mets/@TYPE`, the CSIP content category.

### Format characterization

Set `Characterization` on the `SourcePackage` to add format info. Decode a Siegfried
report with `characterization.DecodeSiegfried`; the library verifies it as strictly as
the command-line tool does (see [Format characterization](#format-characterization)).

### Profiles of your own

The three profiles above are written the same way a profile of your own would be, and
the engine imports none of them. A program that embeds the library can keep a profile
of its own, for a descriptive standard the three do not cover, in a Go package with
three parts:

* a description type implementing `sip.Description`, whose `Validate` and
  `ValidateRequired` are the rules of your standard;
* a metadata model implementing `build.MetadataModel`: `ValidateType`, which refuses a
  description of another type, `Encode`, which writes the document (the profiles here
  use `text/template`), `Schemas`, the XSDs the document points at with their
  contents, and
  `ModelType` and `ModelTypeVersion`, which name the document's format and its version
  for the METS dmdSec (`MDTYPE`, or `OTHERMDTYPE`, and `MDTYPEVERSION`);
* an exported `build.Definition` naming the model, the document's file name and the
  METS values (`sip.MetsDeclaration`: profile URL, content typing, any agents your
  archive asks for). The engine adds the software agent itself, and `WithSubmitter`
  the submitting organization. The model type itself can stay unexported.

Hand that definition to `build.New` as above. Your package supplies its own XSDs: embed
them with `go:embed` and return each as a `build.Schema`, a file name and its contents.
The package ships them in `schemas/` next to the ones its METS documents need. Where your
document uses a schema this repository bundles, such as `xml.xsd`, `build.BundledSchemas`
returns it. A schema name must be a plain file name, its contents must not be empty, and
a name the METS schemas already use must come with the same contents. Your package does
not need to be added to the registry in `profiles/`: the registry only lists the names
`--profile` accepts on the command line, and `build.New` takes any definition. The
example `Example_ownProfile` in [build/example_test.go](build/example_test.go) builds a
package with a profile defined entirely outside `profiles/`.

`ModelType` returns your format's name. A name the METS `MDTYPE` vocabulary lists, such
as `DC` or `MODS`, is written as `MDTYPE`; any other name, such as `EBUCore`, is written
as `MDTYPE="OTHER"` with the name in `OTHERMDTYPE`, as the example shows.

When your profile leaves a value to the program that builds the description, such as the
`type` of a MODS identifier, offer a fixed set of typed constants rather than free text.
Two programs that make the same choice then write the same value into the document
(ADR-0022).

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