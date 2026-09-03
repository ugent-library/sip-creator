# Plan: representations.csv, separating representation name, label, and type

*Status: **agreed, not started** (design discussion 2026-09-03). Prerequisite:
the representation-typing change ([ADR-0013](../decisions/0013-representation-type-from-label.md))
must be committed first; this plan builds directly on it. When this plan ships,
fold the durable parts into `sip-creator-design.md` and `input-spec.md`, record
the strictness decision as an ADR, and retire this file to `archive/`.*

## Context

One string currently does three jobs. The directory name under
`representations/` in the input folder becomes:

1. the package-side directory name and representation METS `OBJID`,
2. the `mets/@LABEL`, and
3. (eark profile, since ADR-0013) the type RODA shows for the representation,
   stamped into both the `TYPE`/`csip:OTHERTYPE` and
   `CONTENTINFORMATIONTYPE`/`OTHERCONTENTINFORMATIONTYPE` attribute pairs.

The domain model already separates the first two (`sip.Representation` has
distinct `Name` and `Label` fields); only the input side conflates them. Two
scenarios the conflation cannot express:

- **Type is a category, label is a name.** Two representations of the same
  kind (say, `access-jpeg` and `access-pdf`) should both show type `access`
  in RODA. With type derived from the directory name they show two different
  types.
- **Directory names are constrained, labels shouldn't be.** The input walker
  enforces the portable character set on directory names. "Access copy (JPEG)"
  cannot be a directory but is a fine `mets/@LABEL`.

For inputs whose directory names already are good type names (UGent's
`archival` / `preservation` / `access` vocabulary), the current derivation is
exactly right and costs nothing. This plan adds the escape hatch for inputs
where it isn't, without changing behavior for anyone who doesn't use it.

## Design decisions (agreed 2026-09-03)

1. **Library first; the CSV is transport.** The input folder is one transport,
   not the API. `profiles.SourceRepresentation` gains the split fields; the
   CSV is `cli/input` wiring that maps onto them. Embedding systems set the
   fields directly and never see a CSV.
2. **`SourceRepresentation` splits into three fields:**
   - `Name`: the package-side directory name and the matching key. Required.
     This is today's `Label` renamed (a deliberate breaking change to the
     exported type; the project is experimental and keeping a misnamed field
     would confuse every future reader).
   - `Label`: the display name, emitted as `mets/@LABEL`. Optional.
   - `Type`: what the eark profile declares as the representation's type
     (both attribute pairs, per ADR-0013). Optional. Inert under the meemoo
     family, which fixes its representation typing to the profile URI.
3. **Defaulting cascade, applied in the library:** empty `Label` defaults to
   `Name`; empty `Type` defaults to `Label`. The cascade lives in the
   assembler (or `Input` validation), not in the CSV parser, so embedding
   systems and the CLI get identical behavior. Consequence: an input that
   sets only `Name` produces byte-identical output to today.
4. **Type does NOT default to "Other" or the profile declaration.** An
   information-free CSV (only directory names filled in) must be a no-op:
   adding the file without adding information must not change the output.
   Defaulting Type to anything other than the label would silently regress
   the RODA type column for producers who add a CSV just to set labels.
5. **Strict when present.** This follows the characterization-sidecar
   precedent (ADR-0009): an optional input file, once present, is fully
   trusted and fully checked. Silent partial application is worse than
   absence. The rejected alternative was "directories not listed in the CSV
   are skipped": that turns the CSV into a packaging manifest whose failure
   mode is silent data loss (a forgotten row drops a representation from the
   SIP with no error, and the package still validates VALID). For a tool
   whose job is getting content safely to ingest, omission must fail loudly.
   A producer who wants to exclude a directory moves it out of
   `representations/`. If a real workflow ever needs in-place exclusion, add
   an explicit marker column then; do not make absence mean exclusion.
