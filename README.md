[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)

# SIP Creator

Create Submission Information Packages (SIP) based on [Meemoo's SIP Specification](https://developer.meemoo.be/docs/diginstroom/sip/)

:warning: **This is an experimental package** :warning:

## Features

* Implements Meemoo SIP Specification v2.0
* Provides support for the `basic` profile

## How to use

Assuming you have data in a `./tmp/basic` directory which you want to convert into a SIP package
stored in a `basic-uuid` directory:

```
go build
./sip-creator create --profile basic ./tmp/basic basic-uuid
```