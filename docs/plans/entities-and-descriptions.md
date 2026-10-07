# Plan: sub-entities and several descriptions per entity (future work)

*Status: **parked** (drafted 2026-10-07). No UGent case needs either today. The plan
records the design and the facts checked, so the questions are not researched twice. It is
written in the terms of the [ugent-profiles plan](ugent-profiles.md): a profile allows one
or several models, and sub-entities are a profile rule. It resumes when a profile page
states a rule that needs one of the two. Update this line when that happens.*

## Context

### Today

A package has one intellectual entity. The entity has one description, of the one
metadata model its profile names, written as one document under one file name
(`Definition.Model`, `Definition.DocumentName`). A representation may carry one
description of its own where the profile allows it. The package METS has one `dmdSec` per
entity and the Metadata division's `DMDID` names one identifier. `sip.Entity` has no
children, since 2026-09-29 ([ADR-0019](../decisions/0019-build-is-the-library-face.md)).

Two routes to a richer package are closed (TODO, 2026-10-07): the tool will not walk an LD
graph to populate a package, and no profile maps its description from BIBFRAME. A
program that holds its data as a graph maps it to a `build.SourcePackage` itself.

### What the specifications and the readers allow

- **CSIP.** `mets/dmdSec` is 0..n (CSIP17). CSIP92: every `dmdSec` identifier is listed in
  the Metadata division's one `DMDID` attribute, space-delimited. The structMap labelled
  `CSIP` has a fixed shape (Metadata, Documentation, Schemas, one division per
  representation); METS allows further structMaps.
- **Meemoo SIP 1.2.** Basic: exactly one `dc+schema.xml`, no descriptive metadata at the
  representation level. Bibliographic: one `mods.xml` at package level, representation
  descriptions permitted. No sub-entities.
- **commons-ip 2.11.2**, checked in its source on 2026-10-07. The validator loops over
  every `dmdSec` (CSIP17 resolves each href and checks every file under
  `metadata/descriptive` is referenced; CSIP18 requires unique IDs; CSIP92 checks each
  `DMDID` reference against the `dmdSec` list). The parser ignores `DMDID` and reads every
  `dmdSec` into one descriptive record each, at package or representation level.
- **RODA.** Its ingest creates one descriptive metadata record per parsed `dmdSec`, keyed
  by the file name: two records need distinct file names, or the second is an "already
  exists" error outside an update job. A record is rendered and indexed when its type is
  configured; the default configuration lists `ead_2002`, `dc_SimpleDC20021212` and
  `key-value`, so a MODS record is stored as it is unless a crosswalk is added. One SIP
  becomes one AIP; hierarchy is parent AIPs, assigned by the ingest job or named inside
  the package by a RODA-specific structMap (`LABEL="RODA"`, an `Ancestors` division with
  one `mptr` per parent).

### The cases that would make this real

- **A.** A Simple Dublin Core record next to a MODS record for the same entity, because
  RODA renders Dublin Core by default and the catalogue record wants MODS.
- **B.** A composite object whose parts need their own description inside one package: a
  bound volume holding several works, a periodical delivered with its issues, where a
  parent AIP per part is not wanted.
- **C.** A Meemoo profile that requires more than one descriptive document. None in 1.2.

## Design

### Several descriptions per entity

- **Definition.** `Models []MetadataModel` replaces `Model`. The document's file name
  moves onto the model (`dc.xml`, `mods.xml`, `dc+schema.xml`), because one
  `DocumentName` cannot serve several models. A description's model is the one whose
  `ValidateType` accepts it. At most one description per model per entity: RODA keys
  records by file name, and two records of one standard would need a naming rule that no
  case asks for.
- **Source.** `SourcePackage.Descriptions []sip.Description`, and the same on a
  representation. `Validate` validates each; the profile says which models are required
  and which allowed.
- **Graph.** `Entity.Descriptions []Description` and `DescriptionFiles []*File`;
  `Package.DescriptiveFiles` returns all of them.
- **Writer.** One document per description, in the models' order.
- **METS.** One `dmdSec` per file; `DMDID` lists every identifier, which METS allows (an
  IDREFS attribute) and CSIP92 describes. The representation METS likewise.