6. **XML safety is a new validation rule.** Labels are XML-safe today only
   because they are directory names (the portable character set excludes
   `"`, `<`, `&`). The METS encoder is `text/template` with no XML escaping,
   so a free-text label would produce a malformed document. Rule: `Label` and
   `Type` values must not contain `<`, `>`, `&`, or `"` (reject with a
   violation, don't escape; revisit escaping in the encoder only if real
   labels need those characters).
7. **CSV row order defines packaging order** of the representations
   (fileSec/structMap order in the package METS). Deterministic, free, and
   gives producers control they don't have today. Without a CSV the order
   stays the directory walk order.
8. **Rename the profile flag.** `Definition.RepresentationTypeFromLabel`
   stops being accurate once the type can come from input. It becomes a name
   that says "type each representation METS from its resolved Type value"
   (exact name at implementation time); `representationDeclaration` takes the
   resolved type instead of the label. ADR-0013's reasoning (dual stamping,
   both pairs) is unchanged; only where the string comes from changes.

## File specification (to fold into input-spec.md when shipped)

`representations.csv`, optional, at the input root next to `metadata.csv`.
Header row required. Columns:

| column | required | meaning |
|---|---|---|
| `Directory` | yes | bare directory name under `representations/` (e.g. `master`, not a path) |
| `Label` | no | display name; empty means the directory name |
| `Type` | no | representation type; empty means the label |

Rules, all MUST violations collected by `check` alongside the existing ones:

- Every row's `Directory` matches an existing directory under
  `representations/`; an unmatched row is a violation.
- Every representation directory is covered by exactly one row; an uncovered
  directory is a violation, and so is a duplicate `Directory`.
- A `representations.csv` with no data rows is a violation.
- `Label` and `Type` must not contain `<`, `>`, `&`, or `"`.
- Unknown column headers are a violation (consistent with the strict posture;
  a typo'd header must not silently drop a column).

## Execution steps

- **S1: library.** Split `SourceRepresentation` into `Name`/`Label`/`Type`;
  apply the defaulting cascade and the XML-safety check in the library;
  rename the profile flag and rework `representationDeclaration`; update the
  assembler (`Name` for paths and OBJID, `Label` for `mets/@LABEL`, resolved
  `Type` for the eark typing). Go tests: cascade defaults, XML-safety
  rejection, eark typing from an explicit `Type`, meemoo family ignores
  `Type`, order preserved.
- **S2: CLI input.** Parse `representations.csv` in `cli/input` (reserved
  name as a constant, like `siegfried.json`); enforce the matching/coverage/
  uniqueness/header rules as collected violations; map rows onto
  `SourceRepresentation` without applying defaults; adopt row order.
  Go tests over fixture folders for each violation and the happy path.
- **S3: docs.** New section in `input-spec.md` (the table and rules above);
  README input requirements; `sip-creator-design.md` (SourceRepresentation
  fields, the cascade); a new ADR recording the strict-coverage decision and
  the rejected skip semantics; note in this plan's status line as steps land.
- **S4: acceptance.** `go test ./...` green. `./build.sh basic` and
  `./build.sh eark` both VALID; `scripts/reference-diff.sh` clean against
  `tmp/reference/pkg` (the basic fixture gains no CSV, so its output must be
  unchanged). Add a `representations.csv` with only the `Directory` column to
  `tmp/eark` and confirm the eark output is byte-identical to the no-CSV
  build (the no-op property of decision 4); then set a distinct `Type` and
  confirm the representation METS carries it in both attribute pairs.

## Open questions

- **Exact new name for the profile flag** (decision 8): pick at
  implementation time; the doc comment matters more than the name.
- **Explicit exclusion marker**: deliberately deferred until a workflow that
  cannot remove directories from the input actually appears.
- **Escaping instead of rejecting XML-unsafe labels**: revisit only if a real
  producer needs `&` or quotes in a label; that change belongs in the METS
  encoder, not in the input rules.
