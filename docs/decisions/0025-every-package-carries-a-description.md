# 0025 — Every package carries a package-level description

Status: **Superseded by [ADR-0029](0029-eark-profiles-offer-csip-as-written.md)**
(2026-10-05): whether a package carries a description is the profile's rule, stated by
its metadata model; `eark/none` carries none. Was Accepted (2026-10-05), recording a
rule the code enforced.

## Context

`SourcePackage.Validate` refuses a package without a package-level description ("no
descriptive metadata supplied") under every profile, and the input reader reports the
same rule so that `check` finds it. Each profile's `ValidateRequired` then asks for at
least an identifier and a title.

The specifications differ per profile:

- **E-ARK CSIP 2.2.0**, CSIP17 (`mets/dmdSec`): 0..n, SHOULD. "Must be used if
  descriptive metadata for the package content is available." CSIPSTR7 says the same of
  the `metadata/descriptive` folder. **E-ARK SIP 2.2.0** adds nothing (section 3.3).
- **Meemoo SIP 1.2**, basic profile: "The `/descriptive` directory at the package level
  MUST contain exactly one metadata file `dc+schema.xml` that describes the IE."

So under `basic` the rule is Meemoo's. Under `eark` and `eark-mods` a package without a
dmdSec would be a valid E-ARK SIP, and the rule goes further than the specification.

## Decision

Every package carries a package-level description, under every profile. For `eark`
and `eark-mods` this is the tool's own policy, not a CSIP requirement: the descriptive
standard (Simple Dublin Core, MODS 3.7) is the one thing these profiles choose
([ADR-0015](0015-descriptive-worlds-dc-and-mods.md)), and a package in either profile
without a document in that standard serves no purpose the profile exists for.

## Alternatives rejected

- **Follow CSIP17 under the eark profiles and make the description optional there.**
  The package would validate, but the archive would receive essence that nothing
  describes, and the producer would have chosen a descriptive standard to supply no
  document in it. It would also make the requirement a value per profile, in
  `SourcePackage.Validate`, the input reader and input specification §3, for a case no
  user of the tool has asked for.

## Consequences

- A reader who compares the eark profiles with CSIP17 finds the tool stricter than the
  specification. This ADR is the answer; do not relax the rule without revisiting it.
- An institution whose own profile does allow packages without a description
  ([ADR-0022](0022-reference-implementation-bring-your-own-profile.md)) needs a change in
  `build`, not only a new profile package: the rule is the engine's, not a profile value.
