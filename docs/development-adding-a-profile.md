# Adding a profile

This guide lists what to decide and which files to change to add a profile to this repository, so that `--profile` and `profiles.Get` accept it. It is for whoever adds the profile, and the `add-profile` skill in `.claude/skills/` follows it. A program that keeps a profile of its own, outside this repository, follows "Profiles of your own" in the [README](../README.md#profiles-of-your-own) instead ([ADR-0033](decisions/0033-ugent-first-profiles-of-your-own.md)).

A profile is a `build.Definition` with a metadata model ([sip-creator-design.md](sip-creator-design.md#profiles)). Most of the work is the decisions below. The code is a definition, one registry line and one mapper entry, unless the profile needs a new descriptive standard.

## Decisions

Make these before you change a file. Each one becomes a value on the definition. The fields are documented on `build.Definition` in `build/definition.go`.

- **Owner and name.** A profile is a content type, named `<owner>/<type>` ([ADR-0034](decisions/0034-a-profile-is-a-content-type.md)). The owner is `meemoo`, `ugent`, or a new owner with a new package under `profiles/`.
- **Specification.** The E-ARK SIP version the package follows, and with it the profile URL the METS declares. A Meemoo profile follows Meemoo's specification, which names the E-ARK version.
- **Descriptive standard.** One of the existing models (Simple Dublin Core as `ugent.Terms`, MODS 3.7 as `ugent.Record`, Meemoo's elements as `meemoo.Terms`), or a new one. Also: whether the profile accepts a finished document supplied as a file, and the document's file name.
- **Representations.** The minimum and maximum number, whether representation names come from a closed vocabulary, whether a representation may carry its own description, and whether its type goes into the METS.
- **Preservation metadata.** Whether the tool writes PREMIS for the package and for each representation.
- **METS declaration.** The content information type, the default content category, any agents, and whether the submitter needs an OR-id.

For a UGent profile, these decisions are the rules on its profile page (step 7). Write the page's rules first and quote them in the definition.

## Steps

1. **Definition.** Add an exported `build.Definition` to `profiles/<owner>/profile.go`. Comment each value with the rule that requires it, quoted from the profile page or the owner's specification, as `profiles/ugent/profile.go` does. A new owner gets a new package with a package doc that says what its profiles are for.
2. **Metadata model**, only for a new descriptive standard. Put it in the owner's package, next to the definition:
   - a description type that implements `sip.Description`, whose `Validate` and `ValidateRequired` carry the standard's rules
   - a model that implements `build.MetadataModel`, writing the document with `text/template`
   - `ValidateDocumentRoot` (`build.DocumentFormat`) if the profile accepts a supplied document
   - `Swap` (`build.IdentifierSwapper`) if the document must state the identifier the build mints, as Meemoo's does
   - the XSDs the document points at, in `schemas/`, returned from `Schemas` through `build.BundledSchemas`

   Describe the model in the descriptive metadata section of [sip-creator-design.md](sip-creator-design.md).
3. **Registry.** Add one line to `registry` in `profiles/registry.go`. In `profiles/registry_test.go`, add the profile to `sampleDescriptions`, to the expected schemas in `TestRegistrySchemas`, and to `TestRepresentationDefinitions`. `TestRegistryDescriptiveDocumentsPointAtShippedSchemas` fails until the first one is there.
4. **Mapping.** Add the profile to `mappers` in `cli/profile.go`, with an existing mapper from `cli/input/mapping/` or a new one for a new model. A new mapper gets tests in `cli/input/mapping/mapping_test.go`. `TestEveryProfileHasAMappingItsModelAccepts` fails until the profile has a mapper.
5. **Example.** Add an input folder at `examples/<owner>/<type>/`, with invented metadata and no `siegfried.json`, and a row in the table in `examples/README.md`. `TestExamplesBuild` fails until the folder exists and builds.
6. **build.sh.** Add the profile and its E-ARK version to the `case` that sets `SPEC_VERSION`. If the profile writes a kind of descriptive document that no other profile writes, add a `validate_documents` line with the schema the package ships for it. The comment at the top of `scripts/validate.sh` lists the profiles per E-ARK version.
7. **Documentation:**
   - a UGent profile gets a page at `docs/profiles/<owner>-<type>.md`, with the sections of [ugent-basic.md](profiles/ugent-basic.md). Meemoo specifies its own profiles, so they get no page here.
   - [README.md](../README.md): a column in the table under Profiles, and a section for the profile
   - [input-spec.md](input-spec.md): the profile's row in the keys table and its paragraph in §3
   - [sip-creator-design.md](sip-creator-design.md): the profile table at the top and the Profiles section
   - the profile list in the package comment of `main.go`
   - the `profiles/` row in the System shape table of [CLAUDE.md](../CLAUDE.md)

   Write an ADR as well if the profile makes a choice a later reader would question, following the rule in [README.md](README.md).
8. **Checks.** `go test ./...` passes, and `./build.sh <owner>/<type>` reports `VALID` with no warnings. The `validate-profiles` skill runs both for every profile. Then copy the built package directory to `tmp/reference/<owner>/<type>/pkg` and record the new reference copy in `tmp/reference/README.md`.
