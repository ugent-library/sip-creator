# 0001 — Package builder, not archive

Status: **Accepted** (2026-07-14)

## Context

SIP Creator generates packages full of identifiers, checksums, and preservation metadata. That surface invites scope creep toward things a repository does: minting authoritative persistent identifiers, keeping a store of what it produced, tracking package state over time, guaranteeing fixity across storage. Each such feature looks reasonable in isolation, and without a stated boundary the tool drifts from a one-shot builder into a half-built archive that duplicates — and misrepresents — the real ones (RODA, meemoo's pipeline).

In OAIS terms (ISO 14721) a SIP is what a **Producer** hands to an archive's **Ingest** entity. SIP Creator is the tool the producer runs to build that package correctly *before* handing it off. Everything after ingest — transformation into an AIP, Archival Storage, dissemination as a DIP — is a separate OAIS entity that lives downstream.

## Decision

**SIP Creator is a package builder, not an archive.** It takes input, produces one well-formed SIP, and stops. Concretely, it does not:

- **Mint identifiers of its own authority.** The `uuid-<uuid>` identifiers it generates are meaningful only within the package, as the common key tying descriptive metadata, entity, representations, and files together. Authoritative persistent identifiers are the downstream repository's to assign.
- **Keep a store.** It writes one package tree and a zip to the destination the caller names, and retains nothing. It has no database, no registry of past runs, no notion of package history.
- **Make preservation guarantees.** It computes fixity so the package is *self-describing and verifiable at ingest*; it does not promise anything about the bytes after handoff. Preservation is what the archive does.

**Litmus test for any proposed feature:** does it help *build a correct package for handoff*, or does it *manage the package after handoff*? The former is in scope; the latter belongs downstream.

## Alternatives rejected

- **Persistent-identifier minting inside the tool.** Rejected: producers do not own the authoritative namespace, and baking in one archive's identifier scheme would couple the builder to a single downstream. Where an external identifier already exists in the source (e.g. a local `dcterms:identifier`), the tool *carries it through* as an additional identifier rather than inventing authority it does not have.
- **A record of generated packages (state/history).** Rejected: it would make the tool stateful and re-introduce the archival concerns (storage, integrity-over-time) this boundary exists to keep downstream. A build is a pure function from input to package.

## Consequences

- A one-line test to apply to every future feature request, so the boundary holds without re-litigating each time.
- The tool stays a composable library and single-shot CLI — easy to run in a pipeline, script, or CI, with no infrastructure.
- Identifier *authority* is explicitly the caller's/downstream's concern; SIP Creator's job is internal consistency and correct carry-through of source identifiers.
