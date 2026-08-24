# sip creator documentation

This folder is organized by **genre**, because genre tells you how to treat a document:
what leads, what is a decision, what is work-in-flight, and what is history. Start with
`sip-creator-design.md`; reach for the rest as noted below.

## The three genres

| Genre | Answers | Lives in | Mutable? |
|---|---|---|---|
| **Design** | *What is*: the system as it exists now | `sip-creator-design.md` | Yes, always kept current with the code |
| **Decision (ADR)** | *Why* a choice was made | `decisions/` | No: accepted ADRs are frozen; supersede, never rewrite |
| **Plan** | *How* we will build something | `plans/` | Yes, until it ships; then it retires |

**Coding and naming conventions are not here**: they live in the repo-root `CLAUDE.md`
(the always-loaded agent/contributor operating manual). The design doc describes the
system; `CLAUDE.md` states the rules for changing it. There is deliberately **one**
instruction file: if a tool that expects `AGENTS.md` is ever adopted, rename `CLAUDE.md`
to `AGENTS.md` and point a stub `CLAUDE.md` at it; don't start a second, parallel
instruction file that would drift from the first.

**Postmortems** (`incident-*`, upstream bug analyses) are a fourth genre: a record of what
went wrong and what was learned. They live in `archive/`. An incident that produces a
*standing decision* graduates that decision into an ADR, with the postmortem kept as its
evidence.

## Tiers

- **root**: authoritative reference:
  - `sip-creator-design.md`: the system design (domain model, package layout, build
    lifecycle, input contract, validation, known gaps). The entry point.
  - `input-spec.md`: the operator-facing input specification (design genre): the input
    contract the CLI enforces; the `check` command validates a folder against it.
  - `TODO.md`: the live project backlog.
- **`decisions/`**: ADRs. The *why*, permanent and live. `0000-template.md` is the shape;
  `0001`+ are real decisions. **Not** archive material: an ADR stays relevant long after
  the plan that carried it is gone.
- **`development-*.md`**: how-to guides for repeatable procedures that span several
  layers and are easy to get subtly wrong.
- **`plans/`**: in-flight or parked proposals. A plan dies on completion (see lifecycle).
- **`archive/`**: shipped plans, postmortems, and superseded notes, kept as history. Not
  authoritative; when it disagrees with the design doc or an ADR, those lead.

## Lifecycle: what happens when a plan ships

A plan is scaffolding. When the work lands it **splits in two**, then the plan retires:

1. **The *what*** → fold the durable design into `sip-creator-design.md` (present tense) so the
   design doc always describes current reality.
2. **The *why*** → distill the decision into a numbered ADR in `decisions/` (context,
   decision, alternatives rejected, consequences). Date it; then treat it as frozen.
3. **The plan doc** → move to `archive/` (or delete it; git retains it).

The payoff: `sip-creator-design.md` never carries argument. A "we decided against X" becomes a
one-line pointer to an ADR, so the design doc stays scannable and the reasoning has a
permanent, greppable home.

**Keep ADRs few and lightweight.** Do not retroactively mine every past decision into an
ADR; write one when a choice is non-obvious, likely to be questioned later, or explicitly
"do not revisit without reading this." A one-page ADR (see `decisions/0000-template.md`)
is the target, not an essay.
