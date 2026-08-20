# Plan: implement the input convention

*Status: **Draft** (2026-08-18) — for review, not yet started. Enacts
[input-spec.md](../input-spec.md); accompanied by
[ADR-0010](../decisions/0010-config-over-self-describing-input.md) (the
config-over-self-describing-input trade-off the spec bakes in).*

## Context

The [input specification](../input-spec.md) replaces the current CLI input
contract (`dc+schema.json` + `representation_N/` directories) with a
convention-based folder: one key–value `metadata.csv`, an optional
`representations/` tree with free-form names, optional `documentation/` and
`premis/` folders, administrative values from configuration. The spec is
operator-facing and complete, including its deferred-features list; this plan
is the code-facing counterpart.

Two decisions were settled before drafting (2026-08-18):

1. **Clean replacement, not a parallel frontend.** The tool is experimental
   with no external users to migrate; two live input contracts means two
   walkers and a fixture zoo. The old contract is removed in the same arc,
   `tmp/basic` is rewritten in the new convention, and the baseline is
   re-blessed once, consciously.
2. **Minimal borrowing from the library-API arc.** The input spec forces
   exactly two items off the embeddability list in [TODO.md](../TODO.md): a
   caller-supplied package identifier (`--updates` reuses the original
   `mets/@OBJID`) and metadata arriving as data rather than files the
   library parses. Those two ride along; stream-first essence, supplied
   fixity, and `Package.Validate()` stay parked with the API arc.
   Side effect, not a goal: moving input discovery out of
   `profiles/assemble.go` shrinks it toward the pure graph-builder that arc
   wants anyway.

### What the gap is (measured against the code, 2026-08-18)

| spec requirement | code today |
|---|---|
| folder walker: reserved names, flat vs. `representations/`, symlink refusal, OS-artifact filtering, NFC comparison, natural sort | none — `assemble.go` walks for `representation_([0-9]+)$` and silently skips the rest |
| `metadata.csv` decode: BOM/CRLF/RFC-4180, repeated keys, `[lang]` tags, unknown-key error, required keys | none — descriptive is JSON-LD decoded into the fixed-field `metadata.Description` struct |
| report **all** violations at once, plain language, check-only mode | none — errors are fail-fast, phrased for developers |
| `--status` / `--updates` → `RECORDSTATUS`, `OBJID` reuse | no `sip.Spec` slot, package identifier always minted |
| content category selectable per run | fixed registry value (standing TODO item) |
| free-form representation labels | standing TODO item (regex, silent skip) |
| per-representation `metadata.csv` → rep-level dmdSec (CSIPSTR12/13) | no description slot on `sip.Representation`, no rep dmdSec emission |
| `premis/` pass-through → `metadata/preservation/`, amdSec refs | not modeled; `Representation.PremisFile` is the *generated* PREMIS |
| `documentation/` (package level) | **done** (eark plan E3) |
| checksums, sizes, characterization | **done** (store, ADR-0009) |

## Design

### Where the input convention lives: a new `cli/input` package

