# 0003 — CSIP validation stays external (commons-ip), not reimplemented as Go tests

Status: **Accepted** (2026-07-14)

## Context

A SIP is only useful if it validates. The natural instinct for a Go project is to encode "is this package valid?" as a Go test suite — assert on the generated XML directly. But the validity contract here is the **E-ARK CSIP profile**: ~140 numbered checks (CSIP1…CSIPn) over structure, METS references, fixity, and metadata, with MUST/SHOULD/MAY cardinalities and an evolving specification version. That contract is already implemented, canonically, by **commons-ip** (the `csip` CLI, the same validator RODA uses on ingest).

Reimplementing those checks in Go would mean maintaining a second, unofficial copy of the spec that could disagree with the real one — and the disagreements that matter are exactly the ones a hand-rolled reimplementation would get wrong.

## Decision

**The CSIP validation rules live in the `csip` tool, not in this repo.** The acceptance check is: generate a package, run `csip validate` over it, and pass iff the tool reports `VALID` (no MUST-level failures). Today this is driven by `build.sh`; the [validation-workflow plan](../archive/validation-workflow-plan.md) reshapes that into a reusable `scripts/validate.sh` with honest exit codes.

Go tests are for **our own logic** — the assembler's graph shape, path computation, identifier carry-through, filesystem primitives — not for re-asserting the CSIP spec. The division: we test that we build what we intended; commons-ip tests that what we built is a valid CSIP.

## Alternatives rejected

- **Reimplement CSIP checks as Go tests.** Rejected: far too cumbersome, and it creates a second source of truth for validity that will drift from commons-ip. When our copy and the real validator disagree, the real validator wins by definition — so our copy adds maintenance cost and false confidence, not safety.
- **A hybrid — a few "important" CSIP checks in Go, the rest external.** Rejected: no principled line between "important" and not, and every check duplicated in Go is one that can silently diverge. Keep the boundary clean: all CSIP rules external, all our-logic tests internal.

## Consequences

- **The acceptance gate is a shell script wrapping `csip`, not a Go test suite**, and it depends on external tools (`csip`, `jq`, plus a working Siegfried for generation). Per the [validation-workflow plan](../archive/validation-workflow-plan.md), the gate is to be honest about MUST vs. SHOULD — SHOULD/MAY failures printed but not build-failing, matching commons-ip's own error/warning split — and to exit non-zero iff a package is INVALID; today's `build.sh` predates that plan and has neither property yet.
- **The gate can start red.** The sample package is known-INVALID today; that is a real signal from the real validator, not a broken test to suppress. Automation that blocks (pre-push, CI) is deferred until the gate is green, to avoid training `--no-verify` reflexes.
- A CSIP spec-version bump is absorbed by upgrading `csip`, not by editing test code. The spec version must be pinned explicitly in the validator script (the plan pins `--specification-version=2.2.0`) so a `csip` upgrade cannot silently move the goalposts.
