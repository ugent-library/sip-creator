# TODO

The live backlog. Items that a plan or ADR now owns point there rather than
sitting as loose unknowns; see [plans/](plans/) and [decisions/](decisions/).
The system as it exists today is described in [sip-creator-design.md](sip-creator-design.md);
its rough edges are collected under [Known gaps](sip-creator-design.md#known-gaps).

## Open design questions (not yet owned)

These need a decision before they become plan work.

- **Administrative values ADR-0010 assigns to configuration are not implemented.** [ADR-0010](decisions/0010-config-over-self-describing-input.md) lists the archival creator, contact persons and a submission agreement reference as configuration, but the tool reads only the submitting organization and the content category, and the METS carries no `ARCHIVIST` agent, contact agents or `altRecordID TYPE="SUBMISSIONAGREEMENT"`. Decide from the E-ARK SIP and Meemoo 1.2 specifications which of them each profile needs before adding environment variables.

- **Format-identification provenance as a PREMIS event.** We record *what* was identified but not *who/when/how*. Proper preservation practice is a PREMIS event ("format identification", agent: siegfried + version, signature file + date). That is what a future archivist actually needs to trust the format claim. Since ADR-0009 the raw material is finally in hand: the sidecar report's header carries exactly this (siegfried version, signature file, scandate), currently ignored by `DecodeSiegfried`. Blocked on `sip.Event` (an empty stub) growing up; when events are modeled, this should be the first one emitted. Raises the feature from "enriches a SHOULD field" to production-grade provenance. No event design exists yet anywhere; picking this up needs an ADR covering the event shape (LoC eventType vocabulary, dateTime, detail, linking identifiers), where events attach on the graph (per-file vs. package-level), PREMIS agents vs. the METS-specific `sip.Agent`, and pass-through of *received* vendor events ([input-spec.md](input-spec.md)) alongside self-generated ones.
  - **The producer's file dates in the metadata.** Since 2026-10-06 every copied file keeps its source's modification time, in the package directory and in the zip entries; METS `file/@CREATED` stays the time of the copy, as CSIP defines it (commons-ip 2 writes the build time too). A file system attribute is lost again on the next careless copy, so the dates belong in the metadata as well, labeled for what they are. Two sources, both for the events design: (1) the modification time Siegfried records per file in its report (`modified`, next to the MD5 that binds it to the bytes), which `DecodeSiegfried` ignores; (2) the creation dates embedded in the files themselves (EXIF `DateTimeOriginal`, a PDF's creation date, a broadcast WAV's origination date), which come from a characterization report such as ExifTool's or FITS's, taken as a sidecar like Siegfried's (ADR-0009). A modification time is not a creation date: PREMIS `dateCreatedByApplication` fits the embedded dates, not the file system's. Neither CSIP, Meemoo 1.2 nor BagIt (RFC 8493, which records only paths and checksums) asks for any of this; it is preservation practice. A system that copies files from storage to scratch disk should set their modification times from the storage metadata first, or the date kept is the download time.

- **Meemoo SIP 2.x migration, when it stabilizes.** The basic profile targets [1.2](https://developer.meemoo.be/docs/diginstroom/sip/1.2/), the stable spec ([meemoo-12 plan](archive/meemoo-12.md)); 2.0/2.1 are release candidates Meemoo's production ingest does not accept. The former 2.0 target's values live in git history (pre-2026-07-17 `profiles/definition.go`). When 2.x goes stable: a change to vocabularies, encodings, values, and the 2.x removal of the bag layer; likely a versioned Meemoo descriptive document alongside 1.2's. A profile's package exports its definition with its descriptive encoder (`profiles/meemoo/profile.go`; [ADR-0015](decisions/0015-descriptive-worlds-dc-and-mods.md) put the standard on the profile, retiring the family constant of [ADR-0007](decisions/0007-profile-families-share-one-writer.md), and [ADR-0018](decisions/0018-engine-and-profile-packages.md) gave each profile its own package), so a new Meemoo document is an encoder in that package and one registry entry per profile. Watch for Meemoo publishing 2.x validation assets at the same moment (the parked meemoo-validation-harness idea). Needs its own spec-delta plan.

- **EDTF dates under basic are checked by nothing.** The basic profile writes `dcterms:created` and `dcterms:issued` with `xsi:type="edtf:EDTF-level1"`, but Meemoo's `edtf.xsd` types the value as plain text, and the library does not check the syntax either: a `created` row of `15 januari 2026` builds a package that claims an EDTF date it does not hold, and `build.sh` reports it valid (found 2026-10-06). The input specification describes the value as "a year or an ISO date". Before deciding whether the library checks the syntax, find out what Meemoo's ingest does with such a value: whether it rejects the package or stores the value as given. If it rejects the package, check EDTF level 1 in `Terms.Validate`, so the CLI reports it with its line. If it accepts it, the check is a question of data quality.

- **The eark profiles' descriptive format: RODA's simpledc or OAI-PMH's oai_dc.** The eark profile writes `simpledc` with unqualified elements, RODA's variant of DCMI's container schema (ADR-0016, ADR-0021), and ships `simpledc.xsd` built from DCMI's files. DCMI never published a standalone record format; the one in wide use is OAI-PMH's `oai_dc` (`oai_dc:dc` root, `dc:` elements, a schema at openarchives.org). For the reference role ADR-0022 gives the profiles, `oai_dc` would serve institutions that do not run RODA. Open: whether RODA ingests, indexes and displays `oai_dc` without configuration, and whether a second Dublin Core profile or a switch is the answer. Needs its own decision.

- **Identifier minting authority.** SIP Creator mints only package-local `uuid-<uuid>` IDs and asserts no authority over them ([ADR-0001](decisions/0001-package-builder-not-archive.md)). Still open:
  - Who mints the *package* identifier, and the identifiers of intellectual entities & representations: the producer, or a downstream identifier service? (Ref: [CSIP1](https://earkcsip.dilcis.eu/).)
  - Is a UUID meaningful only within the SIP acceptable as the common key tying description / IE / representation / file together? Where would externally-minted IDs be recorded if not?

- **Multiple descriptions / entities / formats.** How should the tool handle multiple descriptive records, sub-intellectual-entities, or multiple formats per representation? The model has a per-file `Format` slot; the sub-entity slot was removed on 2026-09-29 ([ADR-0019](decisions/0019-build-is-the-library-face.md)) until a design needs it, and the `basic` profile builds a single root entity.
  - Is the intended shape to walk an LD graph and populate the package from it (cf. [sipin-mh-sip-creator](https://github.com/viaacode/sipin-mh-sip-creator/tree/main/tests/resources))? Implications: identifiers would be minted by external services; there is no strong Go triplestore library, so this likely needs a supporting query API or a tech change. (Format characterisation is meanwhile decided: optional pre-computed sidecar input, [ADR-0009](decisions/0009-characterization-as-sidecar-input.md), superseding ADR-0006's mechanism; fixity stays native and in-process.)
  - For a bibliographic profile: source for mapping to MODS: BIBFRAME or otherwise? (Ref: [MODS–BIBFRAME mapping](https://www.loc.gov/standards/mods/modsrdf/mods-bibframe-mapping.html).)

- **Library builder API: accept streams and pre-computed fixity.** Most of the embeddability list shipped with the [input-convention plan](archive/input-convention.md) (2026-08-20): the builder takes its source package as data (`build.SourcePackage`: descriptive terms, documentation, received PREMIS, characterization report), takes administrative metadata as data (`sip.MetsDeclaration`/`sip.Agent`), accepts a caller-supplied package identifier (updates reuse the original `mets/@OBJID`), keeps representation labels free-form, validates caller-supplied input data, and builds to a caller-controlled destination. What remains: accept essence as **streams** (`io.Reader`/`fs.FS` + logical path), not only filesystem paths, and accept **pre-computed fixity**, computing checksums only when none are supplied. See the [CLI/library boundary](sip-creator-design.md#clilibrary-boundary) in the design doc.
  - **Measured 2026-10-06: MD5, not the disk, sets the build time.** One package of eight 4 GB files of random bytes (32 GB, more than the 18 GB of memory, so reads come from disk), profile `eark`, on an Apple-silicon MacBook's internal SSD. MD5 runs at about 0.6 GB/s on one core; CRC32 at about 30 GB/s.

    | run | per essence file | time |
    |---|---|---|
    | `sf -hash md5 -json .` (the operator's pass) | read, MD5, identify | 101 s |
    | no report, `--no-zip` | read, MD5, write | 53 s |
    | report, `--no-zip` | the above, plus a read and MD5 to check the report | 99.5 s |
    | report, zip | the above, plus two reads and a write for the zip (CRC only) | 124 s |

    The report check costs 46 s (a full MD5 pass); the two zip passes together cost 25 s, because CRC is cheap. Each MD5 pass over the essence is about 50 s per 32 GB, and with a report the essence is hashed three times: by Siegfried, by the report check, and by the copy.
  - **Done 2026-10-06: the report's MD5 is taken as given** ([ADR-0032](decisions/0032-the-report-checksum-is-taken-as-given.md)), as commons-ip 2 takes a checksum the caller supplies. With a report, the build no longer reads the essence to check it, and the copy computes no MD5. Same input, re-measured:

    | run | before | after |
    |---|---|---|
    | no report, `--no-zip` | 53 s | 58 s (unchanged: the copy still hashes) |
    | report, `--no-zip` | 99.5 s | 17 s (a plain copy, about 1.9 GB/s) |
    | report, zip | 124 s | 45 s |

    The essence is now hashed once, by Siegfried. What remains for pre-computed fixity is a source other than the report: a checksum field on `SourceFile` for a program that holds checksums in its storage, under the same rule. Streaming into the zip (the remaining two reads and one write, about 28 s here) stays out of scope: it changes how the writer works and basic's deliverable is the directory, not the zip.
  - **It must also ship `sip.Package.Validate()`**, the first domain-validation method (`Validate` methods on `sip/` types per `CLAUDE.md`), called between assemble and write, covering the graph-level checks: identifier uniqueness across the graph (a set-check; decided 2026-07 when the inert METS `idStore` was removed: no minting-time machinery, catch *systematic* duplication like a node wired in twice, the ~10⁻²⁵ UUIDv4 collision comes free, commons-ip's `xs:ID` XSD check stays the document-level net) and the no-empty-`Mime` invariant ([ADR-0009](decisions/0009-characterization-as-sidecar-input.md)). The *input-data* validation slice already shipped with the input-convention plan ("validation splits in two"); these graph checks become real error classes when callers construct graphs and supply identifiers wholesale.

- **Windows: the final rename can fail while a virus scanner holds a file.** `Build` writes the package into `dest/.<identifier>.tmp` and renames it, and the zip is renamed from `.<identifier>.zip.tmp` ([ADR-0031](decisions/0031-final-names-hold-complete-output.md)). On Windows, Defender or the search indexer opens a file just after it is written, and a rename or removal in that moment fails with "Access is denied". The build then reports an error and leaves nothing under a final name; a rerun works. Not seen yet: the tool has not been run on Windows. When it is, and the error shows up, the fix is a short retry of `os.Rename` and `os.RemoveAll` on access denied (error 5) and sharing violation (error 32), as the Go toolchain does in its internal `cmd/internal/robustio` ([golang.org/issue/31247](https://golang.org/issue/31247)). A version was written and backed out on 2026-10-06 as premature.

- **Parked 2026-10-06: the input reader as library, CI, a changelog and a first tag.** Cut from step 5 of the assessment plan, which kept only the outside profile's own schemas. `cli/input` is importable by any program and imports no profile; moving it to a top-level `input/` would be a rename without a caller, so it waits for one. A GitHub Actions workflow running `go vet ./...` and `go test ./...` needs no Docker, `sf` or `.env`, and is worth adding on its own whenever wanted. A `CHANGELOG.md` and a first tag (`v0.1.0`, which the software agent then states) come when UGent's own use calls for them, such as a stable version for a program that embeds the library to pin ([ADR-0033](decisions/0033-ugent-first-profiles-of-your-own.md)); until then the README's experimental warning stands.

- **Exit statuses for create: decided to wait for a caller.** `check` exits with 1 when the input folder has problems and with 2 when it could not check the folder (`exitProblemsFound`, `exitNotChecked` in `cli/cli.go`), for a script that checks a batch of folders. `create` exits with 1 on every failure. Decided on 2026-10-06 not to tell create's failures apart until a caller needs it: operators read the message, and programs embed the library instead of running the CLI. When one does, the sketch is: 1 for input problems, 2 when the command could not start, 3 when building or zipping failed; and, for the library, one exported sentinel `build.ErrInvalidSource` that every error `Build` returns because of the source package matches with `errors.Is`, without changing the messages (profile rules, `SourcePackage.Validate`, characterization coverage and checksums, received PREMIS roots). Only the sentinel is needed if the caller is a program; only the statuses if it is a script.

## Validator status

**Resolved 2026-07-17** ([meemoo-12 plan](archive/meemoo-12.md)): both sample
packages report **VALID**: `basic` against E-ARK 2.0.4 (the era Meemoo 1.2
builds on; its SIP2 check expects exactly the unversioned profile URL Meemoo
requires), `eark` against 2.2.0. The old SIP2 failure was a spec-version
mismatch, not a package defect; the old XSD-failure evidence stopped
reproducing under commons-ip 2.11.2.

**Resolved 2026-08-20** ([input-convention plan](archive/input-convention.md)
I6b): the last SHOULD-level warning, **CSIPSTR16**, cleared once `tmp/basic`
gained `documentation/` at package and representation level. Both profiles
now validate **VALID with zero warnings**.

**2026-10-01** ([descriptive-model plan](archive/descriptive-model.md) S8):
the `eark-mods` profile joins them, VALID with zero warnings against
2.2.0, and `build.sh` validates every `mods.xml` in the package against the
MODS 3.7 schema with xmllint. All three profiles validate.
