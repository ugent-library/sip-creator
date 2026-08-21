# SIP Creator agent orientation

SIP Creator is both a Go library and a CLI that builds Submission Information Packages (SIPs). It takes a producer's essence files plus descriptive metadata and assembles them into a standards-conformant package. Its primary target is a valid [E-ARK CSIP](https://earkcsip.dilcis.eu/) package — ingestible by RODA and other CSIP-conformant systems — with [meemoo's SIP Specification v1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) (the **stable** version; 2.0/2.1 are release candidates) layered on top as an additional specialization for ingest into the Flemish heritage archive (hetarchief.be). The BagIt envelope meemoo's transfer requires is out of scope: bag the package directory with a reference BagIt implementation ([ADR-0008](docs/decisions/0008-bag-layer-out-of-scope.md)).

## Scope

This is digital-preservation software at the front of the chain. In [OAIS](https://www.iso.org/standard/57284.html) terms (ISO 14721) a SIP is what a *Producer* hands to an archive's *Ingest* entity; SIP Creator is the tool that builds it. It is a **package builder, not an archive** — no permanent identifiers of its own authority, no store, no preservation guarantees: it produces one well-formed SIP and stops. Everything after ingest (AIP, storage, DIP dissemination) lives downstream in RODA and meemoo's pipeline.

A SIP bundles the *essence* (the content files) with the metadata to understand and trust it: **descriptive** (Dublin Core / MODS), **preservation** ([PREMIS](https://www.loc.gov/standards/premis/): provenance, fixity, events, agents), and **structural** ([METS](https://www.loc.gov/standards/mets/): the manifest tying files, checksums, and metadata together). Generating that XML correctly — valid references, accurate fixity, right profile declarations — is the whole job, so spec compliance and valid XML matter more than features or clever code. The E-ARK CSIP profile is the base contract every package must satisfy; meemoo SIP v2.0 specializes on top of it. When in doubt, CSIP validity wins over convenience, and the meemoo spec wins over convenience within the meemoo layer.

The project is **experimental and work in progress**. `docs/README.md` is the map of the documentation: `docs/sip-creator-design.md` describes *what the system is* (schema, domain model, lifecycle); `docs/decisions/` holds the ADRs recording *why* choices were made; coding and naming conventions live in this file. Keep `docs/` current: when a refactor changes the system, update `sip-creator-design.md` in the same change, and when it enacts a decision worth remembering, add or update an ADR. A doc that drifts from the code misleads the future reader, including future-you.

Keep the dependency footprint small and boring. All XML is generated with Go `text/template` templates, not an XML library — keep it that way unless there is a strong, discussed reason to change. Adding a new external dependency is a serious choice, not a convenience.

## Audience

SIP Creator serves digital-preservation staff at UGent Library preparing SIPs for meemoo ingest. Downstream consumers are meemoo's ingest pipeline, CSIP validators (commons-ip / RODA), and the future archivists and systems that will read the preserved metadata decades from now.

Use vocabulary from the meemoo SIP spec and OAIS: SIP, essence, representation, intellectual entity, fixity, descriptive vs. preservation metadata.

## System shape

Single-binary cobra CLI. Entry point is `main.go` → `cli/cli.go` → `cli/create_cmd.go` (plus `cli/check_cmd.go`: validate an input folder without building).

Data flow for `create [src] [dest] --profile basic`:

1. **Input** (`cli/input/`): a source folder prepared per the input specification ([docs/input-spec.md](docs/input-spec.md)) — one `metadata.csv` (descriptive metadata as Dublin Core terms in CSV), content files either flat at the root or under `representations/<label>/`, plus optional `documentation/`, `premis/` (received preservation XML, passed through unparsed), and a `siegfried.json` sidecar. `input.Read` walks and validates the folder, collecting every MUST violation into one `Violations` error (the `check` command runs this standalone, ADR-0010); `Package.BuilderInput()` maps the result onto `profiles.Input`. The folder is one transport, not the API — the library never imports `cli/input`; embedding systems construct the same `profiles.Input` from their own stores.
2. **Profile** (`profiles/`): a profile is data, not code — `definition.go` holds `Definition` (output `Family`, local-identifier scheme, PREMIS emission flags, descriptive-conformance rules, METS values as `sip.Spec`) plus the registry (`Get`/`Names`); the CLI resolves `--profile` there and an unknown value lists the available profiles. One engine reads it: `Builder.Build(def, in)` (`builder.go`) validates the `Input`, checks it against the profile's conformance rules, then runs two phases with a hard seam. `assemble.go` is pure — it takes the validated `Input`, optionally enriches essence files from its pre-decoded characterization report (verifying each record's MD5 binding against the *source* bytes), and builds the complete `sip.Package` graph with zero disk writes; `write.go` emits the graph in the one canonical dependency order (skeleton → schemas → essence copies → descriptive → per-rep PREMIS then METS → package PREMIS → package METS strictly last), back-filling fixity onto graph nodes as each file lands. Errors return (no panics on the build path); a failed assembly leaves nothing on disk. A new profile is one registry entry.
3. **Store** (`store/store.go`): dumb filesystem primitives rooted at the `uuid-<uuid>` package dir under `dest` — callers speak package-relative paths. `CopyFile` computes MD5/size during the streamed copy; `WriteMetadata` renders to memory before writing (a failed template leaves no partial file); all writes truncate.
4. **Encoders** (`encoders/metadata`, `encoders/mets`, `encoders/premis`): thin `Encode*(io.Writer, ...)` APIs backed by `text/template`, one package per metadata standard from Scope — `metadata` holds the descriptive terms model (`metadata.Terms`, decoded from CSV) and emits the descriptive XML — meemoo's `dc+schema` shape or simple DC, chosen by the profile's output family — `premis` the preservation XML, `mets` the structural manifest. UUID minting for METS IDs is the `identifier` template func in the mets encoder.
5. **Schemas** (`schemas/schemas.go`): all XSDs are bundled via `//go:embed` and copied into each SIP's `schemas/` dir.
6. **Archive** (`archive/zip.go`): zips the package dir into `dest/uuid-<uuid>.zip`, stored uncompressed (`zip.Store`).

Supporting pieces:

- **Domain model** (`sip/`): `Package`, `Entity`, `Representation`, `File`, `Identifier`. All identifiers take the form `uuid-<uuid>`. `File.Path` is the href relative to the METS document that references the file (documented on the field — subtle and load-bearing); `File.Source` records where essence comes from; `Entity.Description` carries the decoded descriptive metadata (`metadata.Terms`) until the writer serializes it. `sip.Event` is an empty stub.
- **Characterization** (`characterization/`): format info is **optional pre-computed input** ([ADR-0009](docs/decisions/0009-characterization-as-sidecar-input.md)) — the library takes a pre-decoded `characterization.Report` via `Input.Characterization`; the CLI fills it from the `siegfried.json` sidecar at the input root (generated with `sf -hash md5 -json`; the reserved name is a `cli/input` constant); the tool never executes a characterization tool. Absent → skipped (premis:format is a SHOULD; fixity is computed natively by the store). Present → fully strict: malformed report, missing essence entry, per-entry sf error, missing checksum, or an MD5 mismatch against the source bytes aborts assembly (staleness defense); an entry with empty `matches[]` → nil format for that file only. Documentation files are lenient (entry optional) but checksum-verified when present.
- **Config** (`cli/config.go`): environment vars, with `.env` loaded when present (a missing file is fine; `create` requires the `SIP_SUBMITTER_*` vars, checked at profile resolution). Config is CLI wiring, unexported by design — embedding systems supply their destination and logger as data (`profiles.Config`); a package's material, characterization report included, arrives per build as `profiles.Input`, never via env vars.

## Read the right docs

- [docs/README.md](docs/README.md) — the map of the documentation: which genre (design / decision / plan) answers which kind of question, and the lifecycle rules for each.
- [docs/sip-creator-design.md](docs/sip-creator-design.md) — the system as it is today: domain model, package layout, build lifecycle, known gaps. The entry point for understanding the code.
- [README.md](README.md) — usage, configuration, input requirements, experimental-status warning.
- [docs/input-spec.md](docs/input-spec.md) — the input folder specification: what producers must prepare for the CLI (`metadata.csv`, representations, documentation, premis, sidecar) and every MUST/SHOULD the `check` command enforces.
- [docs/TODO.md](docs/TODO.md) — open design questions and known defects. Check here first when investigating a bug.
- [CONFIG.md](CONFIG.md) — environment variables. This file is **generated** by envdoc; regenerate with `go generate ./cli`, never hand-edit.
- External specs: [meemoo SIP spec 1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/) (stable; 2.x are release candidates), [E-ARK CSIP profile](https://earkcsip.dilcis.eu/), [METS](https://www.loc.gov/standards/mets/) and [PREMIS](https://www.loc.gov/standards/premis/) at loc.gov.
- Concrete examples: `tmp/basic/` is a sample input tree; `basic-uuid/` is sample generated output. Both are local fixtures, not tracked in git.

## Non-negotiables

### Documentation

- Keep docs current in the same change as code: new or changed env vars require `go generate ./cli` to regenerate `CONFIG.md`; changed usage or input requirements go in `README.md`; resolved items get removed from `docs/TODO.md`.

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
- `./build.sh [profile]` — the local CI loop (default `basic`): rebuilds, wipes and regenerates `<profile>-uuid/` from `tmp/<profile>`, validates the zip with dockerized commons-ip, publishes the JSON reports to `reports/runs/<timestamp>-<profile>/`, and exits non-zero iff the package is not `VALID`. **Both gates are green** — each profile validates against the E-ARK spec version of its era (`basic`/meemoo-1.2 → 2.0.4, `eark` → 2.2.0; see the [meemoo-12 plan](docs/archive/meemoo-12.md)). Requires `docker` and `jq`, plus `sf` (Siegfried) on PATH — build.sh regenerates the input fixture's `siegfried.json` sidecar before building (a stale sidecar is a hard build failure by design); the sidecar is optional for the tool itself, but the baseline reference tree contains format info, so the equivalence gate expects it.
- `./scripts/validate.sh [-o report-dir] <sip.zip|sip-dir>...` — validate any package standalone (also unzipped package dirs, for structure debugging).
- `docker compose up -d reports` — serve the HTML validation reports at http://localhost:8080 (see [ADR-0005](docs/decisions/0005-dockerized-validation-and-html-reporting.md)).
- `go generate ./cli` — regenerate `CONFIG.md` from the config struct.

- `go test ./...` — Go tests cover the `store/` primitives, the assembler (`profiles/`), the input reader (`cli/input/`), and the descriptive terms model (`encoders/metadata/`); they run with no external dependencies (no `sf`, no docker, no `.env`). External CSIP validation via `build.sh` remains the acceptance check — Go tests pin internal contracts and failure paths, the validator pins spec conformance. Add Go tests for new logic where practical.
- `./scripts/baseline-diff.sh tmp/baseline/pkg <pkg-dir>` — the standing structural-equivalence gate (born as [refactoring plan](docs/archive/refactoring-plan.md) Phase 0): diffs a generated package against the blessed reference in `tmp/baseline/` with run-varying values normalized. Refactors must diff clean; a deliberate output change re-blesses the baseline consciously (record it in the commit message and `tmp/baseline/README.md`).

## Tone and Style guidelines

### Banned Vocabulary & Phrases

Under no circumstances use the following overused AI tells, buzzwords, or structural clichés in chat responses, documents in the docs directory or code comments. Two carve-outs: "gate" is allowed as the established project name for the validation checks (the baseline gate, the equivalence gate, "the gate is green"), just not as loose metaphor; and `docs/archive/` and `docs/decisions/` are historical records, so leave their existing text as-is rather than rewriting it to comply. The rules apply to new and edited prose.

- **The Structural Group:** "load-bearing" (and "load-bearing seams"), "gates", "seams", "spine", "substrate", "blast radius", "friction", "birth".
- **The Proverbial Group:** "footgun", "yak shaving", "belt-and-suspenders", "smoking gun", "classic trap".
- **The Pretentious Group:** "tapestry", "delve", "testament to", "beacon", "underscore", "honest take", "identity made legible".
- **The "Gaslighting" Transitions:** "You're absolutely right!", "That's totally on me", "Now I have the full picture", "It's worth noting/flagging/considering", "I'd gently reset the framing".
- **The AI Cliché Structure:** Do not use the pretentious "That's not just X, it's Y" writing style.

### Formatting & Punctuation Constraints

- **Em-Dash Ban:** Drastically limit or eliminate the use of em-dashes (—). Write clean, separate sentences instead of embedding clauses.
- **No Invented Acronyms:** Do not invent internal abbreviations on the fly (e.g., converting a function name like `initialVerification` into `IV`).

### Execution & Behavior (Don't Go Rogue)

- **Code is onboarding, not a philosophy essay:** Write code documentation like you are onboarding a smart developer, not writing a dramatic tech essay.
- **Ask before tearing up files:** If a requirement or product decision is ambiguous, do not confidently guess and run a 12-file diff. Stop and ask clarifying questions first.
- **Spell out consequences literally:** Do not just say code is "fragile" or a "trap." Explain exactly what breaks, to whom, and under what specific action.
- **Lead with the point:** Put the solution, code fix, or core answer in the very first sentence. If a user only reads sentence one, they should have the complete gist.
