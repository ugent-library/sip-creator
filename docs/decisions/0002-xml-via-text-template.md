# 0002 — Generate XML with text/template, not an XML library

Status: **Accepted** (2026-07-14)

## Context

The tool's entire output is XML: descriptive metadata (Dublin Core + schema.org), PREMIS, and METS at both representation and package level. The obvious Go reflex is `encoding/xml` (or a third-party XML builder): marshal structs to elements. But the target documents are constrained by external XSDs and by two layered profiles (E-ARK CSIP + meemoo SIP 2.0) that dictate exact element order, attribute presence, namespace prefixes, and `schemaLocation` values. What a validator accepts is a precise, fussy shape — not "some serialization of these fields."

`encoding/xml` fights this: attribute and namespace control is awkward, element ordering is struct-order (not easily conditional), and matching a hand-authored reference document byte-for-shape means bending the marshaller. The reference material for both specs is *example XML documents*, not schemas-as-code.

## Decision

**All XML is generated with Go `text/template`, one template per document type, not with an XML library.** Each encoder package (`encoders/metadata`, `encoders/mets`, `encoders/premis`) holds a `text/template` whose text *is* the target document, with fields and ranges interpolated from the `sip/` graph.

This makes the template a near-literal transcription of the spec's example document: element order, namespace declarations, `xsi:schemaLocation`, and CSIP attributes are written exactly as the validator expects and read side-by-side with the spec. Conditional inclusion (`{{- if .Field }}`) and repetition (`{{ range }}`) are explicit and local.

## Alternatives rejected

- **`encoding/xml` (stdlib marshalling).** Rejected: awkward, verbose control over namespaces/prefixes and attribute ordering; element order tied to struct layout; conditional/optional elements clumsy. Matching a fussy, externally-specified document shape means constantly working around the marshaller instead of with it.
- **A third-party XML builder library.** Rejected on two grounds: it buys little over `text/template` for this fill-in-a-fixed-shape problem, and it adds a dependency. The project deliberately keeps its footprint small and boring; a new dependency is a serious choice, and this one does not earn its place.

## Consequences

- **Templates are the source of truth for document shape**, transcribed from the spec examples and validated externally by commons-ip (see [ADR-0003](0003-validation-stays-external.md)). Correctness is checked against the real validator, not against a marshaller's idea of valid XML.
- **The templates are unaware of the XSDs.** Nothing enforces that a template stays consistent with the schema it claims — that is exactly what external validation catches. A malformed template produces malformed output silently until validated.
- **This is a standing constraint, not a default to drift from.** Keep XML in `text/template` unless there is a strong, discussed reason to change. Reconsider only if the document shapes become dynamic enough that fixed templates no longer express them — not the case for the current profiles.
- A known cost of hand-authored templates: literals meant to be profile-specific (e.g. the meemoo `TYPE` and profile URL) currently sit inside shared templates. That is a factoring problem to fix by parameterizing the templates, not a reason to abandon them — see the [refactoring plan](../plans/refactoring-plan.md).
