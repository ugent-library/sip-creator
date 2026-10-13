# Changelog

This file lists the changes in each version of SIP Creator that an operator or a
program that embeds the library notices. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow
[semantic versioning](https://semver.org/). While the major version is 0, a
minor version may change the command line, the library API or the packages a
profile builds. Every package states the version that built it in the software
agent of its METS.

## Unreleased

The first version. SIP Creator builds an
[E-ARK](https://earksip.dilcis.eu/) Submission Information Package from a
producer's essence files and descriptive metadata.

### Added

- **The command-line tool.** `create` builds a package from an input folder
  that follows the [input specification][input-spec], and zips it unless
  `--no-zip` is given. `check` tests an input folder without building
  anything, and its exit status tells a script whether the folder has
  problems. `--version` prints the version.
- **Updates.** `--status` sets the record status, and `--updates` names the
  package this one updates and reuses its identifier. Under the UGent profiles
  an update states `UPDATE`, the status RODA updates an AIP on. It is not in
  the E-ARK vocabulary.
- **Format information.** A Siegfried report in `siegfried.json` adds PRONOM
  identifiers, and its MD5 checksums are taken as the package's fixity.
- **Three profiles:**
  - `meemoo/basic`: Meemoo SIP 1.2, basic content profile, with
    `dc+schema.xml`. Validated against E-ARK SIP 2.0.4.
  - `ugent/basic`: E-ARK SIP 2.2.0 with Simple Dublin Core (`dc.xml`).
  - `ugent/bibliographic`: E-ARK SIP 2.2.0 with MODS 3.7 (`mods.xml`).

  Under the UGent profiles a representation is `preservation`, `archival` or
  `access`, and RODA shows the `preservation` representation as the
  original.
- **The Go library.** Package `build` takes a profile and a source package and
  writes the package directory. Package `profiles` holds the registry,
  `archive` writes the zip, and `characterization` decodes a Siegfried report.
  A program can define a profile of its own. The API and the runnable
  examples are on [pkg.go.dev](https://pkg.go.dev/github.com/ugent-library/sip-creator).

### Known gaps

- `meemoo/basic` targets Meemoo SIP 1.2. A later version moves it to 2.1 and
  drops 1.2.
- No package has been ingested by Meemoo yet. The packages are validated with
  commons-ip 2.11.2 and xmllint only.
- EDTF dates under `meemoo/basic` are not checked: a `created` value that is
  not an EDTF date builds a package that commons-ip reports valid.
- The METS names no archival creator (`ARCHIVIST`), no contact persons and no
  submission agreement.
- PREMIS events are not written.
- The library takes essence as files on disk, not as streams, and a program
  cannot supply checksums except through a Siegfried report.
- The tool has not been run on Windows. There, a virus scanner can make the
  final rename of the package or the zip fail. A rerun works.

[docs/TODO.md][todo] has the details of each gap.

[input-spec]: https://github.com/ugent-library/sip-creator/blob/main/docs/input-spec.md
[todo]: https://github.com/ugent-library/sip-creator/blob/main/docs/TODO.md
