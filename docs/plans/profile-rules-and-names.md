# Plan: the eark family offers CSIP as written, and profile names by family

*Status: **in progress, reduced** (drafted 2026-10-05). Step 1 landed in 7fc5626
([ADR-0029](../decisions/0029-eark-profiles-offer-csip-as-written.md),
[ADR-0030](../decisions/0030-profile-names-by-family.md)). Steps 6 and 7, the renames,
follow, one commit each. The plan builds on 5074f49 (the escaping series, ADR-0028);
line numbers refer to that tree and are out of date since 2026-10-06. Step 6 landed in
f5d2e7f; step 7 lands with the commit that renames the registry keys. Update this line
as steps land.*

*Parked 2026-10-06: steps 2 to 5 and 8.* They implement ADR-0029's two packages, one
with essence and no descriptive metadata (`eark/none`) and one with metadata only. Neither
is needed within UGent now ([ADR-0033](../decisions/0033-ugent-first-profiles-of-your-own.md)),
and whether UGent's RODA accepts either is unchecked. When they resume, three things
changed on 2026-10-06 that the steps below do not reflect: `build.New` checks the model's
format name, so step 3 must skip that check for a nil model; schemas are `build.Schema`
values (`BundledSchemas(mets.Schemas...)` plus the model's), and the registry test's
`shipped` reads names through `schemaNames`; and step 8 copies the eark profile's neutral
comments, which no longer name RODA. With step 8 parked, step 7 lists three names, not
four.

## Context

Today the registry has `basic` (Meemoo `dc+schema.xml`), `eark` (Simple Dublin Core
`dc.xml`) and `eark-mods` (MODS 3.7 `mods.xml`). Two rules are the engine's under every
profile: a package carries a package-level description (ADR-0025, accepted 2026-10-05)
and a package has at least one representation (`SourcePackage.Validate`, never recorded
in an ADR). Both go further than CSIP, which is what the eark profiles claim to produce.

Decided on 2026-10-05 (the reasoning is in the two ADRs; this plan carries only what the
steps need):

1. **A plain E-ARK SIP profile without descriptive metadata**: the E-ARK SIP as CSIP
   defines it, with no descriptive policy of the tool on top. CSIP17 (`mets/dmdSec`,
   0..n SHOULD) and CSIPSTR7 apply only "if descriptive metadata ... is available";
   CSIP92 (the Metadata div's `DMDID`, 0..1 SHOULD) references dmdSecs that exist; the
   Metadata div itself is CSIP88, 1..1 MUST. E-ARK SIP 2.2.0 §3.3 adds nothing for
   `dmdSec`.
2. **A metadata-only package under `eark/dc` and `eark/mods`**: zero representations.
   CSIP58: "In the case that a package only contains metadata updates, i.e. exclusively
   metadata files, then no file references need to be added to this section." CSIP105
   (representation divs) 0..n SHOULD; CSIPSTR9 (the `representations` folder) SHOULD.
   E-ARK SIP 2.2.0 introduction: "A package with zero representations means that it only
   contains metadata." The tool does not tie zero representations to an update status,
   because CSIP does not; the usual case is `--status` plus `--updates`.
3. **Profile names carry the family and the one thing the family's profiles differ in:**
   `eark/none`, `eark/dc`, `eark/mods`, `meemoo/basic`. Go package names cannot carry the
   slash: `profiles/eark` (none), `profiles/earkdc`, `profiles/earkmods`,
   `profiles/meemoo`.

Both engine rules become profile values, in the way ADR-0026 set for profile rules: a nil
`build.Definition.Model` means no descriptive metadata; a new `MinRepresentations` next
to `MaxRepresentations`. Per profile:

| profile | descriptive metadata | representations |
|---|---|---|
| `meemoo/basic` | Meemoo `dc+schema.xml`, required | exactly one (min 1, max 1) |
| `eark/none` | none | at least one |
| `eark/dc` | Simple DC, required | any number, including none |
| `eark/mods` | MODS 3.7, required | any number, including none |

**Validator facts that shape two details** (commons-ip 2.11.2 source, read 2026-10-05;
the spec allows either way): it passes a package with no Representations file group
(CSIP114 only checks the files of groups that exist), a Metadata div without `DMDID` when
there is no dmdSec (CSIP92), and no dmdSec when `metadata/descriptive` holds no files
(CSIP17). It warns when the `metadata/descriptive` or `representations` folder is
missing (CSIPSTR7, CSIPSTR9). So the writer's skeleton keeps creating both folders, empty
when unused, as it already does for `metadata/preservation`.

The profile name reaches no output file (`Definition.Name` is read only in error strings
and the registry), so the three existing profiles stay byte-identical and
`scripts/reference-diff.sh` must stay clean throughout. `go test ./...` passes after
every step.

## Steps

### Step 1. Two ADRs and status lines (landed, 7fc5626)

`Added: ADR-0029 and ADR-0030: the eark profiles offer CSIP as written (description and representation minimum are profile rules) and profile names carry the family`

ADR-0029 supersedes ADR-0025 and extends ADR-0026; ADR-0030 extends ADR-0015. Status
lines on the three in ADR-0017's form.

### Step 2. METS template tolerates no description (output unchanged)

`Changed: the package METS writes dmdSec and DMDID only when the entity has a descriptive file`

- `encoders/mets/encoder.go` package template: wrap the `<!-- ref to descriptive metadata
  about IE -->` comment and the `range .DescriptiveFiles` in `{{- with .DescriptiveFiles
  }} ... {{- end }}`, the form the PREMIS block below it uses; the Metadata div (l.218)
  becomes `LABEL="Metadata"{{ with .Root.DescriptionFile }} DMDID="{{ esc .Identifier
  }}"{{ end }}`, as the representation template already does (l.124).
- `sip/package.go` `DescriptiveFiles`: nil when `p.Root.DescriptionFile == nil`.
  `sip/entity.go` `DescriptionFile` doc: nil when `Description` is nil.
- Test in `encoders/mets/encoder_test.go`: a package whose root has no description file
  renders no `<dmdSec`, no `DMDID`, one `LABEL="Metadata"`, well-formed.
- `./scripts/reference-diff.sh` clean for all three.

### Step 3. The engine builds without a model

`Changed: a definition without a metadata model builds a package without descriptive metadata; a model requires a package-level description in ValidateSource`

- `build/model.go` `checkDescriptions`: nil model refuses any description (package: "the
  profile carries no descriptive metadata, yet a description was supplied";
  representation: named). Its doc: a missing package description is now
  `ValidateSource`'s finding.
- `build/definition.go` `ValidateSource`: after `checkDescriptions`, `d.Model != nil &&
  source.Description == nil` adds "profile %q needs a package-level description; none was
  supplied" to the joined errors. The nil-model refusal runs first so the
  `AllowRepresentationDescriptions` message is never reached for it. Docs on `Model`
  (nil: no descriptive metadata, ADR-0029), `DocumentName` (unused without a model),
  `AllowRepresentationDescriptions` (irrelevant without a model).
- `build/source.go` `Validate`: drop "no descriptive metadata supplied"; run
  `Validate`/`ValidateRequired` only when a description is present, with a comment naming
  the owner of the "needs one" rule.
- `build/builder.go` `New`: drop the nil-model refusal; `Config.Profile` doc loses "It
  must name a metadata model".
- `build/assemble.go`: `assembleDescriptive` only with a model; the schema list is
  `mets.Schemas` plus `Model.Schemas()` when there is a model (a small helper next to the
  caller). The representation branch keeps its `sr.Description != nil` guard with one
  comment line.
- `build/write.go`: `writeDescription` for the package only when the root has a
  description, the guard `writeRepresentationMetadata` already uses. `writeSkeleton`
  unchanged (see the validator facts).
- `build/doc.go`: a profile may carry no descriptive metadata.
- Tests (`build_test`): `noneDef(t)` helper (an eark definition with `Model = nil`,
  `DocumentName = ""`, `AllowRepresentationDescriptions = false`, name `test/none`; step 8
  switches it to the registry). `TestValidateSourceWithoutModel`,
  `TestValidateSourceRequiresDescriptionWithModel`, `TestBuildWithoutModel` (no
  `DescriptionFile`, empty `DescriptiveFiles`, schema files equal `mets.Schemas`,
  `metadata/descriptive` exists and is empty, package METS without `<dmdSec` and `DMDID`
  but with `LABEL="Metadata"`, representation METS likewise). Delete
  `TestNewRefusesDefinitionWithoutModel` and the `"no descriptive"` case of
  `TestSourcePackageValidate`; add a nil-description acceptance there.
  `TestBuildRejectsDescriptionOfAnotherStandard`: a nil-model case.
  `profiles/registry_test.go`: `shipped` returns `mets.Schemas` for a nil model; split
  `TestRegistryEntriesNameAModel` into a key-equals-name test and a table of which names
  carry a model; the schema test skips a nil model. `cli/profile_test.go`: guard
  `def.Model != nil` for now.
- Docs: design doc Profiles ("the one behavior a profile brings is its metadata model, or
  none"), Build lifecycle (drop "refuses a profile without a metadata model"; the Check
  paragraph), Code organization; README "Profiles of your own": a profile without
  descriptive metadata leaves `Model` nil.

### Step 4. The input reader without a model

`Changed: input.Read takes a nil mapper with a nil model for a profile without descriptive metadata; description.csv is then a violation at every level`

- `cli/input/package.go` `Read`: replace the nil-mapper error with an agreement check,
  `(mapper == nil) != (documentSpec.Model == nil)`: today's "no mapper" message for a
  model without a mapper; the reverse says the profile carries no descriptive metadata,
  so there are no rows to map. Set `folderReader.takesDescription = mapper != nil`.
- `cli/input/walker.go`: field `takesDescription bool` (zero value strict, like
  `AllowRepresentationDescriptions`). `levelDescription`: when `!takesDescription` and
  rows exist, violate "`<path>`: the profile carries no descriptive metadata; remove the
  file (input specification §3)"; the missing-description case applies only when
  `takesDescription`. `found.document` is always empty here, since a nil model implements
  no `DocumentFormat`. `violateMissingDescription` cites `Definition.ValidateSource` and
  ADR-0029.
- `cli/input/document.go`: one sentence on the pairing.
- `cli/profile.go`: `mapperFor(def) (input.Mapper, error)`: `nil, nil` for a nil model,
  else today's lookup; `resolveProfile` calls it; the `mappers` doc says a profile without
  a model has no entry.
- Tests: `read_test.go` `TestReadWithoutDescription` (no description, no violation;
  `description.csv` at root and under a representation are violations naming the path; a
  `dc.xml` at root is content). `description_test.go`: `TestReadRequiresAMapper` becomes
  the two-way agreement test; `TestMapperErrorsNameTheLine` passes `earkDocumentSpec`.
  `walker_test.go`: the three `folderReader` literals set `takesDescription: true`.
  `cli/profile_test.go`: the every-profile-has-a-mapper test skips a nil model and asserts
  no entry; `TestExamplesBuild` uses `mapperFor` and `filepath.FromSlash(name)`.

### Step 5. The representation minimum is a profile rule

`Changed: MinRepresentations on the definition replaces the engine's at-least-one rule; eark packages may carry metadata only`

- `build/definition.go`: field `MinRepresentations int`: "the number of representations a
  package needs at least; zero allows a package without any, a metadata-only package
  (CSIP58, E-ARK SIP 2.2.0 introduction). Meemoo SIP 1.2's basic profile needs exactly
  one: 1 here and in MaxRepresentations." `MaxRepresentations` doc loses "With the one
  representation every package needs (SourcePackage.Validate)". `ValidateSource`:
  `len(source.Representations) < d.MinRepresentations` adds "profile %q needs at least %d
  representation(s), the package has %d".
- `build/source.go`: drop "no representations supplied"; the `Representations` doc
  becomes "the content; the profile says how many a package needs
  (Definition.MinRepresentations)". "representation %q has no content files" stays.
- Profiles: `profiles/meemoo/profile.go` `MinRepresentations: 1` (comment: exactly one,
  with the maximum); `profiles/eark/profile.go` and `profiles/earkmods/profile.go` leave
  it zero with a comment citing CSIP58 and the E-ARK SIP introduction. (Step 8 gives
  `eark/none` a minimum of 1: without a description and without content there is no
  package.)
- `cli/input/walker.go`: the flat branch (`readFlatRepresentation`) makes no
  representation when the folder has no content files, instead of one with no files plus
  "the folder contains no content files"; `readRepresentations` reads an empty
  `representations/` as zero representations instead of a violation. An empty
  representation folder stays a violation. `representations.csv` rules unchanged ("no
  rows" still applies). Comments say the count is the profile's verdict, in
  `ValidateSource`.
- `build/write.go` `writeSkeleton`: unchanged; `representations/` is always created.
- Tests: `build/assemble_test.go` drop the `"no representations"` case of the `Validate`
  table (or assert acceptance); `TestBuildInvalidSourceWritesNothing` still fails under
  `basicDef` through `ValidateSource`, note why in the test. `build/definition_test.go`
  `TestValidateSourceAppliesProfileRules`: "needs at least 1" for basic, zero accepted
  for eark. New `TestBuildMetadataOnly` under `earkDef`: no representations; the package
  METS has no `fileGrp` with `USE="Representations/..."` and no representation div, keeps
  Schemas and Metadata; `representations/` exists and is empty; the zip carries the
  directory entry. `cli/input/read_test.go`: `TestReadNoContent` becomes "only
  `description.csv` reads as zero representations, no violation";
  `TestReadEmptyRepresentationsDir` likewise; `TestReadCollectsAllViolations` loses the
  empty-folder finding if it relied on it.
- Docs: `docs/input-spec.md` §2 ("Every package has at least one" becomes the profile
  table; the simple case with no content is a metadata-only package), the update section
  (a metadata-only update: `description.csv` plus `--status` and `--updates`); design doc
  (profile table gains a representations column; layout comment "one directory per
  representation, none in a metadata-only package"; Profiles; Build lifecycle check
  paragraph); README ("smallest valid input", "Updating an earlier package" gains the
  metadata-only case); CONTRIBUTING: how to validate a metadata-only package with
  build.sh's second argument.

### Step 6. Simple DC moves to `profiles/earkdc`

`Changed: profiles/eark moves to profiles/earkdc; eark.Terms is earkdc.Terms and mapping.Eark is mapping.EarkDC`

- `git mv profiles/eark profiles/earkdc`; package clause and doc ("Package earkdc is the
  eark/dc profile"); `Definition.Name` stays `"eark"` until step 7 so the
  key-equals-name test holds.
- `git mv cli/input/mapping/eark.go earkdc.go`; type `EarkDC`.
- Fix imports and references: `profiles/registry.go`, `cli/profile.go`,
  `build/*_test.go`, `build/doc.go`, `build/source.go` comments, `sip/description.go`
  comment, `cli/input/*_test.go`, `mapping/mapping_test.go`; README (`earkdc.Terms`,
  import path), design doc, CLAUDE.md package map.
- Commit body records the import-path break, as ADR-0015 and ADR-0018 did.

### Step 7. Rename the registry keys; examples and scripts follow

`Changed: profiles are named by family and descriptive standard: meemoo/basic, eark/dc, eark/mods; examples, scripts and docs follow`

- Go: registry keys and package doc; `Definition.Name` in the three profile packages and
  their package docs; mapper error strings in `cli/input/mapping/meemoo.go` and
  `earkmods.go` interpolate `Definition.Name`; comments in `main.go`, `build/source.go`,
  `build/document.go`, `cli/input/walker.go`, `cli/input/document.go`. Tests that spell a
  name: `profiles/registry_test.go`, `build/definition_test.go`,
  `build/assemble_test.go`, `build/document_test.go`, `build/example_test.go`,
  `cli/input/document_test.go`, `cli/input/source_package_test.go`.
- `examples/`: `git mv` to `examples/meemoo/basic`, `examples/eark/dc` (through a
  temporary name), `examples/eark/mods`; `examples/README.md`.
- `build.sh`: default `meemoo/basic`; `NAME="${PROFILE//\//-}"` for everything that is
  one path component (`OUT="$NAME-uuid"`, the reports `run_dir`, which
  `scripts/publish-report.sh` globs as `runs/*/run.json`); the `case` maps `meemoo/basic`
  to 2.0.4 and `eark/dc|eark/mods` to 2.2.0; header comments. `SRC="tmp/build/$PROFILE"`
  may stay nested.
- `.gitignore`: the three `<profile>-uuid/` lines become `*-uuid/`.
- Comments in `scripts/validate.sh`, `scripts/reference-diff.sh`,
  `scripts/schema-catalog.xml`.
- `tmp/reference/` (local, untracked): move `pkg/` to `meemoo/basic/pkg`, `eark/pkg` to
  `eark/dc/pkg`, `eark-mods/pkg` to `eark/mods/pkg`; a dated paragraph in its README. Say
  so in the commit message.
- Docs: README (intro, comparison table, headings, `--profile` examples, the input-spec
  link anchor, which GitHub renders as
  `#supplying-a-finished-document-earkdc-and-earkmods`), CONTRIBUTING.md, CLAUDE.md
  (package map and profile mentions), `docs/input-spec.md`, `docs/sip-creator-design.md`,
  `docs/TODO.md`, `docs/plans/mods-coverage.md`, `docs/development-roda-ingest.md`;
  `docs/plans/eark-writer.md` gets one note line at the top, body untouched. ADRs and
  `docs/archive/` untouched. CONFIG.md needs no regeneration.
- `./build.sh meemoo/basic`, `eark/dc`, `eark/mods` VALID; reference-diff clean against
  the moved copies.

### Step 8. The `eark/none` profile

`Added: the eark/none profile, a plain E-ARK SIP without descriptive metadata`

- New `profiles/eark/profile.go` (the package's only file, with the package doc): `Name:
  "eark/none"`, `Model: nil` with a comment citing CSIP17, CSIPSTR7 and ADR-0029, no
  `DocumentName`, `AllowRepresentationDescriptions` false, `MinRepresentations: 1` with
  its reason, `Emit*Premis` false and `EmitRepresentationType` true with the existing
  RODA comments, `Declaration` copied from `profiles/earkdc/profile.go` (no cross-profile
  import, per ADR-0007).
- `profiles/registry.go`: `"eark/none": eark.Definition`.
- `examples/eark/none/`: one essence file under `representations/master/`,
  `representations.csv`, a `documentation/README.txt` like the other examples, no
  `description.csv`; `examples/README.md` row.
- `build.sh` `case`: `eark/none|eark/dc|eark/mods` map to 2.2.0; the error list.
- Tests: `noneDef` reads `profiles.Get("eark/none")`; registry tests list it without a
  model and with the METS schema set only; `TestExamplesBuild` picks it up through
  `profiles.Names()`; `ValidateSource` refuses it zero representations.
- Docs: README (four profiles; table column; a short section on `eark/none`),
  `docs/input-spec.md` (§1: under `eark/none`, `description.csv` must not be present;
  §3; describing a representation; §7), design doc (profile table row; layout comments;
  Profiles; Validation), CONTRIBUTING.md and CLAUDE.md ("all four profiles"),
  `docs/TODO.md` validator status once validated.
- Validation: `./build.sh eark/none` VALID; copy the package to
  `tmp/reference/eark/none/pkg` and note it in that README and the commit message.

## Verification

- `go test ./...` after every step.
- `./build.sh <profile>` reports `VALID` for `meemoo/basic`, `eark/dc`, `eark/mods` (step
  7 on) and `eark/none` (step 8). `scripts/validate.sh` takes `CSIP_CMD` to run
  commons-ip on host Java instead of docker.
- Metadata-only package (step 5): a folder holding only a `description.csv`, built with
  `./build.sh eark/dc <folder>` (build.sh takes the input folder as its second argument),
  reports `VALID`; the same under `meemoo/basic` is refused by `check` with "needs at
  least 1 representation(s)".
- `./scripts/reference-diff.sh` clean for the three existing profiles after steps 2, 3,
  5, 6 and 7 (their output never changes); a new reference copy for `eark/none` after
  step 8.
- `xmllint` on `eark/mods` `mods.xml` as today.
- Manual: `sip-creator check --profile eark/none examples/eark/none` passes; the same
  folder with a `description.csv` added is refused naming the file; `--profile eark`
  reports an unknown profile and lists the four names.