Per the [CLI/library boundary](../sip-creator-design.md#clilibrary-boundary),
the library never sees a CSV or a source layout, and operator-facing error
reporting is a CLI concern. But `cli/` itself is deliberately thin cobra
wiring — the convention needs real logic with real tests. So: a `cli/input`
package owning the walker, the CSV decoder, the rule table, and violation
reporting. Nested under `cli/` so the location itself states the ownership
(moved from a top-level `input/` during review, 2026-08-19): it is CLI
frontend, not API surface. Exported because `cli/` is its cross-package
caller; the library (`profiles/`, `sip/`) never imports it.

`input.Read(root)` returns an `input.Package` — the neutral in-memory result
of walking and validating one folder:

- representations (label + ordered file list: absolute source path,
  input-relative report key, package-relative logical path), flat case
  normalized to one representation,
- decoded package-level descriptive terms, per-representation terms when
  present,
- documentation and received-premis file lists (package and rep level),
- the decoded characterization report when the sidecar is present.

The CLI hands this to the builder as data (a per-build `profiles.Input`).
`assemble.go` stops walking input directories entirely: it builds the graph
from what it is given. Characterization strictness (ADR-0009 MD5 binding)
stays in assemble — it guards the *graph*, not the folder, and applies
equally to library callers.

### The folder is one transport: every input class arrives as data

*(Naming revised in the 2026-08-19 review: the payload lives on
`profiles.Input`, passed per build to `Build(def, in)` — `profiles.Config`
stays what config means everywhere else in the repo, builder wiring
(destination, logger). Settings configure the machine; Input is the
material it works on — the same distinction ADR-0010 draws.)*

The `profiles.Input` fields are typed **library-side** (`profiles/`,
`sip/`, `encoders/metadata`), and `cli/input` maps the folder onto them —
the import direction is `input → profiles`, never the reverse. This is the
load-bearing choice of the whole arc: it makes the folder convention one
producer of the builder's input among others. Systems that automate ingest
workflows — where package building happens as one step in a pipeline, with
descriptive records in a database and essence staged from object storage to
local disk — construct the same config values directly and never write a
`metadata.csv` (or `siegfried.json`) into the staging directory just for
the tool to re-parse it. The rule mirrors ADR-0009's caller-supplied
`characterization.Report` verbatim: *the file is one transport, not the
API* — now applied to every input class (descriptive, documentation,
received PREMIS, representations), not just characterization.

Consequence for the existing characterization seam: the builder's
sidecar-file fallback (`assembleCharacterization`'s `os.Open` path) is
removed in I3 — `input/` reads and decodes `siegfried.json`, the library
only ever receives a decoded `Report`. `Definition.CharacterizationSource`
leaves the `Definition` alongside `DescriptiveSource`: the sidecar filename
is a reserved name of the input convention (spec §1), not per-profile data.
ADR-0009's decision is untouched — optional in contract, fully strict when
present, MD5-bound in assemble; only the filename-discovery seam moves from
profile data to the input convention.

### Violations, not errors

Input-contract failures are collected, not thrown: `input.Read` accumulates
every MUST violation and returns them together (`input.Violations`, an
`error` whose message is the full plain-language list, one line per finding,
naming the file or folder concerned). SHOULD findings come back as warnings
alongside. This is the one place in the codebase where collect-all beats
fail-fast, because the audience is an operator fixing a folder, not a
developer reading a stack. Library errors stay fail-fast `fmt.Errorf`.

The check-only mode is a new command, `sip-creator check [src]`: run
`input.Read`, print violations and warnings, exit non-zero on violations,
build nothing. It needs no configuration — the input rules are
config-independent by construction (ADR-0010).

### Validation splits in two: transport rules and graph rules

*(Added in the 2026-08-19 review.)* `cli/input` holds two kinds of rules,
and only one of them belongs to it. **Transport rules** are meaningless
without a filesystem — reserved names, flat vs. `representations/`,
symlinks, OS artifacts, CSV mechanics, sidecar file decoding — and stay
CLI-side: an embedding caller constructing a `profiles.Input` directly can
never violate them. **Graph rules** bind every producer: at least one
representation, each with at least one file; descriptive present with
identifier and title, exactly one local identifier; legal `Terms` elements
(the DCMI vocabulary); portable-charset representation labels; no duplicate
logical paths. Those move library-side, where every producer hits them —
per the design doc's standing "validation splits in two" principle:

- **I2**: `Terms` gains validation next to the type in `encoders/metadata`
  (the DCMI vocabulary table and schema-shape rule move out of
  `cli/input`); the CSV decoder calls it and wraps findings as operator
  `Violations` with file/line context.
- **I3**: the builder validates its input data before assembling —
  fail-fast `error`s, not `Violations`; the library's audience is a
  developer, and the CLI keeps its own overlapping checks where they buy
  better messages. Duplication is deliberate: two audiences.

This un-parks the *input-data* slice of the `sip.Package.Validate()` item
in [TODO.md](../TODO.md): its deferral trigger ("the invariants become real
input-error classes only when callers construct graphs") fires at the I3
cut-over, which makes `profiles.Input` exactly that. The graph-level
remainder (identifier uniqueness across the graph, the no-empty-`Mime`
invariant) stays parked with the library-API arc.

### Descriptive metadata: an ordered-terms type, not the JSON-LD struct

The CSV model (any key repeatable, `[lang]` on any value) does not fit
`metadata.Description`, whose fixed fields hold exactly one title and one
description. Forcing it in would silently drop repeated values — the exact
failure the spec's unknown-key rule exists to prevent.

Instead the CSV decodes to an ordered list of terms — `(dcterms element,
lang, value)` triples, CSV order preserved — and a new type in
`encoders/metadata` (working name `metadata.Terms`) implements
`sip.Descriptive` over it with a new template define that renders the same
document shape (root element, namespaces) with one element per term. The
plain-key → dcterms mapping table (spec §3/§7) lives in `input/` — it is
input contract, not encoding. `Terms` carries the small identifier seams
assemble needs today (`SetObjectIdentifier`, `GetLocalIdentifier`), so the
`MEEMOO-LOCAL-ID` flow is unchanged. `Terms` is plainly constructible —
an exported slice-of-triples, not a parse-only product — because the CSV
decoder is only one producer of it: an embedding system maps its own
records onto `Terms` (or supplies any other `sip.Descriptive`) directly.

The JSON-LD decoder and the `dc+schema` define are removed with the old
contract **only if** the eark family doesn't share them; the family seam
(ADR-0007: families select encodings) is re-pointed at the terms encoding.
`Definition.DescriptiveSource` and `Definition.CharacterizationSource` both
leave the `Definition` — the input convention is profile-independent, so
there are no per-profile input filenames any more (see "The folder is one
transport" above for the characterization side).

### Small spec/domain additions

- `sip.Spec.RecordStatus` → `metsHdr/@RECORDSTATUS` (SIP3 vocabulary,
  default `NEW`); CLI flags `--status` and `--updates <id>`; `--updates`
  makes the package identifier caller-supplied (`sip.NewPackage` learns to
  accept one, minting only when none given — the first embeddability item).
- Content category: `--content-category` flag with a configured default,
  landing in `sip.Spec.Type` — closes the standing `mets/@TYPE` TODO item.
- `sip.Representation` gains `Description`/`DescriptionFile` (mirroring
  `Entity`), and received-premis file lists appear on both `sip.Package` and
  `sip.Representation` — pass-through files are graph nodes like
  documentation, copied not parsed.

### Received PREMIS: honesty over the spec's MUST

The spec says `premis/` files "MUST be valid PREMIS 3.0 XML", but this tool
does not XSD-validate (validation stays external, ADR-0003), and importing
an XSD validator for one rule fails the small-and-boring test. v1 enforces
what it can honestly enforce: well-formed XML whose root is
`premis:premis` in the PREMIS 3.0 namespace; full schema validity remains
commons-ip's job downstream. **The spec's §5 wording should be revised to
match** (MUST be well-formed PREMIS 3.0-namespaced XML; SHOULD be
schema-valid) — flagged for the spec revision accompanying this plan.

## Steps

Every step ends with `./build.sh` green for both profiles and a clean (or
consciously re-blessed) baseline diff — that gate is the last checkbox of
each step. Check items off as they land; a step is done when every box is.

### I1 — the `cli/input` package

Pure addition, nothing wired; this package is almost all contract and the
validator can't see any of it, so the tests carry the step.

- [x] Package skeleton: `input.Read(root)` → `input.Package`, the neutral
      in-memory model (representations, descriptive terms, documentation,
      received premis, characterization report).
- [x] Walker: reserved top-level names; flat case normalized to one
      representation; `representations/` case with content-elsewhere
      violation; rep-name character rule.
- [x] Walker rules: symlinks are an error; OS artifacts (`.DS_Store`,
      `Thumbs.db`, `desktop.ini`, `._*`) silently ignored; NFC-normalized
      path comparison; stable file order. *(Deviation, 2026-08-19 review:
      natural sort dropped — no spec assigns semantics to file order, and
      explicit sequencing is the deferred manifest → METS `ORDER` feature.
      Order is deterministic lexical traversal; spec §2/§7 reworded to
      match.)*
- [x] Sidecar handling: decode `siegfried.json` when present (reuses
      `characterization.DecodeSiegfried`); absent is fine.
- [x] CSV decoder: UTF-8 + BOM, CRLF, RFC 4180 quoting; `key,value` header;
      repeated keys; `[lang]` suffixes; the plain-key → dcterms table;
      `dcterms:*`/`schema:*` prefixed keys; required `identifier`/`title`;
      unknown key is a violation.
- [x] Per-representation `metadata.csv` variant (identifier/title optional).
- [x] `input.Violations`: collect-all `error`, one plain-language line per
      finding naming the file or folder; SHOULD findings as warnings.
- [x] Go tests: table cases for every MUST rule above, plus a
      three-violations-reports-all-three case.
- [x] Gate: `./build.sh basic` + `eark` green, baseline diff clean
      (trivially — nothing is wired).

### I2 — the terms encoding

- [x] `metadata.Terms`: exported ordered `(element, lang, value)` triples,
      plainly constructible; implements `sip.Descriptive`; carries
      `SetObjectIdentifier`/`GetLocalIdentifier`.
- [x] New dcterms template define rendering the same document shape (root,
      namespaces), one element per term, `xml:lang` when set.
- [x] Family seam re-pointed at the terms encoding (ADR-0007 mechanics).
- [x] `Terms` validation next to the type (legal element names — the DCMI
      vocabulary table and schema-shape rule — and language-tag shape all
      move here from `cli/input`); the CSV decoder wraps its findings as
      `Violations` with file/line context, keeping only the key *syntax*
      (plain-key spelling table, `[lang]` brackets) as transport.
- [x] Unit tests pin the XML shape: repeated elements, `xml:lang`,
      escaping, term order preserved — plus term validation (bogus element
      rejected identically via CSV and via direct construction).
- [x] Gate: builds green, baseline clean (still not reachable from the CLI).

### I3 — the cut-over

The big one: after this step the old contract no longer exists and the
builder is purely data-fed.

- [x] The library-side input fields land as `profiles.Input` (types in
      `profiles/`/`sip/`/`encoders/metadata`, per "The folder is one
      transport"; `Config` reverted to wiring-only in the 2026-08-19
      naming review).
- [x] `create` reads via `input.Read`, maps the result onto
      a `profiles.Input`, prints violations/warnings operator-style.
- [x] `assemble.go` drops all input reading: the `representation_N` regex,
      `decodeDescriptive`, the documentation walk, the sidecar-file
      fallback.
- [x] `Definition.DescriptiveSource` and `Definition.CharacterizationSource`
      removed from the `Definition`.
- [x] Old JSON-LD path removed (decoder + `dc+schema` define, if the eark
      family confirms it doesn't share them).
- [x] `sip-creator check [src]` command: validate only, print findings,
      exit non-zero on violations, build nothing, no config needed.
- [x] `tmp/basic` and the eark fixture rewritten in the new convention;
      `build.sh` untouched or minimally adjusted (sidecar regeneration
      unchanged).
- [x] Builder validates its input data before assembling (fail-fast
      errors): ≥1 representation, each with ≥1 file; descriptive present
      with identifier and title, exactly one local identifier; terms valid;
      portable-charset labels; no duplicate logical paths — the
      embedding-caller guardrail (see "Validation splits in two").
- [x] Assemble test: hand-constructed `profiles.Input` (no `metadata.csv`
      / `siegfried.json` on disk) assembles the same graph as the folder
      path — the embedding-caller contract. Its negative twin: an invalid
      hand-constructed config (zero representations, bogus term element)
      is rejected with nothing written.
- [x] **Baseline re-bless**: turned out NOT to be needed (2026-08-19) —
      the fixture CSV states its terms in the old template's emission
      order, so the cut-over output is structurally identical and the
      baseline stands unchanged (noted in `tmp/baseline/README.md`).
- [x] Gate: `./build.sh basic` + `eark` **VALID** against the unchanged
      baseline.

### I4 — status, updates, content category

- [x] `sip.Spec.RecordStatus` → `metsHdr/@RECORDSTATUS` (SIP3 vocabulary,
      default `NEW`); metsHdr template addition.
- [x] Caller-supplied package identifier: `sip.NewPackage` accepts one,
      mints only when none given.
- [x] CLI flags: `--status`, `--updates <id>` (reuses the original
      identifier as `mets/@OBJID`), `--content-category` with a configured
      default landing in `sip.Spec.Type`.
- [x] Config addition regenerated: `go generate ./cli` → `CONFIG.md`.
- [x] Validator loop: a `--status replacement --updates <id>` fixture run
      conforms (SIP3), both profiles stay VALID.
- [x] Gate: builds green, baseline clean (defaults leave output unchanged).

### I5 — per-representation descriptive

- [x] Domain slots: `sip.Representation.Description`/`DescriptionFile`.
- [x] Writer step: rep `metadata.csv` terms → rep
      `metadata/descriptive/*.xml` in the canonical emission order.
- [x] Rep-METS template: dmdSec referencing it (CSIPSTR12/13).
- [x] Fixture with a rep-level `metadata.csv`; validator loop until VALID.
      *(The eark fixture carries one permanently, so every `./build.sh
      eark` exercises the rep dmdSec; basic verified via a scratch fixture
      at 2.0.4 — the standing basic fixture stays rep-descriptive-free to
      keep the baseline maximally sensitive.)*
- [x] Producer label homed (added in review, 2026-08-19): the operator's
      representation label is emitted as the rep METS `mets/@LABEL`
      (spec §2's "keeps your label as the human-readable name").
      `sip.Representation` split `Name` (package-side: directory, OBJID,
      paths) from `Label` (the human-readable name) so field names match
      the METS attributes they feed. **Deliberate baseline re-bless** —
      the reviewed delta was exactly `LABEL=""` → `LABEL="master"`;
      recorded in `tmp/baseline/README.md`.
- [x] Gate: builds green; baseline clean (basic fixture carries none unless
      deliberately added — if added, conscious re-bless).

### I6 — received PREMIS pass-through

- [x] Graph slots for received-premis file lists on `sip.Package` and
      `sip.Representation` (nodes copied, never parsed).
- [x] Acceptance check per the Design note: well-formed XML, root
      `premis:premis` in the PREMIS 3.0 namespace — no XSD validation.
- [x] Writer: copy under `metadata/preservation/` (package and rep level);
      amdSec/digiprovMD references in the templates.
- [x] Coexistence verified: received files alongside the *generated*
      PREMIS in the same amdSec.
- [x] Fixture with received PREMIS at both levels; validator loop until
      VALID.
- [x] Gate: builds green, baseline clean.

### I6b — representation documentation (added in review, 2026-08-20)

The gap between I5 and I6: spec §4's representation-level documentation
was never scheduled, guarded only by a create-time warning.

- [x] `sip.Representation.DocumentationFiles`;
      `SourceRepresentation.Documentation`; the package-level
      documentation-node logic became the shared
      `assembleDocumentationNodes` (same ADR-0009 leniency at both levels).
- [x] Writer copies under `representations/<name>/documentation/`; rep METS
      gains the `fileGrp USE="Documentation"` and structMap Documentation
      division, mirroring the package METS.
- [x] `reportUnsupported` deleted — no legal input goes unpackaged any more.
- [x] The eark fixture carries rep-level documentation permanently; a basic
      scratch run with documentation at both levels validates
      **VALID, 0 warnings, 132 passed** — the first fully clean basic
      verdict: CSIPSTR16 clears.
- [x] Gate: builds green, baseline clean (the standing basic fixture is
      unchanged).

### I7 — docs and retirement

- [ ] [input-spec.md](../input-spec.md): status flips from draft to
      current; §5 wording revised (MUST well-formed PREMIS 3.0-namespaced,
      SHOULD schema-valid); §1's check-mode sentence scoped honestly
      (check validates structure and metadata rules; content verification —
      characterization bindings, premis conformance — happens at build;
      decided in the 2026-08-20 I6 review).
- [ ] [ADR-0010](../decisions/0010-config-over-self-describing-input.md)
      status → Accepted (if not already flipped at I3).
- [ ] Design doc: input-contract section rewritten; CLI/library known-gap
      paragraph updated (input discovery now CLI-side, builder data-fed).
- [ ] README: usage, input requirements, `check` command, new flags.
- [ ] CLAUDE.md: system-shape input description updated.
- [ ] TODO housekeeping: input-contract, `mets/@TYPE`, and rep-naming items
      close; library-API item trimmed of what this arc delivered.
- [ ] This plan retires to `archive/` per the docs lifecycle.
- [ ] Gate: final `./build.sh basic` + `eark` run green.

## Verification

1. `./build.sh basic` and `./build.sh eark` **VALID** after every step; the
   I3 re-bless is the only baseline change in the arc, and only the
   descriptive XML may differ in it.
2. Go tests: walker rules (each MUST as a table case — symlink, misplaced
   content, reserved-name misuse, NFC mismatch pair), CSV decode (BOM, CRLF,
   quoting, repeated keys, `[lang]`, unknown key, missing required key),
   collect-all (a folder with three violations reports all three), terms
   encoding shape, stable file ordering into the structMap.
3. The data-not-files contract, pinned by a test: a build driven by a
   hand-constructed `profiles.Input` — `Terms` and `Report` supplied as
   values, essence as plain source paths, no `metadata.csv` or
   `siegfried.json` anywhere on disk — assembles the same graph the folder
   path does. This is the embedding-caller flow and must not regress.
4. Negative paths by hand: `check` on a broken fixture lists every finding
   and exits non-zero; `create` on the same folder writes nothing.
5. I4–I6 each get a validator loop over a fixture exercising the new
   emission (a `--status replacement` run; a rep with its own
   `metadata.csv`; a package with received PREMIS at both levels).
6. The old contract is gone: no `dc+schema.json` reference outside git
   history and the archive docs.

## Out of scope, recorded

Everything in the spec's §8 deferred list (explicit manifest,
operator-supplied descriptive XML, multiple intellectual entities, BagIt
input, per-package config overrides), plus the remaining library-API arc:
stream-first essence, supplied fixity, and `Package.Validate()`'s
graph-level checks — identifier uniqueness, the no-empty-`Mime` invariant
([TODO.md](../TODO.md) keeps ownership; the *input-data* validation slice
was pulled into I2/I3, see "Validation splits in two"). PREMIS *events* for received files
stay with the format-provenance design question.
