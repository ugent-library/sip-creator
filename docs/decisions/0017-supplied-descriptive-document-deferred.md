# 0017 — Descriptive metadata arrives as terms only; the supplied-document route is deferred

Status: **Superseded by
[ADR-0021](0021-descriptive-model-follows-its-standard.md)** (2026-09-30):
the route returns for the two eark profiles as a second description type
behind `sip.Description`, not as the second input path whose cost this
record describes. Was Accepted (2026-09-28), superseding in part
[ADR-0016](0016-descriptive-input-rows-or-supplied-document.md): its
supplied-document decisions were deferred; its rule that the profile fixes
the descriptive standard and its file naming stood.

## Context

ADR-0016 gave descriptive metadata a second way into the package. Next to
the terms the tool encodes, a producer could hand it a finished document
of the profile's standard (`dc.xml`, later `mods.xml`), copied into the
package like essence after a check that it is well-formed XML with the
standard's root element. The route was meant for records the flat rows
cannot say: a catalogue export, or a MODS record with structure
([ADR-0015](0015-descriptive-worlds-dc-and-mods.md)).

The library half was implemented on the branch `eark-mods-support` in the
working tree and never committed. A review of the S2 refactor on
2026-09-28 found the descriptive code more entangled than the refactor's
goals justified, and the document route was the largest single source of
that. The check of the root element was 26 lines. The cost was the second
path the route opened at every level:

- a `DescriptiveDocument` path next to `Descriptive` on `profiles.Input`
  and on each `SourceRepresentation`, with a rule at both levels in
  `Input.Validate` that a level has terms or a document, not both;
- a document check on the profile's descriptive standard, doubling as the
  flag that says whether the standard takes documents at all;
- a branch and a node builder in the assembler at package level, and the
  branch again in the representation loop;
- a branch in the writer that copies a supplied document or generates one
  from terms;
- a new package, `encoders/xmldoc`, lifted out of the received-PREMIS
  check so the document check could share its XML reading;
- one more case in which `Entity.Description` and
  `Representation.Description` are nil, which every reader of those
  fields had to know about.

No producer needs the route today. The systems that automate ingest hold
descriptive metadata as flat columns and construct terms; the CLI reads
rows. The route served a case that has not arrived.

## Decision

**Descriptive metadata reaches the tool as terms only.** `Input.Descriptive`
and `SourceRepresentation.Descriptive` are the only descriptive inputs. A
profile's descriptive standard is a type check plus an encoder and knows
nothing about documents. The writer generates every descriptive document.
Received PREMIS pass-through is unaffected: it predates the route and keeps
its own well-formedness and root element check inline.

**The working-tree implementation is discarded, not parked on a branch.**
The same review starts a simplification of the descriptive code (who owns
terms validation, what the `Description` interface is for), and code kept
aside against a moving model would be stale by the time the route is
wanted. ADR-0016 records the design and this ADR records the shape of the
discarded change; a return starts from those, on whatever the model has
become by then.

**The route is deferred, not rejected.** The reasons in ADR-0016 hold: a
record the rows cannot express needs a way in eventually. The route returns
when a producer has such a record in hand, as a plan of its own, after the
descriptive code has one owner for validation.

## Alternatives rejected

- **Keep the route and drop only the check of the root element.** The
  check was the cheapest part; every branch listed above would stay.
- **Keep the route at package level only.** Halves the representation
  branches and keeps everything else, for a feature no producer calls.
- **Keep the code on a side branch.** The descriptive model is about to
  change under it; rebasing it later would cost more than rewriting it
  from the recorded design.

## Consequences

- The limits of the flat MODS model ([ADR-0015](0015-descriptive-worlds-dc-and-mods.md),
  first consequence) stand without a route for richer records: a record
  the table cannot express is not packageable until the table grows a row
  for its shape or this route returns. ADR-0015's text says so now.
- The [eark-mods plan](../archive/eark-mods.md) loses decisions 6, 7 and 8
  and its S5 (CLI supplied documents); S3 loses the MODS document check.
  The xmllint pass in S6 stays, because it validates the emitted
  `mods.xml`.
- The input specification keeps operator-supplied descriptive XML as a
  deferred item, as it has since ADR-0011.
- `profiles.Input` has one descriptive field per level. No library caller
  is broken: `DescriptiveDocument` never shipped.
- The route's sentences in `README.md`, `CLAUDE.md` and the design doc go
  with it.
