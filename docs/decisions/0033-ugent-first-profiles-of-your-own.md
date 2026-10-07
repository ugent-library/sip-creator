# 0033 — The library serves UGent Library first; a profile of your own stays possible

Status: **Accepted** (2026-10-06). Supersedes in part [ADR-0022](0022-reference-implementation-bring-your-own-profile.md): the library is no longer a reference implementation for other institutions, and the MODS model's "speak MODS for others" premise no longer holds on its own. ADR-0022's extension point and its typed constants stand. **Note, 2026-10-07:** the MODS vocabulary question below is settled by [ADR-0034](0034-a-profile-is-a-content-type.md): the model keeps the library's catalogue words, as UGent's application profile of MODS that `ugent/bibliographic` writes.

## Context

[ADR-0022](0022-reference-implementation-bring-your-own-profile.md) made the library a reference implementation of E-ARK SIP packaging that other institutions can use as it stands, with its documents written for them as much as for UGent. No other institution has asked for that, and nobody outside UGent depends on the module. An assessment of the project on 2026-10-06 named the claim as a promise without a caller. It had already shaped work: the README invited outside callers before there was a tagged release, and the parked mods-coverage plan rests on the argument that other institutions would otherwise translate UGent's words twice.

The mechanism ADR-0022 described is a different matter. A profile is a `build.Definition` with a metadata model, handed to `build.New`; the engine imports no profile ([ADR-0018](0018-engine-and-profile-packages.md)). The three in-tree profiles are built that way, and a program that embeds the library can keep a profile of its own without changing this repository. Since 2026-10-06 such a model can also ship its own XSDs and name a format METS does not list.

## Decision

**SIP Creator is an implementation of several distinct profiles, built first for UGent Library's use: SIPs for Meemoo and for UGent's RODA.** Others may find it useful as it is; its documents are written for UGent's operators and developers, not for other institutions.

**A profile of your own stays possible, as architecture, not as a promise.** The extension point stays as it is, because the in-tree profiles use it and a program that embeds the library may need it. The docs describe how it works; they do not invite outside institutions or promise support.

**Typed constants for values a profile leaves to its caller stay** (ADR-0022): two programs that make the same choice write the same value, which holds within UGent as well.

**The MODS model's vocabulary is decided again when the mods-coverage plan resumes.** ADR-0022 chose MODS's own words over UGent's catalogue words for the sake of other institutions. Without that audience, the choice between them is open; the plan records the question.

## Alternatives rejected

- **Keep ADR-0022 as it stands.** It keeps a promise nobody asked for and the work it implies: writing every document for an outside reader, and a release process before any outside caller exists.
- **Remove the extension point as well.** It is how the in-tree profiles are built, and removing it would tie the engine to them again ([ADR-0018](0018-engine-and-profile-packages.md)). A program that embeds the library may need a profile of its own.

## Consequences

- `CLAUDE.md`'s audience section, the README and the design doc stop addressing other institutions. The README's section on profiles of your own describes the mechanism for a program that embeds the library.
- The parked mods-coverage plan carries a note that its premise needs a new decision.
- A tagged release and a changelog are no longer owed to outside callers. They come when UGent's own use calls for them, such as a stable version for the ingest system to pin.
- Archived plans and other ADRs that cite ADR-0022 keep their text as written.