- **Input.** A supplied document per model, told apart by file name (`dc.xml` and
  `mods.xml` side by side). Rows (`description.csv`) map to one model, the profile's
  first; rows for a second model need a second file name, decided when the case exists.
- **No descriptions.** An empty `Models` list means the profile carries no descriptive
  metadata: no descriptive file node, no descriptive XSD, a package METS without `dmdSec`
  and without `DMDID` on its Metadata division, which stays (CSIP88), and an input reader
  that reports a `description.csv` as a violation. That is what ADR-0029 decided for a nil
  model; the retired profile-rules-and-names plan ([archive](../archive/profile-rules-and-names.md))
  spelled it out in its steps 2 to 4, which are this list's empty case and land with step
  2 below. No in-tree profile uses it.
- **A cheaper answer to case A.** A model that derives a Simple Dublin Core record from
  the MODS record (title, identifier) at build time, so the package carries both without
  the operator supplying two. Worth weighing against a crosswalk in RODA before the engine
  changes.

### Sub-entities

- **Graph.** `Entity.Children []*Entity`; each child has its own identifier and
  descriptions. Its content is the difficulty: CSIP ties files to the package's
  representations, not to entities.
- **Three ways to say which files belong to a child**, to be decided when case B exists:
  1. **A logical structMap.** A second `structMap` (`TYPE="LOGICAL"`, a label of the
     tool's) with nested divisions per entity, each with its `DMDID` and `fptr` elements
     to the files that belong to it. The CSIP structMap is untouched. Conformant METS; RODA
     and Meemoo ignore it, so it carries meaning for a reader of the package only, which
     is still what a package is for. Check first whether commons-ip's structure checks
     object to a second structMap.
  2. **A representation per part.** Rejected on sight: a CSIP representation is a version
     of the content, and using it for a part misuses the concept.
  3. **Hierarchy by reference.** One package per entity, and a parent reference emitted
     as RODA's ancestors structMap, or left to the ingest job. What RODA supports today,
     and what systems that automate UGent's ingest do by dropfolder. Needs no children in
     the graph: a `ParentIdentifiers` value on the source package and a profile that
     allows it.
- **Descriptions of children.** The profile's models; the identity rule applies per child.
- **PREMIS.** Under a profile that emits PREMIS, a child is a PREMIS intellectual entity
  with a structural relationship to its parent.
- **Profile rules.** `AllowSubEntities bool`, or a maximum depth; the models children may
  carry; whether a parent reference is required (a periodical issue must name its
  periodical).
- **Input.** An `entities/<name>/` folder per child, or a column in a manifest; decided
  with the case.

### Left out on purpose

- Page order and file structure inside a representation: a profile rule of its own, which
  needs an ordered structMap in the graph and the METS encoder; the ugent-profiles plan
  keeps it out until a profile page says MUST.
- Several formats per representation: answered by the per-file `Format`.

## Steps (coarse; refined when the plan resumes)

1. ADR: descriptions as a list; models on the definition as a list; the document name on
   the model.
2. Engine: the lists in source, graph, writer and template, the empty list included;
   tests; the three profiles' output byte-identical, since none uses more than one
   description.
3. Input: supplied documents told apart by file name; rows to the first model.
4. Case A: the profile rule and its page, or the derived Dublin Core record instead.
5. Case B: an ADR on the form of the hierarchy in METS (logical structMap or parent
   reference); graph, template, input, PREMIS.
6. Case B by reference: the parent reference on the source package and its structMap.

## Verification

- `scripts/reference-diff.sh` clean after steps 2 and 3.
- The commons-ip validation in build.sh reports VALID for a test package with two
  `dmdSec` elements under 2.2.0 (a fixture, not an example).
- The first ingest of such a package into RODA shows both records on the AIP.

## Open questions

- Does commons-ip's structure validation accept a second structMap? Checked when step 5
  starts.
- Does Meemoo 2.x define sub-entities or several descriptive documents? Checked when the
  2.x migration plan is written.
- The name of a second rows file for a second model.
