[![Go Reference](https://pkg.go.dev/badge/github.com/ugent-library/sip-creator.svg)](https://pkg.go.dev/github.com/ugent-library/sip-creator)

# SIP Creator

Create Submission Information Packages (SIP) based on [Meemoo's SIP Specification](https://developer.meemoo.be/docs/diginstroom/sip/)

:warning: **This is an experimental package** :warning:

## Features

* Implements [Meemoo SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/)
  (the stable version — 2.0/2.1 are release candidates) and plain
  [E-ARK SIP](https://earksip.dilcis.eu/) 2.2.0.
* Profiles are registry entries selected with `--profile`: `basic` (meemoo SIP 1.2) and
  `eark` (plain E-ARK SIP for RODA-class repositories). An unknown (or omitted)
  `--profile` fails with the list of available profiles.
* Optional file format characterization via a pre-computed [Siegfried](https://github.com/richardlehane/siegfried)
  report: a `siegfried.json` sidecar in the input folder enriches the PREMIS metadata with
  PRONOM format info (a SHOULD in the meemoo spec); without it the build still succeeds —
  fixity is computed natively. The tool never runs Siegfried itself (see Format characterization).

## Requisites

* For the validation loop: [Docker](https://www.docker.com/) (runs the commons-ip validator and the report server) and `jq`.
* Recommended: [Siegfried](https://github.com/richardlehane/siegfried) to generate the characterization sidecar (optional, see Format characterization).

## Configuration

Configuration is read from the environment (a `.env` file is loaded when present — start
from `.env.example`). All environment variables are documented in
[CONFIG.md](CONFIG.md), which is generated from the config struct — regenerate it with
`go generate ./cli` rather than editing it by hand.

**Submitting organization** (required for `create`)

Every package's METS names the organization submitting it, so `SIP_SUBMITTER_NAME` is
required for **every** profile — including `eark`, which builds with the name alone.
meemoo profiles (`basic`) additionally require `SIP_SUBMITTER_OR_ID` — the organization's
identifier in [meemoo's organization register](https://developer.meemoo.be/) — which is
emitted as the agent's `IDENTIFICATIONCODE` note (meemoo SIP 1.2); other profiles ignore
it. A build refuses to run when a value its profile requires is missing, rather than
emitting a package that would be rejected at ingest:

```
SIP_SUBMITTER_NAME="Universiteitsbibliotheek Gent"
SIP_SUBMITTER_OR_ID="OR-a1b2c3d"
```

**Format characterization** (optional, input — not configuration)

Format info comes from a pre-computed Siegfried report placed next to your input, not
from an installed tool: generate it **from the input root** and the build picks it up
by name:

```sh
cd ./your-input && sf -hash md5 -json . > siegfried.json
```

Without a `siegfried.json` the build succeeds with no format info (`premis:format` is a
SHOULD; checksums and sizes are always computed natively). When the sidecar is present it
is strictly verified: a malformed report, an essence file missing from it, a report made
without `-hash md5`, or a file changed since the report was generated aborts the build —
a stale format claim is worse than none. Regenerate the sidecar whenever the input
changes.

Migration note: the former `SIP_FILE_FORMAT_*` environment variables are gone and
silently ignored if they linger in your `.env` — the tool no longer executes Siegfried
(see [ADR-0009](docs/decisions/0009-characterization-as-sidecar-input.md)).

## How to use

Assuming you have data in a `./tmp/basic` directory which you want to convert into a SIP package
stored in a `basic-uuid` directory:

```
go build -o bin/sip-creator .
./bin/sip-creator create --profile basic ./tmp/basic basic-uuid
```

This writes the package directory `basic-uuid/uuid-<uuid>/` and zips it (uncompressed) to
`basic-uuid/uuid-<uuid>.zip`. Pass `--no-zip` to skip the zip when the package directory
itself is what you need next.

Further flags:

* `--content-category` sets the package's content category (`mets/@TYPE`,
  CSIP vocabulary), overriding `SIP_CONTENT_CATEGORY` and the profile default.
* `--status` sets the record status (SIP3 vocabulary: `new`, `supplement`,
  `replacement`, `test`, `version`, `delete`; omitted means `new`). A status
  that updates an earlier package requires `--updates <identifier>`: the
  original package's identifier is reused as this package's identifier
  (`mets/@OBJID`) — how a conformant archive matches the update to its
  holdings.

**Delivering to meemoo (basic profile):** meemoo's transfer format wraps the SIP in a
BagIt bag — an envelope this tool deliberately does not produce
([ADR-0008](docs/decisions/0008-bag-layer-out-of-scope.md)). Bag the package *directory*
(not our zip) with a reference BagIt implementation, then follow meemoo's transfer
instructions:

```
./bin/sip-creator create --profile basic --no-zip ./tmp/basic basic-uuid
bagit.py --md5 basic-uuid/uuid-<uuid>/
```

**Delivering to a RODA-class repository (eark profile):** the zip is the deliverable;
ingest it directly.

## Validation

`./build.sh [profile]` (default `basic`) is the local CI loop: it rebuilds, regenerates
the sample SIP from `tmp/<profile>`, validates the zip with
[commons-ip](https://github.com/keeps/commons-ip) (dockerized, release jar pinned),
prints every FAILED check with its messages, and exits non-zero if the package is not
`VALID`. Each profile validates against the E-ARK spec version of its era: `basic`
(meemoo 1.2) against 2.0.4, `eark` against 2.2.0. Both are expected to report `VALID`.

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

One folder is one package, prepared per the
[input specification](docs/input-spec.md) — the authoritative statement of
every rule. In short:

* **`metadata.csv`** (required): the descriptive metadata as two-column
  `key,value` rows — a closed vocabulary of Dublin Core terms (see the
  spec's key table). `identifier` and `title` are always required; meemoo
  profiles also require `description` and `created`, and a Dutch entry
  wherever a language-tagged key is used (`title[nl]`).
* **Content**: either flat in the folder (one representation), or one folder
  per version under `representations/<your-name>/` — names are free-form
  labels (letters, digits, `._-`); the tool decides the package-side naming
  and keeps your label as the human-readable name.
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

## Documentation

[docs/](docs/) holds the project documentation: [docs/sip-creator-design.md](docs/sip-creator-design.md)
describes the system as it is today, [docs/decisions/](docs/decisions/) records why key
choices were made, and [docs/TODO.md](docs/TODO.md) is the live backlog. Start with
[docs/README.md](docs/README.md) for how it all fits together.