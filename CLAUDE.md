# SIP Creator agent orientation

SIP Creator is both a Go library and a CLI that builds Submission Information Packages (SIPs). It takes a producer's essence files plus descriptive metadata and assembles them into a standards-conformant package. Its primary target is a valid [E-ARK CSIP](https://earkcsip.dilcis.eu/) package — ingestible by RODA and other CSIP-conformant systems — with [meemoo's SIP Specification v2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/) layered on top as an additional specialization for ingest into the Flemish heritage archive (hetarchief.be).

## Scope

This is digital-preservation software at the front of the chain. In [OAIS](https://www.iso.org/standard/57284.html) terms (ISO 14721) a SIP is what a *Producer* hands to an archive's *Ingest* entity; SIP Creator is the tool that builds it. It is a **package builder, not an archive** — no permanent identifiers of its own authority, no store, no preservation guarantees: it produces one well-formed SIP and stops. Everything after ingest (AIP, storage, DIP dissemination) lives downstream in RODA and meemoo's pipeline.

A SIP bundles the *essence* (the content files) with the metadata to understand and trust it: **descriptive** (Dublin Core / MODS), **preservation** ([PREMIS](https://www.loc.gov/standards/premis/): provenance, fixity, events, agents), and **structural** ([METS](https://www.loc.gov/standards/mets/): the manifest tying files, checksums, and metadata together). Generating that XML correctly — valid references, accurate fixity, right profile declarations — is the whole job, so spec compliance and valid XML matter more than features or clever code. The E-ARK CSIP profile is the base contract every package must satisfy; meemoo SIP v2.0 specializes on top of it. When in doubt, CSIP validity wins over convenience, and the meemoo spec wins over convenience within the meemoo layer.

The project is **experimental and work in progress**. `docs/README.md` is the map of the documentation: `docs/sip-creator-design.md` describes *what the system is* (schema, domain model, lifecycle); `docs/decisions/` holds the ADRs recording *why* choices were made; coding and naming conventions live in this file. Keep `docs/` current: when a refactor changes the system, update `sip-creator-design.md` in the same change, and when it enacts a decision worth remembering, add or update an ADR. A doc that drifts from the code misleads the future reader, including future-you.

Keep the dependency footprint small and boring. All XML is generated with Go `text/template` templates, not an XML library — keep it that way unless there is a strong, discussed reason to change. Adding a new external dependency is a serious choice, not a convenience.

## Audience

SIP Creator serves digital-preservation staff at UGent Library preparing SIPs for meemoo ingest. Downstream consumers are meemoo's ingest pipeline, CSIP validators (commons-ip / RODA), and the future archivists and systems that will read the preserved metadata decades from now.

Use vocabulary from the meemoo SIP spec and OAIS: SIP, essence, representation, intellectual entity, fixity, descriptive vs. preservation metadata.

## System shape

Single-binary cobra CLI. Entry point is `main.go` → `cli/cli.go` → `cli/create_cmd.go`.

Data flow for `create [src] [dest] --profile basic`:

1. **Input**: a source dir (e.g. `./tmp/basic`) with exactly one `dc+schema.json` (JSON-LD descriptive metadata) and one or more `representation_N/` directories holding essence files.
2. **Profile** (`profiles/`): a profile is data, not code — `definition.go` holds `Definition` (descriptive source, local-identifier scheme, PREMIS emission flags, METS values as `sip.Spec`) plus the registry (`Get`/`Names`); the CLI resolves `--profile` there and an unknown value lists the available profiles. One engine reads it: `Builder.Build(def)` (`builder.go`) runs two phases with a hard seam. `assemble.go` is pure — it walks the input, runs format identification on the *source* files, and builds the complete `sip.Package` graph with zero disk writes; `write.go` emits the graph in the one canonical dependency order (skeleton → schemas → essence copies → descriptive → per-rep PREMIS then METS → package PREMIS → package METS strictly last), back-filling fixity onto graph nodes as each file lands. Errors return (no panics on the build path); a failed assembly leaves nothing on disk. A new profile is one registry entry.
3. **Store** (`store/store.go`): dumb filesystem primitives rooted at the `uuid-<uuid>` package dir under `dest` — callers speak package-relative paths. `CopyFile` computes MD5/size during the streamed copy; `WriteMetadata` renders to memory before writing (a failed template leaves no partial file); all writes truncate.
4. **Encoders** (`encoders/metadata`, `encoders/mets`, `encoders/premis`): thin `Encode*(io.Writer, ...)` APIs backed by `text/template`, one package per metadata standard from Scope — `metadata` emits the descriptive XML (Dublin Core), `premis` the preservation XML, `mets` the structural manifest. UUID minting for METS IDs lives in the mets `idStore`.
5. **Schemas** (`schemas/schemas.go`): all XSDs are bundled via `//go:embed` and copied into each SIP's `schemas/` dir.
6. **Archive** (`archive/zip.go`): zips the package dir into `dest/uuid-<uuid>.zip`, stored uncompressed (`zip.Store`).

Supporting pieces:

- **Domain model** (`sip/`): `Package`, `Entity`, `Representation`, `File`, `Identifier`. All identifiers take the form `uuid-<uuid>`. `File.Path` is the href relative to the METS document that references the file (documented on the field — subtle and load-bearing); `File.Source` records where essence comes from; `Entity.Description` carries the decoded descriptive metadata (`sip.Descriptive`) until the writer serializes it. `sip.Event` is an empty stub.
- **Format identification** (`formats/`): pluggable registry (`Identificator` interface, `Register`/`New` factory). `formats/siegfried` self-registers and shells out to the external `sf` binary, which must be installed on the system.
- **Config** (`services/config.go`): `.env` loaded via godotenv, parsed with caarlos0/env. Required vars: `SIP_FILE_FORMAT_NAME`, `SIP_FILE_FORMAT_COMMAND`, `SIP_FILE_FORMAT_ARGS`.

## Read the right docs

- [docs/README.md](docs/README.md) — the map of the documentation: which genre (design / decision / plan) answers which kind of question, and the lifecycle rules for each.
- [docs/sip-creator-design.md](docs/sip-creator-design.md) — the system as it is today: domain model, package layout, build lifecycle, known gaps. The entry point for understanding the code.
- [README.md](README.md) — usage, configuration, input requirements, experimental-status warning.
- [docs/TODO.md](docs/TODO.md) — open design questions and known defects. Check here first when investigating a bug.
- [CONFIG.md](CONFIG.md) — environment variables. This file is **generated** by envdoc; regenerate with `go generate ./services`, never hand-edit.
- External specs: [meemoo SIP spec 2.0](https://developer.meemoo.be/docs/diginstroom/sip/2.0/), [E-ARK CSIP profile](https://earkcsip.dilcis.eu/), [METS](https://www.loc.gov/standards/mets/) and [PREMIS](https://www.loc.gov/standards/premis/) at loc.gov.
- Concrete examples: `tmp/basic/` is a sample input tree; `basic-uuid/` is sample generated output. Both are local fixtures, not tracked in git.

## Non-negotiables

### Documentation

- Keep docs current in the same change as code: new or changed env vars require `go generate ./services` to regenerate `CONFIG.md`; changed usage or input requirements go in `README.md`; resolved items get removed from `docs/TODO.md`.

### Git

- Commit messages use a change-type prefix: `Added:`, `Changed:`, `Fixed:`, `Removed:` — capitalized, followed by a colon and a short summary. This matches existing history.
- Keep commits small and focused. Direct commits to `main` are acceptable in this experimental phase.
- No rebase or force-push of pushed history unless the user explicitly directs it.
- Never commit `.env`, `tmp/`, `basic-uuid/`, or other local fixtures and generated output.

### Code

Write idiomatic Go (Effective Go, Go Code Review Comments, Google's Go Style Guide) — clarity over cleverness, happy path left-aligned, early returns, useful zero values, exported symbols documented. The project-specific rules below take precedence where they overlap.

- **Order code so readers don't jump around.** Top-to-bottom reading should be enough; if a reader has to scroll backwards or sideways to understand the next line, restructure.
- **One concern per function.** If a function does two things, split it.
- **Name things by what they are, not how they're implemented.**
- **New helpers live next to their callers** until used from at least two places. No `helpers.go`, no anticipatory toolbox functions.
- **Name validation primitives by what they return.** `Is…`/`IsValid…` are pure `bool` predicates (e.g. a future `IsValidIdentifier`); `Validate…` returns an explanatory `error` naming the rule that failed; `Parse…` parses a string form and validates it in one step, returning the parsed values plus an error; `Resolve…` maps a value to the domain record(s) it belongs to. Choose by what the caller needs — a branch wants a predicate, a rejection path wants the error — and follow this split when adding a new primitive.
- **When a domain file grows past ~250 lines, split by concept.** E.g. rather than letting one file in `sip/` carry the package, the identifier scheme, and the file/fixity types all at once, lift each concept into its own file.

A clever-but-dense diff gets rejected; a verbose-but-obvious one is accepted. Write it the way a colleague reading it for the first time would understand fastest.

Keep the API surface small. Default to unexported — every exported symbol is a commitment in a long-lived codebase. Don't export a function or type unless it has a cross-package caller.

Keep operational failures (I/O, config) as `error`/`fmt.Errorf`. Domain validation belongs on the `sip/` domain types as `Validate` methods (none exist yet; that is where they go when they arrive). Wrap with `%w` and unwrap with `errors.Is`/`errors.As` where a caller needs to branch on the cause; don't both log and return an error — pick one.

**Streaming I/O.** An `io.Reader` is consume-once — reading advances it. Essence files can be large: prefer streaming (`io.Copy` reader-to-writer) over buffering whole files in memory.

Comment the why, not the what. Don't restate what the code or a function signature already says; explain why something exists or why it's done the unobvious way. Keep that explanation as short as it can be — usually a sentence — but a longer comment is right when it carries a domain rule or justifies a non-obvious workaround. Delete commented-out code rather than leaving it.

## Development commands

- `go build -o bin/sip-creator .` — produces the binary in the gitignored `bin/`.
- `./bin/sip-creator create --profile basic ./tmp/basic basic-uuid` — generate a sample SIP from the local fixture.
- `./build.sh` — the local CI loop: rebuilds, wipes and regenerates `basic-uuid/`, validates the zip with dockerized commons-ip, publishes the JSON reports to `reports/runs/<timestamp>/`, and exits non-zero iff the package is not `VALID`. Requires `docker` and `jq`, plus a configured `.env` pointing at a working Siegfried (`sf`) install. The gate is currently red — the sample package is known-INVALID ([docs/TODO.md](docs/TODO.md)); that exit code is a real signal, not a broken script.
- `./scripts/validate.sh [-o report-dir] <sip.zip|sip-dir>...` — validate any package standalone (also unzipped package dirs, for structure debugging).
- `docker compose up -d reports` — serve the HTML validation reports at http://localhost:8080 (see [ADR-0005](docs/decisions/0005-dockerized-validation-and-html-reporting.md)).
- `go generate ./services` — regenerate `CONFIG.md` from the config struct.

- `go test ./...` — Go tests cover the `store/` primitives and the assembler (`profiles/assemble_test.go`); they run with no external dependencies (no `sf`, no docker, no `.env`). External CSIP validation via `build.sh` remains the acceptance check — Go tests pin internal contracts and failure paths, the validator pins spec conformance. Add Go tests for new logic where practical.
- `./scripts/baseline-diff.sh tmp/baseline/pkg <pkg-dir>` — structural-equivalence gate for the in-flight [refactoring plan](docs/plans/refactoring-plan.md): diffs a generated package against the Phase-0 reference with run-varying values normalized.
