# 0018 — The engine lives in build/, each profile in its own package, renderers in encoders/

Status: **Accepted** (2026-09-28). Supersedes in part
[ADR-0015](0015-descriptive-worlds-dc-and-mods.md): the descriptive
standard is an exported interface on the engine, implemented by each
profile package, and the set of standards is closed by the registry rather
than by an unexported field. ADR-0015's descriptive worlds stand; they
moved.

## Context

After the S2 refactor and the simplification of 2026-09-28 the package
layout no longer said what the system is. `profiles/` was nine tenths
engine (builder, assembler, writer, input validation) and one tenth
profile data: a registry with two entries and two descriptive-standard
values. `encoders/` held two kinds of package under one name: `mets` and
`premis`, which render the domain graph and take every profile difference
as data, and `dcschema` and `dc`, which each owned a terms type, a
vocabulary table, validation, a required list, a template and an XSD list.
That is not an encoder; it is everything one profile knows about its
descriptive metadata. `dcschema` carried meemoo's basic-profile namespace,
XSD and rules; `dc` wrapped a real standard, but its choices (the
`simpledc` root RODA reads, no `xml:lang`, identifier and title required)
were the eark profile's.

The two problems were one problem. The engine imported both worlds to
build its standard values, so the generic package depended on the specific
ones, and a package holding a profile could not import the engine without
a cycle. The definitions therefore had nowhere to live but inside the
engine.

## Decision

**The engine is `build/`.** It holds `Builder`, `Config`, `Input` and its
`Validate`, the assemble and write phases, `Definition`, and the interface
a profile plugs in: `DescriptionEncoder` (`Check` and `Encode`) with the
optional `IdentifierSwapper` for a standard whose spec links descriptive
and preservation metadata by a shared identifier. The engine imports no
profile and speaks `sip.Description` only.

**Each profile is one package under `profiles/`.** `profiles/meemoo` is
the `basic` profile and `profiles/eark` the `eark` profile. A profile
package holds its terms type, vocabulary table, rules, template and the
XSDs its document points at, and exports one `Definition` value carrying
the rest as data, with an unexported type implementing the standard. A
new standard is a new profile package; a second profile writing an
existing standard is a second definition in the same package.

**`profiles/` itself is the registry only.** `Get`, `Names` and the map
from name to definition. The set of profiles, and with it the set of
standards, is closed here.

**`encoders/` keeps `mets` and `premis`.** Renderers of the graph, and the
received-PREMIS check next to the PREMIS template.

**Imports run one way.** `cli` → `profiles` → `profiles/*` → `build` →
`sip`, `store`, `schemas`, `encoders/*`. The CLI also imports the profile
packages directly for `Terms` and `ResolveKey`: the file name
`dcschema.csv` already means the meemoo world. (Since later on
2026-09-28, for `Terms` alone: the CLI decodes rows into `sip.Term`
values and wraps them in the world's `Terms`; `ResolveKey` is gone. See
the note of that date on ADR-0015.)

**The engine's tests are an external test package.** `build_test` with an
`export_test.go` that exposes the assembly phase, so the tests keep
building with the real profiles.

## Alternatives rejected

- **Keep the worlds under a neutral name** (`descriptive/`, `vocabularies/`)
  with the definitions in `profiles/`. Two places per profile again, and
  the observation that started this was that the worlds are
  profile-specific.
- **Keep the standard closed by an unexported field on `Definition`.** It
  forces the engine to import the profiles, the direction this decision
  reverses. Closedness by registry is enough: embedding systems take
  definitions from the registry, and a caller who builds a definition of
  their own is off the supported path without breaking anything in the
  engine, which declares in METS whatever the definition says.
- **Put `Definition` and the standard interface in `sip/`** so that profile
  packages need not import the engine. The domain package would carry
  build configuration and a contract with an `io.Writer`; it stays plain
  structs and constructors.
- **A fake standard for the engine's tests** instead of an external test
  package. It loses real-profile coverage in the engine's tests for
  nothing an external package does not give.

## Consequences

- Library callers change again: `profiles.New`, `profiles.Config` and
  `profiles.Input` become `build.New`, `build.Config` and `build.Input`;
  `dcschema.Terms` becomes `meemoo.Terms` and `dc.Terms` becomes
  `eark.Terms`; `profiles.Get` is unchanged. The commit message records
  the break, as the earlier renames did.
- The output of both profiles is unchanged, checked with the structural
  comparison against the reference copies.
- A third profile, `eark-mods`, is a third profile package holding the
  MODS world and a definition that copies the eark declaration values.
- The engine still spells one meemoo constant, `MEEMOO-LOCAL-ID`, where it
  records what the swap returns. Moving it behind the profile is open.
- `Definition` carries no schema list. Each encoder reports the XSDs its
  document points at (`mets.Schemas`, `Schemas()` on the descriptive
  encoder) and the assembler ships that set and nothing else. This
  dropped the six meemoo XSDs the eark package used to ship without
  referencing them, the output change the eark-mods plan had left open;
  the eark reference copy was refreshed with it (2026-09-28).
