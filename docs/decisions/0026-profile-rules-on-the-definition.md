# 0026 — A profile's rules on the package are definition values the library checks

Status: **Accepted** (2026-10-05). **Extended by
[ADR-0029](0029-eark-profiles-offer-csip-as-written.md)** (2026-10-05):
`MinRepresentations` joins `MaxRepresentations`, so a profile also states the least a
package needs.

## Context

Meemoo SIP 1.2's basic profile has two requirements the tool did not enforce: "The IE
MUST be represented by exactly one representation" and "There MUST NOT be any
descriptive metadata at the representation level." The tool accepted several
representations and a representation's `description.csv` under every profile, so a
producer following the input specification could build a `basic` package that breaks
Meemoo's profile. The eark profiles have neither rule.

Neither rule is about the descriptive standard: the number of representations has
nothing to do with dc+schema.org. Both belong to the content profile.

The `check` command runs the input reader and never builds. The reader takes the
profile's mapper and its `input.DocumentSpec`, not the definition, so it cannot see a
rule that lives only on the definition.

## Decision

A profile's rules on the shape of the package are values on `build.Definition`:
`MaxRepresentations` (zero for no limit) and `AllowRepresentationDescriptions`. The
Meemoo definition sets a maximum of 1 and leaves descriptions on representations
refused; the eark definitions allow them. The zero value of the boolean is the strict
one, so a new profile refuses a representation's description unless it opts in.

`Definition.ValidateSource` checks them, after the existing check that the descriptions
belong to the profile's metadata model, and returns one error per finding, joined.
`Builder.Build` runs it first. The `check` command runs it once `input.Read` reports
no violations and returns its error as it is, one finding per line. The reader does not
repeat these rules.

## Alternatives rejected

- **An optional interface on the metadata model**, like `build.DocumentFormat`. The
  reader can already reach the model through `DocumentSpec.Model`, so `check` would
  need no new call. But it fits only the description rule; the representation count is
  not the model's.
- **Pass the values to `input.Read` and repeat the rules in the reader**, so the
  messages name the file. The rule would then exist twice, and `Read` would take a
  fourth argument or `DocumentSpec` would stop describing only the document. The
  library's message names the representation, which is its folder name, so the
  operator can still find the file.

## Consequences

- `check` and `create` refuse the same packages for these rules, from one function.
  `check` reports them only once the folder has no violations of its own, so a folder
  with both kinds of problems takes two runs.
- A further rule of this kind is a field on `Definition`, a line in `ValidateSource`
  and a value in the profile package. An institution's own profile
  ([ADR-0022](0022-reference-implementation-bring-your-own-profile.md)) sets the fields
  it needs.
- The findings read as library messages ("representation "access" has a description;
  profile "basic" allows one at the package level only"), not as the reader's, which
  name a file and a line.
