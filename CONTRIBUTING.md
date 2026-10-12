# Contributing to SIP Creator

This guide is for working on SIP Creator itself. For using the tool or the library, see
the [README](README.md).

## Requirements

* [Go](https://go.dev/dl/) 1.27 or later, for building and for `go test ./...`.
* For `build.sh` and the validation scripts:
  * Java 17 or later on your `PATH`, which runs the commons-ip validator;
  * `curl`, which fetches the commons-ip jar once;
  * `jq`;
  * `sf` ([Siegfried](https://github.com/richardlehane/siegfried)) on your `PATH`,
    because `build.sh` generates a `siegfried.json` for the input before building;
  * `xmllint` (part of libxml2, present on macOS and most Linux systems), which checks
    every descriptive document in the package against the schema the package ships.

`scripts/fetch-commons-ip.sh` pins the commons-ip version and the jar's sha256, downloads
the jar into `tmp/commons-ip/` on the first run and verifies the checksum on every run
([ADR-0035](docs/decisions/0035-commons-ip-runs-on-host-java.md)). To validate with
another commons-ip version, for example the one RODA bundles, set `CSIP_CMD` to the
command to run instead:

```sh
CSIP_CMD="java -jar commons-ip2-cli-2.8.0.jar" ./build.sh ugent/basic
```

## Building

```sh
go build -o bin/sip-creator .
./bin/sip-creator create --profile meemoo/basic examples/meemoo/basic tmp/output/meemoo/basic
```

`bin/` and `tmp/` are ignored by git.

## Tests

```sh
go test ./...
```

The Go tests need none of the tools above: no Java, no `sf`, no `.env`. They include
a build of every folder in [examples/](examples/), so a change that makes an example
invalid fails the tests.

The GitHub Actions workflow in `.github/workflows/test.yml` runs `gofmt -l .`,
`go vet ./...` and `go test ./...` on every push to `main` and every pull request.

## Validating generated packages

`./build.sh [profile] [input]` is the local CI loop. The profile defaults to `meemoo/basic` and
the input to [`examples/<profile>`](examples/); pass another input folder to validate
your own. It:

1. rebuilds the binary;
2. copies the input to `tmp/build/<profile>`, generates its `siegfried.json` there, and
   builds the package into `tmp/output/<profile>`, so the input folder itself is never
   changed;
3. validates the zip with [commons-ip](https://github.com/keeps/commons-ip), fetched and
   checksum-verified by `scripts/fetch-commons-ip.sh`, printing every failed check with
   its messages;
4. runs `xmllint` over every descriptive document in the package against the schema the
   package ships for it, offline through the XML catalog in `scripts/schema-catalog.xml`,
   because commons-ip does not validate the descriptive documents the METS points at:
   `mods.xml` against `mods-3-7.xsd`, `dc.xml` against `simpledc.xsd`, and
   `dc+schema.xml` against Meemoo's `descriptive_basic.xsd`. Meemoo's schema checks
   element names only; the cardinality and required elements of Meemoo SIP 1.2 are
   checked by the library, and EDTF dates by nothing yet (see docs/TODO.md).

It exits non-zero when the package is not `VALID` or a descriptive document is not
valid against its schema. Each
profile validates against its E-ARK spec version: `meemoo/basic` (Meemoo 1.2) against 2.0.4,
`ugent/basic` and `ugent/bibliographic` against 2.2.0. Every registered profile is expected to report `VALID`.

To validate a package without representations under a UGent profile, pass a folder that
holds only a `description.csv` as the second argument, for example
`./build.sh ugent/basic path/to/folder`. It reports `VALID` with two SHOULD-level
warnings from commons-ip (CSIPSTR11 and CSIPSTR13), explained in
[docs/TODO.md](docs/TODO.md#validator-status).

Each run's reports are written to `reports/runs/<timestamp>-<profile>/`. To browse them
as HTML (run history, per-check detail, links into the E-ARK specs), serve `reports/`
with any static file server; the viewer loads the reports with `fetch`, which browsers
refuse from a page opened straight from disk:

```sh
python3 -m http.server --directory reports 8080
open http://localhost:8080
```

`./scripts/validate.sh [-o report-dir] <sip.zip|sip-dir>...` validates any package on
its own, including an unzipped package directory when debugging structure.

## Comparing against the reference copy

```sh
./scripts/reference-diff.sh tmp/reference/<profile>/pkg tmp/output/<profile>/uuid-<uuid>
```

compares a generated package with a reference copy kept in `tmp/reference/`, after
replacing the values that change on every run or per environment: UUIDs, timestamps,
the checksums and sizes of generated XML files, and the submitting organization. The
script's header lists what it normalizes and what it never does. A refactor must
produce no difference. When a change alters the output on purpose, replace the
reference copy and record why in the commit message and in `tmp/reference/README.md`.
`tmp/` is local and not tracked in git. The reference copies contain format info, so
build the package to compare with `build.sh`, which generates `siegfried.json`.

## Documentation

Update the docs in the same change as the code:

* a new or changed environment variable: run `go generate ./cli` to regenerate
  [CONFIG.md](CONFIG.md), which is never edited by hand;
* changed usage or input requirements: [README.md](README.md) and
  [docs/input-spec.md](docs/input-spec.md);
* changed library usage: the runnable examples in
  [build/example_test.go](build/example_test.go), which pkg.go.dev shows;
* a changed design: [docs/sip-creator-design.md](docs/sip-creator-design.md), and an ADR
  in [docs/decisions/](docs/decisions/) when the change records a decision;
* a renamed or removed exported symbol: a dated note under the Status line of every ADR
  whose decision text names it, giving the current name. ADRs are frozen, so the note is
  how a reader who greps for the old name finds the new one;
* a new profile: every file
  [docs/development-adding-a-profile.md](docs/development-adding-a-profile.md) lists;
* a resolved item: remove it from [docs/TODO.md](docs/TODO.md);
* a change an operator or a program that embeds the library notices: a line under
  `Unreleased` in [CHANGELOG.md](CHANGELOG.md), under the heading that matches the
  commit's prefix (`Added`, `Changed`, `Fixed` or `Removed`).

[docs/README.md](docs/README.md) explains which document answers which question.
[docs/development-writing-style.md](docs/development-writing-style.md) holds the rules for
code comments and for the markdown in `docs/`.

## Releasing

A version is an annotated tag `vX.Y.Z` on `main`
([ADR-0036](docs/decisions/0036-versioned-releases.md)). While the major version is 0,
raise the minor version when the command line, the library API or a profile's
output changes, and the patch version for a fix that changes none of them.

1. Check `main`: `go test ./...` passes, `./build.sh <profile>` reports `VALID` for
   every profile, `./scripts/reference-diff.sh` shows no difference, and the GitHub
   Actions run of the last commit passed. Build with the latest patch release of the Go
   version `go.mod` names.
2. In [CHANGELOG.md](CHANGELOG.md), rename `Unreleased` to `X.Y.Z - YYYY-MM-DD` and add
   an empty `Unreleased` above it. Commit with `Changed: CHANGELOG.md names version
   X.Y.Z` and push `main`.
3. Tag the commit and push the tag:

   ```sh
   git tag -a vX.Y.Z -m "SIP Creator vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. Publish the release page with the version's section of the changelog:

   ```sh
   awk '/^## X\.Y\.Z /{f=1;next} /^## /{f=0} f' CHANGELOG.md > tmp/release-notes.md
   gh release create vX.Y.Z --title vX.Y.Z --notes-file tmp/release-notes.md
   ```

5. Check the result from outside the clone. In an empty temporary directory, with
   `GOPATH` and `GOMODCACHE` set to folders inside it, run
   `go install github.com/ugent-library/sip-creator@vX.Y.Z`. Its
   `sip-creator --version` prints `vX.Y.Z`. If pkg.go.dev does not list the version
   yet, open `https://pkg.go.dev/github.com/ugent-library/sip-creator@vX.Y.Z` and use
   its button to request it.

## Commits

Start the message with the kind of change, capitalized and followed by a colon:
`Added:`, `Changed:`, `Fixed:` or `Removed:`. Keep commits small and focused. Never
commit `.env`, `tmp/` or generated packages.
