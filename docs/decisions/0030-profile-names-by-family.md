# 0030 — Profile names carry the family and the one choice the family leaves open

Status: **Accepted** (2026-10-05). Extends
[ADR-0015](0015-descriptive-worlds-dc-and-mods.md): a profile still names its
descriptive standard; the name now also carries the family.

## Context

The registry named three profiles from two vocabularies. `basic` is Meemoo's name for a
content profile and says nothing without "Meemoo" in front of it; `eark` and `eark-mods`
named a family and, for one of the two, a standard.
[ADR-0029](0029-eark-profiles-offer-csip-as-written.md) adds a fourth profile, the
E-ARK SIP without descriptive metadata, and Meemoo SIP 2.1 defines four content profiles
(basic, bibliographic, material artwork, film), so the registry will grow on both sides.

## Decision

A profile name is `<family>/<choice>`. The family is the specification the package
conforms to beyond CSIP: `eark` for the plain E-ARK SIP, `meemoo` for Meemoo's SIP. The
choice is the one thing the family's profiles differ in. For `eark` that is the
descriptive standard: `eark/none`, `eark/dc`, `eark/mods`. For `meemoo` it is the content
profile Meemoo names: `meemoo/basic`.

The slash is part of what `--profile` takes. Go package names cannot carry it, so the
package is the name without the slash, with `none` dropped: `profiles/eark` holds
`eark/none`, `profiles/earkdc` holds `eark/dc`, `profiles/earkmods` holds `eark/mods`,
`profiles/meemoo` holds `meemoo/basic`.

The absence of descriptive metadata is spelled out as `none`, so that it is a choice an
operator types knowingly. The family name alone is not a profile.

## Alternatives rejected

- **`eark/general` and `eark/bibliographic`**, naming what the data is about. The tool
  never knows that. It knows which elements a description takes, and that is what the
  operator needs when preparing the folder. Simple Dublin Core describes a book as well
  as MODS does, and `bibliographic` is a Meemoo 2.1 content profile name, so the word
  would mean different things on the two sides of the slash.
- **The bare `eark`** for the profile without descriptive metadata. It reads as the
  default, and the one profile nobody should pick by accident is the one that delivers
  essence that nothing describes. An operator with a `description.csv` in the folder
  would be refused instead of served.
- **Hyphens** (`eark-dc`, `meemoo-basic`). They leave the scripts alone, but
  `meemoo-material-artwork` does not show where the family ends.
- **Keeping `basic` without a family.** A second family's basic profile would collide.
- **Keeping `eark` for Simple Dublin Core and adding `eark-none`.** The name would say
  less than its siblings do.

## Consequences

- Old `--profile` values report an unknown profile and list the four names. A program
  that resolves a profile by name through `profiles.Get` changes the name; one that holds
  a definition directly changes the import path: `profiles/eark` becomes
  `profiles/earkdc`, and `eark.Terms` becomes `earkdc.Terms`.
- Scripts that use the name as one path component flatten the slash: `build.sh` derives
  `eark-dc-uuid/` and the report directory from `eark/dc`. Examples and reference copies
  nest by family: `examples/eark/dc/`.
- A package's output does not change: the profile name reaches no file.
- A further Meemoo content profile is `meemoo/<its name>`; a further descriptive
  standard under E-ARK is `eark/<standard>`.
- The design doc, the input specification, README and CONTRIBUTING spell the new names.
  ADRs and `docs/archive/` keep the old names as history.
