# SIP Creator agent orientation

SIP Creator is a Go CLI that builds Submission Information Packages (SIPs) conforming to [meemoo's SIP Specification v2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/) for ingest into the Flemish heritage archive (hetarchief.be). It takes a source directory of essence files plus JSON-LD descriptive metadata and produces a standards-compliant package: METS 1.12 structure following the E-ARK CSIP profile (DILCIS `csip:`/`sip:` extensions), PREMIS v3 preservation metadata, Dublin Core/DCTERMS + schema.org descriptive metadata (JSON-LD in, XML out), EDTF dates, embedded XSD schemas, all zipped uncompressed. Packaging is METS/CSIP-based; bagit is not used.

This is digital-preservation software. Spec compliance and valid, correct XML matter more than features or clever code. When in doubt, the meemoo SIP spec and the CSIP profile win over convenience.

The project is **experimental and work in progress**. `TODO.txt` is the live design record: it lists open design questions and known defects. Read it before "fixing" something — it may be a known, deliberate, or already-diagnosed issue.

Keep the dependency footprint small and boring. All XML is generated with Go `text/template` templates, not an XML library — keep it that way unless there is a strong, discussed reason to change. Adding a new external dependency is a serious choice, not a convenience.

## Audience

SIP Creator serves digital-preservation staff at UGent Library preparing SIPs for meemoo ingest. Downstream consumers are meemoo's ingest pipeline, CSIP validators (commons-ip / RODA), and the future archivists and systems that will read the preserved metadata decades from now.

Use vocabulary from the meemoo SIP spec and OAIS: SIP, essence, representation, intellectual entity, fixity, descriptive vs. preservation metadata.

## System shape

Single-binary cobra CLI. Entry point is `main.go` → `cli/cli.go` → `cli/create_cmd.go`.

Data flow for `create [src] [dest] --profile basic`:

1. **Input**: a source dir (e.g. `./tmp/basic`) with exactly one `dc+schema.json` (JSON-LD descriptive metadata) and one or more `representation_N/` directories holding essence files.
2. **Profile** (`profiles/profile.go`, `profiles/basic.go`): creates a `uuid-<uuid>` package dir under `dest`, builds the skeleton (`metadata/descriptive`, `metadata/preservation`, `representations`, `schemas`), copies essence files, runs format identification per file, generates PREMIS and METS XML at representation and package level. `profiles/roda.go` is a second, WIP profile not yet wired into the `create_cmd.go` switch.
3. **Encoders** (`encoders/metadata`, `encoders/mets`, `encoders/premis`): thin `Encode*(io.Writer, ...)` APIs backed by `text/template`. UUID minting for METS IDs lives in the mets `idStore`.
4. **Schemas** (`schemas/schemas.go`): all XSDs are bundled via `//go:embed` and copied into each SIP's `schemas/` dir.
5. **Archive** (`archive/zip.go`): zips the package dir into `dest/uuid-<uuid>.zip`, stored uncompressed (`zip.Store`).

Supporting pieces:

- **Domain model** (`sip/`): `Package`, `Entity`, `Representation`, `File`, `Identifier`. All identifiers take the form `uuid-<uuid>`. `sip.Event` is an empty stub.
- **Format identification** (`formats/`): pluggable registry (`Identificator` interface, `Register`/`New` factory). `formats/siegfried` self-registers and shells out to the external `sf` binary, which must be installed on the system.
- **Config** (`services/config.go`): `.env` loaded via godotenv, parsed with caarlos0/env. Required vars: `SIP_FILE_FORMAT_NAME`, `SIP_FILE_FORMAT_COMMAND`, `SIP_FILE_FORMAT_ARGS`.

## Read the right docs

- [README.md](README.md) — usage, configuration, input requirements, experimental-status warning.
- [TODO.txt](TODO.txt) — open design questions and known defects. Check here first when investigating a bug.
- [CONFIG.md](CONFIG.md) — environment variables. This file is **generated** by envdoc; regenerate with `go generate ./services`, never hand-edit.
- External specs: [meemoo SIP spec 2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/), [E-ARK CSIP profile](https://earkcsip.dilcis.eu/), [METS](https://www.loc.gov/standards/mets/) and [PREMIS](https://www.loc.gov/standards/premis/) at loc.gov.
- Concrete examples: `tmp/basic/` is a sample input tree; `basic-uuid/` is sample generated output. Both are local fixtures, not tracked in git.

## Non-negotiables

### Documentation

- `AGENTS.md` and `CLAUDE.md` must always be identical. A change made to either file must be reflected in both.
- Keep docs current in the same change as code: new or changed env vars require `go generate ./services` to regenerate `CONFIG.md`; changed usage or input requirements go in `README.md`; resolved items get removed from `TODO.txt`.

### Git

- Commit messages use a change-type prefix: `Added:`, `Changed:`, `Fixed:`, `Removed:` — capitalized, followed by a colon and a short summary. This matches existing history.
- Keep commits small and focused. Direct commits to `main` are acceptable in this experimental phase.
- No rebase or force-push of pushed history unless the user explicitly directs it.
- Never commit `.env`, `tmp/`, `basic-uuid/`, or other local fixtures and generated output.

### Code

- Generated XML must validate against the embedded XSDs and pass commons-ip CSIP validation. Validation output from `build.sh` is the acceptance check.
- Keep XML generation in `text/template` templates inside `encoders/*`. Do not introduce `encoding/xml` or an XML library for output without discussion.
- All identifiers are `uuid-<uuid>`; METS ID minting stays in the mets `idStore`.
- New format identification tools register via the `formats` registry pattern, following `formats/siegfried`.
- New SIP profiles are methods on `Profile` in `profiles/` and must be wired into the `create_cmd.go` profile switch.
- Existing profile/IO code uses `panic(err)` — acknowledged debt. New code should return errors; do not add new panics.
- Keep the exported API surface small. Every exported symbol is a long-term commitment.

## Development commands

- `go build` — produces the `./sip-creator` binary.
- `./sip-creator create --profile basic ./tmp/basic basic-uuid` — generate a sample SIP from the local fixture.
- `./build.sh` — the full dev loop: wipes `basic-uuid/`, regenerates via `go run`, validates every produced package with commons-ip (`csip validate`), and surfaces `FAILED` checks. Requires `csip`, `jq`, and `catmandu` on the PATH, plus a configured `.env` pointing at a working Siegfried (`sf`) install.
- `go generate ./services` — regenerate `CONFIG.md` from the config struct.

There are no `*_test.go` files yet; external CSIP validation via `build.sh` is the current acceptance check. Add Go tests for new logic where practical.
