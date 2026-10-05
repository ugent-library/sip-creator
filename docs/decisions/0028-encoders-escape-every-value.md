# 0028 — The METS and PREMIS templates escape every value and percent-encode every href

Status: **Accepted** (2026-10-05). Supersedes in part [ADR-0014](0014-representations-csv-strict-when-present.md): its rule that `Label` and `Type` must not contain `< > & "` is dropped.

## Context

The METS and PREMIS templates wrote every value as it was. ADR-0014 kept that safe for representation labels and types by refusing the XML-active characters, and deferred escaping until a real need appeared. File names were never covered by such a rule: an essence file named `R&D scan.tif` produced a representation METS with `xlink:href="data/R&D scan.tif"`, which no XML parser accepts, and `create` reported success. The same held for a producer's local identifier with `&` in the package PREMIS, an original file name in the representation PREMIS, and the submitting organization's name in the package METS. The descriptive templates in `profiles/` already escaped every value.

Escaping alone does not make a file name a correct href. commons-ip, and RODA through it, decodes every `xlink:href` before it opens the file: `Utils.extractedRelativePathFromHref` calls `java.net.URLDecoder`, and `EARKSIP` switches that decoding on when it parses a SIP (commons-ip 2.8.0, which RODA 5.7.1 bundles, and 2.11.2, which `build.sh` validates with). `URLDecoder` reads `%41` as `A`, a raw `+` as a space, and throws on a `%` that starts no escape. A file named `100%.jpg` gave an href that is not a valid `anyURI`; one named `100%41.jpg` was read as `100A.jpg`. The descriptive document's href was the one href already encoded, with `url.QueryEscape`, which also encoded the slashes between segments (`metadata%2Fdescriptive%2Fdc.xml`).

## Decision

The METS and PREMIS templates pass every value they read from the package graph through an XML escape (`xml.EscapeText`). Only the IDs and timestamps a template mints itself are written as they are. With that in place, labels, types and the content category may hold any text, and the input reader and `SourcePackage.Validate` no longer refuse `< > & "` in them.

What XML 1.0 cannot carry at all is refused, not escaped: a value that is not valid UTF-8, or that holds a control character other than tab, line feed and carriage return (or U+FFFE, U+FFFF). `build.ValidateXMLText` holds the rule. `SourcePackage.Validate` applies it to file paths, labels, types and the content category, `Definition.WithSubmitter` to the submitter's name and OR-id, and each profile's description `Validate` to every value it writes; the CLI's input reader reports the same findings with file and line.

Every `xlink:href` in the METS documents is written by one function: each byte of the path except the unreserved characters of RFC 3986 (letters, digits, `- . _ ~`) and the slashes between segments is percent-encoded. `R&D 1+2.tif` becomes `R%26D%201%2B2.tif`. That reads back to the file's name under RFC 3986 and under `URLDecoder` alike.

## Alternatives rejected

- **Refuse the XML-active characters in file names as well**, extending ADR-0014's rule: `&` and `'` are ordinary in file names (`R&D`, `Jan's notes`), and producers would have to rename files only to satisfy the tool's templates.
- **Escape only the values that can carry producer text**: correct today, but every new field would need someone to decide whether it can, and a wrong guess yields a document that is not well-formed. Escaping every value is one rule to follow.
- **Keep the label and type rule next to the escaping**: it would refuse labels the templates now write correctly.
- **Refuse `%`, `+`, `#` and `?` in file names** instead of encoding hrefs: the same objection as for `&`, and `+` and `%` are common in file names (`v1+2`, `50%`).
- **Go's `url.PathEscape`**: it leaves `+` as it is, which `URLDecoder` reads as a space.

## Consequences

- Output for values and file names without `< > & " ' % + # ?`, spaces or non-ASCII letters is byte for byte the same as before, except the descriptive document's href, which keeps its slashes: `metadata/descriptive/dc.xml` instead of `metadata%2Fdescriptive%2Fdc.xml`. Both decode to the same path.
- A consumer that opens an href without decoding it finds no file for a name with a space or a non-ASCII letter. The specifications make an href a URI, and the consumers checked (commons-ip, RODA) decode.
- A file name or value with a control character is refused before anything is written. Without the rule, `xml.EscapeText` would replace the character with U+FFFD: the href would name no file and the metadata value would be changed, with `check` passing both.
