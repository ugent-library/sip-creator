---
name: tidy-comments
description: >-
  Review and rewrite Go code comments in this repository so they follow the
  comment rules in CLAUDE.md. Use when the user asks to clean up, tidy or review comments
  in a file, package or the current diff. Takes an optional target: a file, a package
  path, or nothing for the uncommitted diff.
---

# Tidy comments

Rewrite comments so a colleague reading the code for the first time gets the why,
in plain language, and nothing they could read from the code itself.

## Scope

- Target: the argument (a file or package path), or else the files in `git diff HEAD`.
  In diff mode, touch only comments on changed lines or on declarations whose code changed.
- Go files only. Leave out `docs/archive/`, `docs/decisions/`, generated files
  (`CONFIG.md`, anything with a `// Code generated` header) and testdata.
- Comments only. Never change code, identifiers or template output. If a comment
  can't be fixed without a rename, propose the rename in chat instead (CLAUDE.md:
  discuss renames first).

## What to fix

Work through each comment and apply these rules, in this order:

1. **Delete** what the code already says: comments that repeat the function name,
   narrate the code (a loop, a variable, the next line), or describe internals
   step by step. Keep the steps a function takes when a caller can observe them:
   what it reads, what it checks, what it returns. Those are what the function
   promises. Delete commented-out code.
2. **Delete or move** comments about callers. A comment on a type or field states the
   data and its constraints, not where it is checked, who calls it, or the order in
   which other packages process it. That order belongs in
   docs/sip-creator-design.md. Keep it only when the code deliberately leaves a rule
   to someone else; then it names that owner.
   The same goes for implementations: a comment on an interface states what the
   interface promises, not the concrete types that satisfy it or which packages do
   and don't depend on it. Such a list goes stale with every new implementation, and
   gopls and pkg.go.dev already show it.
3. **Say it once, at the level where it holds.** A rule that holds for every type
   in a package goes in the package doc, once. Don't repeat it on a type or a
   field: a reader takes the repeat to mean the rule is special to that type.
   A reason already given on a type is not repeated on its methods.
4. **Keep the why.** Why something exists, or why it is done the unobvious way, in
   one sentence where possible. A longer comment stays when it carries a domain rule
   (a CSIP, Meemoo SIP, METS or PREMIS requirement) or explains a workaround.
5. **Mention an absence only when it is a decision.** Say that something is left
   out (an interface not implemented, an element not written, a check not made)
   only when all three hold: a reader would expect it, because a sibling type or
   package build has it; it is left out on purpose, for a reason you can name or cite;
   and adding it would break something. Otherwise say nothing: every type lacks
   far more than it has. An absence includes a design choice not made: no
   caching, no locking, no retry, no buffering. The same three conditions apply.
6. **No musing.** Don't argue for or against a design in a comment: no remarks on
   performance trade-offs, alternatives not taken, or why something is "good
   enough" ("caches nothing", "small enough for that", "so the type stays a plain
   value"). A comment describes what the code does and why it must be so. An open
   design question goes in docs/TODO.md, not in the code.
7. **State the domain fact, then the reason.** Write "x, because y" in the terms
   of the package and the specs, not "x does not implement z" in the terms of Go.
   Write "dc.xml keeps the producer's identifier, because the catalogue indexes
   it", not "simpledc does not implement IdentifierSwapper". A reader who knows
   the mechanism connects the two; a reader who doesn't learns nothing from an
   interface name.
8. **A reason must add information.** Delete a reason that is true by definition
   or would hold for any code: "the schema decides what the document must state",
   "the profile's rule is the profile's". Test: would the sentence still be true
   if you swapped in another type or function? Then it explains nothing.
9. **Justify with specs, not tools.** Point to the CSIP or Meemoo rule, not to how
   commons-ip or RODA behave, unless the comment explains a validator workaround.
   When an ADR or a spec section gives the reason, cite it inside the sentence and
   stop: "It does not validate against the PREMIS schema (ADR-0003)." Don't
   paraphrase the reason in your own words, and don't copy an ADR's or a design
   doc's description of the design into a comment. Cite it, or state only the fact
   the reader of this code needs.
10. **Exported types:** every field documented, on its own line above the field, as
   a sentence starting with the field name. No end-of-line comments on exported
   fields. In unexported code, comment only what a name can't say.
11. **Say what it does, then what it returns.** A function's doc comment states what
   the function does in active, positive terms, in the order a caller sees it
   happen. Then it states the result, and what happens on failure. For a
   `Validate…` function: "X checks that …. It returns an error if …". Don't open
   with the failure path, and don't use "unless" or a double negative.
   A method that deliberately does nothing, such as one that always returns nil,
   still says what that means for the caller: what passes, what is skipped. Then
   why. Write "ValidateRequired accepts every supplied document and returns nil",
   not "ValidateRequired returns nil".
12. **Doc comments** start with the exact name of the thing and are full sentences.
13. **One reading only.** State the rule, then the exceptions, not a list of
   exceptions that leaves the reader to work out the rule. When a field is set in
   some cases and empty in the rest, name the cases where it is set. Don't nest a
   clause or a parenthesis inside an item of a list.
14. **One fact per sentence.** Write short sentences. Don't join two clauses with
   a semicolon; write two sentences. If a reader has to invert a condition to
   understand it, rewrite it.
15. **Write the condition, not a name for it.** Don't use a generic programming
   term that stands in for a concrete fact, such as "invariant", "contract",
   "idempotent" or "side effect". Write "every node has an identifier, a path and
   a media type", not "the graph's invariants".
16. **Language:** apply the Tone and style section of CLAUDE.md: no banned words,
   no programming jargon, no vague qualifiers that name no tool or step ("the
   validators downstream"), no em-dashes, no invented abbreviations or hyphenated
   shorthand, checks named by what they do, "Meemoo" capitalized in prose, folder
   for the input and directory for the package.
17. **Never invent facts.** If you can't tell why code exists, leave the comment and
   list it as a question in the report.

## Examples

Before: the failure path comes first, the condition is negative, two clauses are
joined with a semicolon, and the reason is paraphrased.

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

Before: the method that does nothing says only what it returns, and the reason
is true of any document.

```go
// ValidateRequired returns nil. The document's schema decides what it must
// state, and no profile reads an identifier or a title from a supplied
// document.
```

After: what that means for the caller, then a reason that is specific to this
method.

```go
// ValidateRequired accepts every supplied document and returns nil.
// Package build does not read a document's content (ADR-0003), so it
// cannot tell whether the document states an identifier and a title.
```

Before: the last two sentences muse about caching, which the code doesn't do and
nobody asked about.

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

## Steps

1. List the target files and read each one in full.
2. Edit comments in place, file by file.
3. Re-read each comment you rewrote as someone new to the code, and read it aloud.
   If a sentence can be read two ways, if you have to invert a condition, or if
   you have to reread it to find where a clause or list item ends, rewrite it.
   For each sentence after the first, ask: does a reader need this to call or
   change the code? If it argues, speculates or repeats the type's doc, delete it.
4. Run `gofmt -l` on the touched files and `go vet ./...`; both must be clean.
5. Report in chat: per file, what was deleted, rewritten or kept with a question.
   Propose a one-line commit message starting with `Changed:` and wait for the
   user's go before committing.
