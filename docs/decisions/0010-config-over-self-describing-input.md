# 0010 — Administrative values come from configuration, not the input folder

Status: **Proposed** (drafted 2026-08-18 with the
[input-convention plan](../plans/input-convention.md); accept when its I3
cut-over lands).

## Context

The [input specification](../input-spec.md) makes a folder the operator's
entire interface: content files plus one `metadata.csv` describing the
content. But a SIP also carries administrative metadata — submitting
organization, archival creator, contact persons, submission agreement
reference, content category — and *somewhere* has to supply it.

The tension is between two goods. A **self-describing input** (the folder
alone fully determines the package) is reproducible and auditable from the
input side: hand the folder to anyone, get the same SIP. **Configuration**
matches how the values actually behave: they span every package an
installation produces, change on the timescale of contracts rather than
collections, and are owned by the institution, not the operator describing
a photo album. Repeating them per package invites drift — a typo in one
folder's agreement number is exactly the class of silent error this tool
exists to prevent — and the operator writing `metadata.csv` is usually not
the person who knows them.

There is also a boundary argument: the library takes administrative
metadata as data (`sip.Spec`/`sip.Agent`), and embedding systems supply it
programmatically. Whatever the CLI does is frontend policy, not domain
design — which frees the CLI to optimize for its actual audience,
digital-preservation staff preparing many packages under one standing
agreement.

## Decision

**The input folder carries only content and content-describing metadata.
Administrative values come from the tool's configuration, merged in at
build time.**

- Configuration owns: submitting organization, archival creator, contact
  person(s), submission agreement reference, default target profile,
  default content category.
- The command line may override per run what plausibly varies per package
  (profile, content category, record status, the updated package's
  identifier) — flags, not folder contents.
- The generated package records every administrative value used (metsHdr
  agents, altRecordID): **audit the output, not the input**.
- Input-contract validation (`check`) is config-independent: a folder's
  conformance to the spec never depends on installation settings.
- Consequence accepted openly: one installation serves one submitting
  organization. If that assumption ever breaks, this decision must be
  revisited (per-package override files are the likely shape — deferred in
  the spec's §8, not designed).

## Alternatives rejected

- **An administrative block in `metadata.csv`** (or a second per-package
  admin file). Puts contract-scoped values in collection-scoped files:
  every package repeats them, every repetition can contradict the last, and
  the person editing the file is the wrong author for them. The spec's
  unknown-key strictness would also blur — one file mixing keys an operator
  must write with keys they must never touch.
- **Fully self-describing folders as a hard requirement.** The
  reproducibility gain is real but thin — the package itself already
  records the values used, so the audit trail survives — and the cost is
  paid on every package forever.
- **Per-package config overrides now.** Wanted by no current user; designing
  the merge semantics speculatively violates the deferred-features
  discipline. Recorded in the spec's §8 so it is chosen against, not
  forgotten.

## Consequences

- An input folder alone does not determine the package; rebuilding a SIP
  bit-identically requires the same configuration. Accepted: the SIP, not
  the input folder, is the record of what was submitted.
- Operators write less and can break less; institutional values change in
  one place.
- `check` stays runnable anywhere — CI, a bare laptop — with no `.env`.
- Multi-tenant use of one installation is explicitly unsupported until this
  ADR is revisited.
- The CLI/library boundary stays clean: configuration is CLI wiring;
  embedding systems never see it (`profiles.Config` takes data, per the
  existing design).
