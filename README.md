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

## Configuration

Create a `.env` file with these variables:

```
- `SIP_FILE_FORMAT_NAME` (**required**, non-empty) - The name of the format identification tool (Valid values: `siegfried`)
- `SIP_FILE_FORMAT_COMMAND` (**required**, non-empty) - The location of the binary of of the format identification tool on your system.
- `SIP_FILE_FORMAT_ARGS` (**required**, non-empty) -  Extra arguments passed to the format identification tool.
```

**Siegfried**

Configure Siegfried like this:

```
SIP_FILE_FORMAT_NAME="siegfried"
SIP_FILE_FORMAT_COMMAND="/location-of-sf-binary"
SIP_FILE_FORMAT_ARGS="-hash md5 -json"

## How to use

Assuming you have data in a `./tmp/basic` directory which you want to convert into a SIP package
stored in a `basic-uuid` directory:

```
go build
./sip-creator create --profile basic ./tmp/basic basic-uuid
```

## Input

The input needs to adhere to these requirements:

**Basic profile**

* There must be exactly one `dc+schema.json` file.
* The metadata is serialized in `JSON-LD`. 
  * The `@context` property refers to these two ontologies:
    * `dcterms`: `http://purl.org/dc/terms/`
    * `schema`: `http://schema.org/`
* There must be exactly one directory called `representation_1`. 
  * There must be at least one file in the `representation_1 ` directory.