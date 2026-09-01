# 0012 — The eark profile keeps the producer's identifier in dc.xml

Status: Accepted (2026-09-01)

## Context

Assembly used to replace the descriptive `dcterms:identifier` with the entity's
UUID in every profile. That swap implements a meemoo SIP 1.2 rule: descriptive
and preservation metadata are linked by a shared identifier, stored in the
`premis:objectIdentifier` of the intellectual entity and in the
`dcterms:identifier` of `dc+schema.xml`. The primary PREMIS object identifier
must be a UUID, so the descriptive document must carry the UUID. The producer's
own identifier survives as a second `premis:objectIdentifier` of type
`MEEMOO-LOCAL-ID`, which the spec recommends for exactly this purpose.

The eark profile inherited the swap but not the second half: it emits no PREMIS
(RODA drops package PREMIS that describes no agents or events), so the
producer's identifier (for example an Alma MMS ID) was required as input,
overwritten during assembly, and appeared nowhere in the finished package.
That loss matters: operators find and work with archived material in RODA by
searching the identifier they know, and RODA indexes the simple-DC fields of
`dc.xml` into its catalogue search. A `dc.xml` whose only identifier is a UUID
minted at build time is useless for that.

## Decision

The swap is profile data: `Definition.SwapObjectIdentifier`. The meemoo family
keeps swapping, because the spec's linking rule requires it. The eark profile
does not swap: `dc.xml` carries the producer's `dcterms:identifier` unchanged,
at package and representation level. CSIP has no rule tying the descriptive
identifier to the package identifier; the package UUID stays in `mets/@OBJID`
and in the METS IDs. The input rule is unchanged: exactly one
`dcterms:identifier`, still required.

## Alternatives rejected

- **Emit both identifiers in `dc.xml`** (`identifier` is repeatable in plain
  Dublin Core): requires relaxing the exactly-one-identifier rule per family,
  and the second value serves no reader. Machines read `mets/@OBJID`; operators
  search their own identifier. Two untyped identifiers in one document is the
  ambiguity the exactly-one rule exists to prevent.
- **Emit eark PREMIS carrying the local identifier**: RODA drops package PREMIS
  without agents or events, and where it does keep PREMIS it treats it as
  technical metadata, not something the catalogue search surfaces to an
  operator.

## Consequences

- The two profiles emit different identifiers into their descriptive
  documents: meemoo's `dc+schema.xml` carries the entity UUID (with the local
  identifier in PREMIS as `MEEMOO-LOCAL-ID`); eark's `dc.xml` carries the
  producer's identifier.
- The eark package records its own UUID only in METS, no longer in `dc.xml`.
  Nothing consumed it there.
- The flag's zero value means no swap, so a future profile added without
  reading this history does not silently destroy producer metadata.
