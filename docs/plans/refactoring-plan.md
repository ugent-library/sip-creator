# Refactor plan: assemble→emit split + declarative profiles

*Status: accepted, not yet started. Drafted 2026-07-02 from an architectural assessment comparing SIP Creator against commons-ip, RODA-in, bagit-create, Archivematica, and meemoo's own sipin tooling.*

## Context — why this change

The assessment found that SIP Creator's model (`sip/`), serializers (`encoders/`), and format-ID registry (`formats/`) all match how the field structures this problem. The outlier is the profile layer:

1. **Profiles are duplicated imperative pipelines.** `Basic()` and `Roda()` are ~90% line-for-line identical; every comparator tool expresses profile variation as *data over one build path* (commons-ip's `IPContentInformationType` enums, RODA-in's template files, meemoo's own SHACL shapes keyed by `OTHERCONTENTINFORMATIONTYPE`).
2. **Model-building is fused with disk IO.** A `sip.File` node is only born after its bytes hit disk (`createMetadataFile` stats the just-written file); essence nodes are born inside `siegfried.Process` on the *copied* file; `File.Path` is derived by string-slicing against a `p.BaseDir` that is mutated in place (profile.go:53).
3. **Seven ordering dependencies are implicit** (format-ID before PREMIS, rep METS before package METS, package PREMIS before package METS, …), re-hand-sequenced per profile. `Roda()` already broke one (omits rep PREMIS → the known "Preservation metadata file not found" defect in TODO.txt).
4. **Meemoo literals are baked into shared templates** (`TYPE="Photographs – Digital"`, the `/sip/2.0/basic` profile URL, the UGent agent block in encoders/mets/encoder.go:56-60,105-118), which blocks the agreed long-term direction: a genuine E-ARK CSIP base (RODA-acceptable) with meemoo SIP 2.0 as a specialization on top.

Decisions already made:
- Do the **two-phase split first** (assemble graph → writer emits in one canonical order), preserving current meemoo output, validated by `build.sh`.
- Then **lift meemoo values into a declarative `ProfileSpec`** + registry (mirroring the existing `formats/` registry pattern).
- The **E-ARK/RODA writer** is a follow-up phase: this plan creates the seam, not the full E-ARK implementation. Roda is intended to become a *true* E-ARK SIP, not a meemoo variant — so the current broken, CLI-unreachable `Roda()` gets deleted, not migrated.
- **Format identification becomes an optional enricher and essence fixity moves to the store** — companion plan [format-identification-optional.md](format-identification-optional.md) ([ADR-0006](../decisions/0006-format-identification-optional.md)), executed with Phase 1. It amends this plan's details in two places: Step 1's `CopyFile` returns `(Info, error)` (fixity computed during the streamed copy), and Step 3's `p.Formats.Process(src)` becomes an optional `Identify(src)` enrichment of an assembler-built `sip.File`.

Non-negotiables honored throughout: XML stays in `text/template`; new code returns errors (no new panics); METS ID minting stays in the mets encoder; exported surface kept small; `CLAUDE.md` kept current. `CONFIG.md` is untouched by *this* plan's steps, but the companion [format-identification-optional plan](format-identification-optional.md) does relax the `SIP_FILE_FORMAT_*` vars to optional (with the required `go generate ./services` regeneration).

## Target architecture — what it looks like after

```
cli/create_cmd.go
  spec, ok := profiles.Get(flagProfile)        // registry lookup, like formats.New
  pkg, err := profile.Build(spec)              // one engine, two phases:
                                               //   assemble(spec)  → complete sip.Package graph, ZERO writes
                                               //   write(st, pkg, spec) → canonical emission order, back-fills fixity
  archive.Zip(pkg)

profiles/
  profile.go    Profile struct + Build(spec)
  assemble.go   input walk → graph (pure; format-ID runs on SOURCE files)
  write.go      THE canonical emission order, encoded exactly once
  spec.go       Spec type + registry ("basic" → values)
store/
  store.go      dumb filesystem primitives rooted at the package dir (the TODO.txt "Store package")
sip/            graph gains: File.Source, Entity.Description (interface), Package.Spec
encoders/       unchanged APIs in Phase 1; templates read .Spec values in Phase 2
```

Key invariant that makes this honest: package METS references the checksum/size of PREMIS and rep-METS files, so a fully-inert "dump the graph" is impossible (commons-ip has the same constraint inside `build()`). The design is therefore: **assemble the structure completely, then emit in one canonical order, back-filling metadata `File` nodes (size/checksum/created) as each is rendered.** The ordering exists once, in `write.go`, instead of per profile.

`File.Path` semantics (a real gotcha, preserved from current behavior): `Path` is the href **relative to the METS document that references the file**. Essence files and rep PREMIS are representation-relative (`data/img.jpg`, `metadata/preservation/premis.xml`); schema files, descriptive files, and rep METS entries are package-relative (`schemas/xlink.xsd`, `representations/representation_1/METS.xml`). The writer sets these deterministically; document this on the `Path` field.

---

## Phase 0 — capture a baseline (no code changes)

*Status: **done** (2026-07-15). The baseline lives in `tmp/baseline/` (gitignored): `pkg/` is the reference tree, `failed-checks.txt` the commons-ip FAILED checks at baseline, `README.md` the environment notes. The normalize-and-diff is `scripts/baseline-diff.sh <ref-pkg-dir> <new-pkg-dir>` — after each phase, run it against `tmp/baseline/pkg` plus the failed-checks comparison.*

The originally sketched inline sed one-liner was replaced by the script because implementation surfaced three defects in it: the timestamp regex ate element text through the closing tag (PREMIS timestamps aren't quote-delimited); it under-normalized (METS records the MD5/size of the *raw* generated XML, which contains UUIDs and timestamps, so those attributes churn on every run and the diff could never be clean); and mapping every UUID to one token would hide swapped or dangling cross-references. The script instead maps UUIDs to first-seen sequence numbers, normalizes ISO datetimes with a bounded regex, and blanks SIZE/CHECKSUM only on generated-XML entries (keyed on href — essence and schema fixity stay comparable; MIMETYPE is unusable as the discriminator because of the known METS MIMETYPE defect).

The gate is verified in both directions: two independent `build.sh` runs diff clean against the baseline, and three planted regressions (corrupted essence checksum, swapped structMap cross-reference, deleted file) are each caught.

**Finding for Phase 1:** current output is nondeterministic — schema `<file>` entries in package METS come out in random order per run (`for name := range schemas.Get()` iterates a Go map). The script sorts `<file>` siblings by href to compensate; the canonical emitter (`write.go`) should sort schema names at the source, making that normalization inert.

Acceptance per phase: `baseline-diff.sh` reports structurally identical trees, and `csip validate` in build.sh reports the same (or fewer) FAILED checks than `tmp/baseline/failed-checks.txt` (currently: SIP2 [MUST], CSIPSTR16 [SHOULD]).

## Phase 1 — split assembly from emission

*Status: **done** (2026-07-16). All six steps landed; the gate passed: `baseline-diff.sh` reports the trees structurally identical, `build.sh` reports the same two FAILED checks as baseline (SIP2, CSIPSTR16), and the fail-fast property is demonstrated (bad input → clean error, no partial package dir). Deviations from the letter of this plan, all discussed:*

- *Step 1 adopted the companion plan's `CopyFile (Info, error)` signature up front; the writer back-fills essence fixity from it (same checksum/size as siegfried's, `Created` = copy mtime, matching prior behavior). The **rest** of the [format-identification plan](format-identification-optional.md) was deliberately sequenced **after** this phase rather than with it — one variable at a time through the equivalence gate.*
- *The `Description` interface was deleted, not retained: after `createDescriptiveFile` died it had zero uses (the assembler works with the concrete `*metadata.Description`).*
- *`Profile.BaseDir` was renamed `OutDir` — with the in-place mutation gone it is only ever the operator's destination dir.*
- *`metadata.Decode` now returns an error (the "touch only if trivial while nearby" case).*
- *Schema `File` nodes are assembled in sorted order, fixing the Phase-0 nondeterminism finding at the source; PREMIS/METS nodes are born in the writer (their existence is what Phase 2's `Emit*Premis` flags toggle).*
- *Step 6 grew walk-edge tests beyond the listed assertions: lexical discovery order, non-matching dirs skipped, nested files flattening into `data/`.*

### Step 1: create the `store` package (filesystem primitives, error returns)

New `store/store.go`. This absorbs `createDir`, `copy`, and `createMetadataFile` from profiles/profile.go:255-336, converted from `panic` to returned errors, rooted at the package dir so callers deal only in relative paths (kills the `BaseDir` string-slicing):

```go
package store

type Store struct{ root string } // absolute path of the package dir

func New(root string) *Store { return &Store{root: root} }

func (s *Store) MkdirAll(rel string) error {
	return os.MkdirAll(filepath.Join(s.root, rel), 0775)
}

func (s *Store) CopyFile(src, rel string) error { /* streamed copy, O_TRUNC */ }

// Info carries the fixity facts the writer back-fills onto graph nodes.
type Info struct{ Size, Checksum, Created string }

// WriteMetadata renders fn into a buffer (metadata files are KBs),
// writes it at rel, and returns size/MD5/mtime.
func (s *Store) WriteMetadata(rel string, fn func(io.Writer) error) (Info, error) {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return Info{}, err
	}
	sum := md5.Sum(buf.Bytes())
	dest := filepath.Join(s.root, rel)
	if err := os.WriteFile(dest, buf.Bytes(), 0600); err != nil {
		return Info{}, err
	}
	fi, err := os.Stat(dest)
	if err != nil {
		return Info{}, err
	}
	return Info{
		Size:     strconv.FormatInt(fi.Size(), 10),
		Checksum: hex.EncodeToString(sum[:]),
		Created:  fi.ModTime().Format(time.RFC3339Nano),
	}, nil
}
```

**Deliberate bug fix folded in:** the current primitives open with `O_APPEND` (profile.go:268, 309), so a re-run into an existing file *concatenates* XML. `os.WriteFile`/`O_TRUNC` fixes that. Flag it in the commit message (`Fixed:`).

Add `store/store_test.go` (uses `t.TempDir()`): WriteMetadata returns correct MD5/size; CopyFile truncates on rewrite.

### Step 2: let the graph carry pending content (`sip/` additions)

Three small additions so the graph is complete *before* any write:

```go
// sip/file.go
type File struct {
	...
	Source string // absolute path of the input essence file; empty for generated metadata
	Path   string // href relative to the METS document that references this file
}

// sip/entity.go
// Descriptive is decoded source metadata awaiting serialization by the writer.
type Descriptive interface {
	Encode(w io.Writer) error
}
type Entity struct {
	...
	Description Descriptive
}
```

Give `*metadata.Description` the method (in encoders/metadata):

```go
func (d *Description) Encode(w io.Writer) error { return Encode(w, d) }
```

No import cycle: `sip` imports nothing new; `encoders/metadata` satisfies the interface implicitly. These are the only exported-surface additions in Phase 1.

### Step 3: write the assembler (`profiles/assemble.go`) — pure, zero writes

Everything currently interleaved with IO in `Basic()` + the profile.go helpers becomes one pure function. Format identification moves from the copied file to the **source** file (same bytes → same PRONOM ID and checksum, and we identify before committing anything to disk):

```go
func (p *Profile) assemble() (*sip.Package, error) {
	pkg := sip.NewPackage(p.BaseDir) // Location = dest/uuid-<uuid>; p.BaseDir no longer mutated

	e := sip.NewEntity()

	// descriptive metadata: decode + rewire identifiers; encode happens at write time
	src := filepath.Join(p.InDir, "dc+schema.json")
	f, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("open descriptive source: %w", err)
	}
	defer f.Close()
	d := metadata.Decode(f)
	d.SetObjectIdentifier(e.Identifier)
	e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", d.GetLocalIdentifier("dcterms"))
	e.Description = d

	df := sip.NewFile()
	df.Name = "dc+schema.xml"
	df.Path = "metadata/descriptive/dc+schema.xml" // declared, not derived from disk
	e.AddDescriptionFile(df)

	// schema file nodes: identifier + declared Path now, fixity back-filled at write
	for name := range schemas.Get() {
		sf := sip.NewFile()
		sf.Name, sf.Path = name, "schemas/"+name
		pkg.AddSchemaFiles(...)
	}

	// representations: same regex walk as eachDirectory (profile.go:151),
	// but Process(src) runs on the source path and f.Source records it
	//   f := p.Formats.Process(src)
	//   f.Source = src
	//   f.Path = "data/" + filepath.Base(src)   // rep-relative, per Path semantics
	//   f.SetRepresentation(r); r.AddFile(f)
	// r.SetEntity(e); e.AddRepresentation(r)

	pkg.AddRootEntity(e)
	return pkg, nil
}
```

Guidance: keep the `representation_([0-9]+)$` regex and walk behavior identical (including its TODO comments); fix the ignored `err` parameter in the walk callbacks while moving them (`if err != nil { return err }` — `info` is nil when Walk passes an error). Errors are returned, never panicked; the string-panic for "src is a directory" (profile.go:108) becomes a returned error.

### Step 4: write the emitter (`profiles/write.go`) — the canonical order, once

```go
// write emits pkg to disk in dependency order, back-filling fixity on
// metadata File nodes as each is rendered. This ordering is load-bearing:
// package METS references the checksums of every file written before it.
func (p *Profile) write(st *store.Store, pkg *sip.Package) error {
	// 1. skeleton
	for _, d := range []string{"metadata/descriptive", "metadata/preservation", "representations", "schemas"} {
		if err := st.MkdirAll(d); err != nil { return err }
	}
	// 2. schemas (referenced by package METS fileSec)
	for _, sf := range pkg.SchemaFiles {
		info, err := st.WriteMetadata(sf.Path, func(w io.Writer) error {
			_, err := w.Write(schemas.Get()[sf.Name]); return err
		})
		if err != nil { return err }
		sf.Size, sf.Checksum, sf.Created = info.Size, info.Checksum, info.Created
	}
	// 3. per representation: dirs + essence copies
	//    st.MkdirAll("representations/"+r.Label+"/data") etc.
	//    st.CopyFile(f.Source, "representations/"+r.Label+"/data/"+f.Name)
	// 4. descriptive XML (referenced by package METS dmdSec)
	//    info, err := st.WriteMetadata("metadata/descriptive/"+df.Name, pkg.Root.Description.Encode)
	// 5. per representation: PREMIS then METS (rep METS reads r.PremisFile fixity)
	//    premis → premis.EncodeRepresentation; back-fill; r.AddPremisFile
	//    mets   → mets.EncodeRepresentation;   back-fill; r.AddMetsFile
	//    NOTE: rep PREMIS Path is rep-relative ("metadata/preservation/premis.xml"),
	//          rep METS File node Path is package-relative ("representations/<label>/METS.xml")
	// 6. package PREMIS (package METS reads pkg.PremisFile fixity)
	// 7. package METS — strictly last
	return nil
}
```

Steps 5–7 reuse the existing `premis.Encode*`/`mets.Encode*` functions unchanged — they already take `(io.Writer, model)` and are exactly what `WriteMetadata` needs.

### Step 5: rewire `Basic()`, delete `Roda()`, propagate errors

`profiles/basic.go` shrinks to:

```go
func (p *Profile) Basic() (*sip.Package, error) {
	pkg, err := p.assemble()
	if err != nil { return nil, err }
	st := store.New(pkg.Location)
	if err := p.write(st, pkg); err != nil { return nil, err }
	return pkg, nil
}
```

Keep the current `slog` progress lines (moved into assemble/write at the equivalent moments). Delete `profiles/roda.go`: it is unreachable from the CLI (`create_cmd.go` switch has no `"roda"` case), known-broken, and the agreed direction is a genuine E-ARK writer in Phase 3, not a migrated copy. Delete the now-dead helpers in profile.go (`createPackage`, `createDescriptiveFile`, `eachEssenceFile`, `generate*`, `createMetadataFile`, `copy`, `createDir`, …) — profile.go should retain only `Config`, `Profile`, `New`, and the `Description` interface (which moves next to its use, or is replaced by `sip.Descriptive`).

`cli/create_cmd.go` handles the new signature:

```go
case "basic":
	pkg, err = profile.Basic()
	if err != nil { return err }
```

### Step 6: tests + acceptance for Phase 1

- `profiles/assemble_test.go`: fake `formats.Identificator` (returns a canned `*sip.File`), a fixture input dir under `t.TempDir()` with a minimal `dc+schema.json` and `representation_1/`; assert graph shape — root entity wired, `MEEMOO-LOCAL-ID` extracted, files carry `Source` and rep-relative `Path`, **and no files were created on disk**.
- Acceptance: `go build`, then `./build.sh`; run the Phase-0 normalized diff. This is the equivalence gate before Phase 2.

Commits (small, prefixed): `Added: store package…`, `Changed: split profile into assemble and write phases`, `Fixed: metadata files opened with O_APPEND…`, `Removed: unreachable Roda profile…`.

## Phase 2 — declarative ProfileSpec + registry

*Status: **done** (2026-07-16). Steps 7–9 landed; the gate passed after each step (structurally identical trees, same two FAILED checks). Deviations from the letter of this plan, all discussed:*

- *Naming: the profile-as-data type is `profiles.Definition`, not `Spec`/`ProfileSpec` — it would have collided with `sip.Spec` one import away, and `ProfileSpec`/`profiles.Profile` stutter. The engine type was renamed `Profile` → `Builder` in the same move (it never was a profile), with files `builder.go` and `definition.go` named accordingly. A profiles-vs-builder package split was considered and rejected: no consumer wants one without the other, and Phase 3's writer-selector field would create an import cycle across that boundary.*
- *The registry is a plain in-package map — no `Register`/mutex; the formats/ registry needs self-registration from sub-packages, definitions don't.*
- *Step 7's agent range emits uniform indentation where the old hardcoded block mixed tabs and spaces; `baseline-diff.sh` now normalizes leading whitespace and blank lines (verified still to catch planted regressions).*
- *Both METS templates got the premis-less guards (amdSec + conditional structMap ADMID), not just the package template — unexercisable while `basic` emits all PREMIS; they exist for future definitions.*
- *Removing the inert `idStore` spawned a decision recorded in [TODO.md](../TODO.md): duplicate-identifier protection becomes a future `sip.Package.Validate()` set-check on the assembled graph (catches systematic duplication, not just the ~10⁻²⁵ UUIDv4 collision), rather than any minting-time machinery.*

### Step 7: `sip.Spec` on the package; templates read data instead of literals

```go
// sip/spec.go — the profile-level values a METS document declares
type Spec struct {
	ProfileURL                  string // METS PROFILE attr
	Type                        string // METS TYPE (TODO.txt: should come from a vocabulary — now fixable in one place)
	ContentInformationType      string // csip:CONTENTINFORMATIONTYPE
	OtherContentInformationType string // csip:OTHERCONTENTINFORMATIONTYPE
	Agents                      []Agent
}
type Agent struct{ Role, OtherRole, Type, OtherType, Name, Note string }
```

Add `Spec *Spec` to `sip.Package`; the assembler sets it from the profile spec. `mets.EncodePackage` keeps its signature (the template already receives the package — read `{{ .Spec.Type }}`, `{{ .Spec.ProfileURL }}`, `{{ .Spec.ContentInformationType }}`, `{{ .Spec.OtherContentInformationType }}`, and range `.Spec.Agents` for the metsHdr, replacing the hardcoded attrs at encoder.go:56-60 and 103-118). The representation template needs the spec too; use a view struct:

```go
type repView struct {
	*sip.Representation
	Spec *sip.Spec
}
func EncodeRepresentation(w io.Writer, r *sip.Representation, spec *sip.Spec) error {
	return dc.ExecuteTemplate(w, "representation", repView{r, spec})
}
```

While in this file, remove the dead `idStore` global (encoder.go:16-27): `identifier()` never appends to it, so the `lo.Contains` check is inert; UUIDv4 collision within one document is not a real risk. Deleting it likely removes the `samber/lo` dependency entirely (verify with grep) — smaller footprint, and it closes the TODO.txt "Fix identifiers in METS file" confusion. `Fixed:` commit.

### Step 8: the Spec type + registry (`profiles/spec.go`)

Mirror the `formats/` registry shape (formats/formats.go:17-35):

```go
type Spec struct {
	Name                     string
	DescriptiveSource        string // "dc+schema.json"
	LocalIdentifierScheme    string // "dcterms" → MEEMOO-LOCAL-ID; "" disables
	EmitPackagePremis        bool
	EmitRepresentationPremis bool
	Mets                     sip.Spec
}

var registry = map[string]Spec{
	"basic": {
		Name:                     "basic",
		DescriptiveSource:        "dc+schema.json",
		LocalIdentifierScheme:    "dcterms",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		Mets: sip.Spec{
			ProfileURL:                  "https://earkcsip.dilcis.eu/profile/E-ARK-CSIP.xml",
			Type:                        "Photographs – Digital", // known-wrong value, now data: fix per CSIP vocab without touching templates
			ContentInformationType:      "OTHER",
			OtherContentInformationType: "https://data.hetarchief.be/id/sip/2.0/basic",
			Agents: []sip.Agent{ /* SIP creator + Universiteitsbibliotheek Gent, as today */ },
		},
	},
}

func Get(name string) (Spec, bool)
func Names() []string // sorted, for the CLI error message
```

`assemble(spec)` and `write(st, pkg, spec)` become spec-driven: descriptive source filename, the local-identifier extraction guard, and `if spec.EmitRepresentationPremis { … }` / `if spec.EmitPackagePremis { … }` around emitter steps 5–6. Guard the package-METS template too: `{{ .PremisFile }}` derefs nil-panic if package PREMIS is off — wrap the amdSec block in `{{ with .PremisFile }}…{{ end }}` so a premis-less profile renders valid METS.

### Step 9: replace the CLI switch with a lookup

```go
spec, ok := profiles.Get(flagProfile)
if !ok {
	return fmt.Errorf("unknown profile %q (available: %s)", flagProfile, strings.Join(profiles.Names(), ", "))
}
pkg, err := profile.Build(spec)
```

`Build(spec Spec)` is the single engine (assemble+write); `Basic()` disappears. A future meemoo content profile (material-artwork, newspaper, …) is one registry entry, not an 80-line method. Re-run the Phase-0 diff + `build.sh` — output must still be structurally identical.

## Phase 3 — E-ARK/RODA writer seam (scoped outline, follow-up work)

*Status: expanded into the [eark-writer plan](eark-writer.md) (2026-07-16, reworked 2026-07-17), which supersedes this outline — including a correction: there is no second writer and no eark template family. Design review showed the family-varying surface is data plus one encoding choice, so profile families share the one canonical writer and select encodings per [ADR-0007](../decisions/0007-profile-families-share-one-writer.md); sibling writers would have reintroduced the `Basic()`/`Roda()` duplication at writer level.*

Goal: `--profile eark` producing a plain, valid E-ARK SIP that RODA ingests. Not implemented in this plan; phases 1–2 create the seam it needs:

- The `sip/` graph is spec-neutral and shared. Variation *within* the meemoo family is registry data (Phase 2). Variation *across* families (meemoo vs E-ARK) is a **writer + template set**: an `encoders/mets` E-ARK template family (`define "eark-package"` etc., or a sibling encoder package) with `PROFILE="https://earkcsip.dilcis.eu/profile/E-ARK-SIP.xml"`, a vocabulary `CONTENTINFORMATIONTYPE` (no meemoo URL), E-ARK directory conventions, and plain-DC descriptive output (the unused `"dc"` template in encoders/metadata/encoder.go:120 finally gets a caller).
- Mechanically: `Spec` gains a writer selector (e.g. `Family string` or a `write func` field); `write.go` stays the meemoo emitter, `write_eark.go` is added when this phase starts.
- Acceptance for that phase: commons-ip validation in E-ARK SIP mode **plus an actual ingest test against a RODA instance** — the spec-on-paper and what RODA accepts must both be checked.

## Deferred input-side changes (separate follow-up plan — design for, don't implement)

Two operator-experience changes are agreed in principle but deliberately **excluded** from this refactor, because the refactor's acceptance gate is behavior preservation (Phase-0 diff): changing the input contract mid-refactor would invalidate the equivalence check. They become small assembler-local changes once Phases 1–2 land:

1. **CSV descriptive metadata input** (alongside JSON-LD, not replacing it): a decoder registry keyed on file extension — same pattern as `formats/` — producing a `sip.Descriptive`. The XML output side is untouched. Open design point for that plan: CSV convention for multi-valued fields (repeated keys in a key/value layout).
2. **Free-form representation folder names**: replace the `representation_([0-9]+)$` regex rule with "every top-level source directory is a representation, named by its directory name". **Prerequisite check**: whether the meemoo SIP 2.0 spec mandates `representation_N` naming inside the package — if yes, normalize output dirs to `representation_1..N` and keep the operator's name as the Label; if no, sanitize names for filesystem/URL safety. README update in the same change.

**Seams this plan must keep clean for them** (guidance for the implementer):
- Descriptive decoding stays behind exactly one function call in `assemble()` (input path in → `sip.Descriptive` out), so adding a decoder never touches the walk or the writer.
- Representation discovery (the regex walk) stays an isolated, separately-testable function in `assemble.go`, with the matching rule a candidate for a future `Spec` field — do not inline it into the essence-file loop.

## Docs & housekeeping (same change as the code, per CLAUDE.md)

- Update the "System shape" section of `CLAUDE.md`: assemble/write phases, `store/` package, spec registry replacing the switch, `Roda()` removal, error-return signatures. (The design doc's *Build lifecycle* and *Code organization* sections change in the same way.)
- `TODO.txt`: remove resolved items — "Split out file system handling into a Store package", "Figure out how to handle paths & base path of the target package", "Fix identifiers in METS file", the RODA rep-PREMIS defect note, and "Figure out how to pass on additional metadata for Mets file(s)" (agents/TYPE now data). Leave the TYPE-vocabulary question but note it's now a one-line registry fix.
- `README.md`: `--profile` now lists available profiles from the registry; note the error-message change. No env vars change, so `CONFIG.md` is untouched.
- Out of scope, deliberately: `archive.Zip` panic style, the `"representation_0"` regex TODO, `metadata.Decode`'s panic (touch only if trivial while nearby), and everything in Phase 3.

## Verification

1. **Phase 0 baseline diff** after Phase 1 and again after Phase 2: normalized XML trees structurally identical, same file inventory.
2. `./build.sh` green (requires `docker`, `jq`, working `sf` per `.env` — see [ADR-0005](../decisions/0005-dockerized-validation-and-html-reporting.md)): commons-ip validation reports no new FAILED checks — this is the project's acceptance check.
3. `go test ./...`: new `store` and `profiles` (assembler) tests pass; `go vet ./...` clean.
4. Negative paths by hand: missing `dc+schema.json` → clean error, not a panic, and **no partial package dir left in dest** (assembly fails before any write — this is the fail-fast payoff, worth demonstrating).
5. `./sip-creator create --profile nope ./tmp/basic basic-uuid` → lists available profiles.
