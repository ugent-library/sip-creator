# 0014 — representations.csv is strict when present, and defaults cascade

Status: Accepted (2026-09-03)

## Context

One string used to do three jobs: the directory name under `representations/`
became the package-side directory name, the `mets/@LABEL`, and (eark profile,
ADR-0013) the type shown in RODA. That works when directory names are good
type names (`archival`, `preservation`, `access`) and fails when they aren't:
two representations of the same kind cannot share a type, and the portable
character set the directory names must satisfy is too narrow for display
labels ("Access copy (JPEG)").

The domain model already kept `Name` and `Label` distinct; only the input
side conflated them. The split lands library-first:
`profiles.SourceRepresentation` carries `Name` (required, the matching key
and package-side name), `Label`, and `Type`, and an optional
`representations.csv` at the input root is the CLI transport onto those
fields. Two rules needed a decision.

## Decision

**Strict when present.** A `representations.csv`, once present, must be
complete and correct: every row must name an existing representation
directory, no two rows may name the same one, and every directory must have a
row. A directory without a row is a violation, never an exclusion. This
follows the characterization-sidecar precedent (ADR-0009): an optional input
file, when present, is fully trusted and therefore fully checked.

**Defaults cascade name → label → type, in the library.** An empty `Label`
resolves to the `Name`; an empty `Type` resolves to the resolved label. The
cascade is applied by the assembler, not the CSV decoder, so an embedding
caller constructing `profiles.Input` directly gets identical behavior. A CSV
listing only directory names therefore produces byte-identical output to no
CSV at all.

Two supporting rules: `Label` and `Type` must not contain `< > & "` (the METS
templates do no XML escaping, and directory names were only safe by virtue of
the portable character set), and the CSV's row order is the packaging order
of the representations.

## Alternatives rejected

- **Unlisted directories are skipped** (the CSV as a packaging manifest): the
  failure mode is silent data loss. A producer who forgets a row gets a
  valid, VALID-validating SIP that is missing a representation, and nothing
  ever says so. For a tool whose job is getting content safely to ingest,
  omission must fail loudly; excluding material is done by moving it out of
  the input folder. If a workflow ever truly needs in-place exclusion, add an
  explicit marker, never absence.
- **Type defaults to "Other" or the profile declaration**: adding an
  information-free CSV would then regress the RODA type column compared to no
  CSV (which derives types from directory names, ADR-0013). Adding a file
  with no information in it must not change the output.
- **Escaping XML-unsafe labels in the encoder** instead of rejecting them:
  more machinery for a need no producer has shown. Revisit in the encoder if
  real labels ever need `&` or quotes.

## Consequences

- `SourceRepresentation.Label` was renamed to `Name`, a breaking change to
  the library API, taken deliberately: the field is the identity key, and
  `Label` now means only the display name.
- `Definition.RepresentationTypeFromLabel` became `EmitRepresentationType`:
  the type it emits now comes from the resolved input value, not always from
  the label.
- A producer can list representations in `representations.csv` purely to fix
  their order in the package; file order within a representation remains
  tool-determined (the deferred manifest feature).
- The flat single-representation case has no directories to name, so a
  `representations.csv` there is a violation; the simple case stays simple.
