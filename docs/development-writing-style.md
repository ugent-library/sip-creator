# Writing code comments and documentation

This file holds the rules for code comments and for the markdown in `docs/`. Read it before you write or review a comment or a doc. The rules for all writing, including the banned words and the formatting rules, are in the Tone and style section of [CLAUDE.md](../CLAUDE.md). They apply here too. The `tidy-comments` skill in `.claude/skills/` applies these rules to a file, a package or the current diff.

Write for a colleague who reads the code or the doc for the first time.

## Scope

The rules cover comments in `.go` files and in the shell scripts (`build.sh` and `scripts/`). They also cover the markdown in `docs/` and the markdown files in the repository root. They don't cover:

- generated files: `CONFIG.md` and anything with a `// Code generated` header
- `docs/archive/` and `docs/decisions/`, which are historical records whose existing text stays as it is
- text inside strings, including the METS and PREMIS templates
- `testdata/` and the sample input folders in `examples/`

## Comments that stay

Some comments stay whatever the rules below say:

- Comments a tool reads: `//go:embed` and `//go:generate` directives, build constraints, and the `// Code generated` header.
- A TODO that says what to do, such as `// TODO: model the event's fields.` in `sip/event.go`. Don't delete one because it looks old.

## Code comments

- **Comment the why, not the what.** Say why something exists or why it is done the unobvious way, usually in one sentence. A longer comment is right when it carries a CSIP, Meemoo SIP, METS or PREMIS requirement, or explains a workaround. Delete commented-out code.
- **Delete a comment that adds nothing.** Don't shorten it instead. A reader still has to read a shorter one and find nothing in it. Don't replace a deleted comment with a link to `docs/`.
- **Comment the thing, not its callers.** A comment on a type or field states the data and its constraints ("must be a well-formed premis:premis document"), never where they are checked ("checked at assembly"). The exception: code that deliberately leaves a rule to someone else names that owner, so the next reader doesn't add the check twice. A comment on an interface states what it promises, not which types implement it.
- **Say a rule once, at the level where it holds.** A rule for every type in a package goes in the package doc. A reason given on a type is not repeated on its methods.
- **State the domain fact and its reason in the terms of the specs, not of Go.** Write "dc.xml keeps the producer's identifier, because the catalogue indexes it", not "simpledc does not implement IdentifierSwapper". The same goes for what a function returns: name the value by what it is in the package ("the declaration a representation's METS document carries"), not by how the code makes it ("a copy of base"). Delete a reason that would still be true for any other type or function.
- **Describe what the code does and why it must be so.** Don't argue trade-offs or alternatives not taken. An open design question goes in [TODO.md](TODO.md).
- **Mention something left out only when all three hold:** a reader would expect it, it is left out on purpose, and adding it would break something. Otherwise say nothing. Something left out can be an element not written, a check not made, or a design choice such as no caching.
- **A function's doc comment says what the function does before what it returns or how it fails.** Follow the order in which a caller sees it happen. Write "checks that …. It returns an error if …", not "returns an error unless …". Leave out what the signature already says. A method that deliberately does nothing still says what that means for the caller.
- **A doc comment on an exported identifier starts with its name**, as in "ValidateReceived parses r as XML and checks that the root element is premis:premis". `go doc` and pkg.go.dev list the comment under that name. Comments on tests and unexported code start wherever the sentence reads best. A doc comment is written in full sentences either way.
- **On exported types, document every field, stdlib style.** The comment goes on its own line above the field, as a sentence starting with the field name, never at the end of the line.
- **In unexported code, comment only what a name can't say.** Short inline comments are fine.
- **In tests, comment unusual setup and the regression a test protects against.** Don't repeat the test name or its assertions.

The same rules apply to comments in shell scripts.

## Documentation

- **Lead with what the doc answers.** The first paragraph says which question the file answers and who needs it.
- **Describe the system as it is.** A plan goes in `docs/plans/` and a decision in an ADR, as [README.md](README.md) describes.
- **One doc owns each rule.** Other docs link to it. CLAUDE.md repeats the rules of the architecture from [sip-creator-design.md](sip-creator-design.md) in short form, because an agent reads CLAUDE.md in every session.
- **Name concrete things:** files, packages, functions, scripts, spec versions. Write "`SourcePackage.Validate`", not "the validators downstream".

## Examples

Before: the failure path comes first, the condition is negative, two clauses are joined with a semicolon, and the reason is paraphrased.

```go
// ValidateReceived returns an error unless r parses as XML and its root
// element is premis:premis in the PREMIS 3 namespace. It does not validate
// against the PREMIS schema; that stays external (ADR-0003).
```

After: what it does, then what it returns, one fact per sentence, the reason cited.

```go
// ValidateReceived parses r as XML and checks that the root element is
// premis:premis in the PREMIS 3 namespace. It returns an error if this
// check fails. It does not validate against the PREMIS schema (ADR-0003).
```

Before: the method that does nothing says only what it returns, and the reason is true of any document.

```go
// ValidateRequired returns nil. The document's schema decides what it must
// state, and no profile reads an identifier or a title from a supplied
// document.
```

After: what that means for the caller, then a reason that is specific to this method.

```go
// ValidateRequired accepts every supplied document and returns nil.
// Package build does not read a document's content (ADR-0003), so it
// cannot tell whether the document states an identifier and a title.
```

Before: the comment says how the code makes the value: a copy, a parameter name, a comparison with that parameter.

```go
// representationDeclaration returns a copy of base for a representation's
// METS document. When EmitRepresentationType is set, the copy's content
// typing names typ, as EmitRepresentationType describes. Otherwise the copy
// equals base.
```

After: what the value is in the package, then the one case where it differs.

```go
// representationDeclaration returns the declaration a representation's METS
// document carries. When the profile emits representation types, the
// declaration names typ as the representation's type, as
// EmitRepresentationType describes. Otherwise it is the package's
// declaration.
```

Before: the last two sentences muse about caching, which the code doesn't do and nobody asked about.

```go
// Root reads the whole file with xmldoc.Root and returns its root element.
// It returns an error if the file cannot be opened or is not XML. Root
// reads the file again on every call and caches nothing, so the type stays
// a plain value. Descriptive documents are small enough for that.
```

After: what it does and what it returns, nothing else.

```go
// Root reads the whole file with xmldoc.Root and returns its root element.
// It returns an error if the file cannot be opened or is not XML.
```
