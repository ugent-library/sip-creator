---
name: tidy-comments
description: >-
  Review and rewrite code comments in this repository so they follow the rules in
  docs/development-writing-style.md and the Tone and style section of CLAUDE.md. Use
  when the user asks to clean up, tidy or review comments in a file, package or the
  current diff. Takes an optional target: a file, a package path, or nothing for the
  uncommitted diff.
---

# Tidy comments

Rewrite comments so a colleague reading the code for the first time gets the why,
in plain language, and nothing they could read from the code itself.

## Scope

- Target: the argument (a file or package path), or else the files in `git diff HEAD`.
  In diff mode, touch only comments on changed lines or on declarations whose code changed.
- Go files only, minus what the Scope section of docs/development-writing-style.md
  excludes.
- Comments only. Never change code, identifiers or template output. If a comment
  can't be fixed without a rename, propose the rename in chat instead (CLAUDE.md:
  discuss renames first). If you find a bug, report it in chat and leave the code as
  it is.
- Run `git status` first. Leave changes you didn't make as they are.

## What to fix

Read docs/development-writing-style.md and the Tone and style section of CLAUDE.md
first. Then check each comment in the order below. Each item adds only what a review
needs on top of those rules.

Leave the comments listed under "Comments that stay" in the style doc exactly as
they are. Check that list before you change any comment.

1. **Delete what the code already says.** This covers comments that repeat the
   function name, narrate the code (a loop, a variable, the next line), or describe
   internals step by step. Keep the steps a caller can observe: what the function
   reads, what it checks, what it writes, what it returns. Those are what the
   function promises. Delete commented-out code, summaries of past changes, and
   banners that repeat the file's structure. Delete such a comment. Don't shorten it.
2. **Delete or move comments about callers and implementations.** The order in which
   packages process something belongs in docs/sip-creator-design.md. A list of the
   types that implement an interface goes stale with every new one, and gopls and
   pkg.go.dev already show it.
3. **Say it once.** A reader takes a repeat on a type or a field to mean the rule is
   special to that type.
4. **Keep the why.** A longer comment stays when it carries a CSIP, Meemoo SIP, METS
   or PREMIS requirement or explains a workaround.
5. **Absences.** All three conditions in the style doc must hold. Otherwise say
   nothing: every type lacks far more than it has. A design choice not made counts
   as an absence too: no caching, no locking, no retry, no buffering.
6. **No musing.** Remarks like "caches nothing", "small enough for that" or "so the
   type stays a plain value" go.
7. **Domain fact, then reason.** A reader who knows the mechanism connects the two.
   A reader who doesn't learns nothing from an interface name.
8. **A reason must add information.** Test: would the sentence still be true if you
   swapped in another type or function? Then delete it. Examples: "the schema
   decides what the document must state", "the profile's rule is the profile's".
9. **Cite and stop.** Don't copy an ADR's or a design doc's description of the
   design into a comment. Cite it, or state only the fact the reader of this code
   needs: "It does not validate against the PREMIS schema (ADR-0003)."
10. **Exported fields.** Apply the rule in the style doc as written.
11. **What it does, then what it returns.** A method that deliberately does nothing,
    such as one that always returns nil, still says what that means for the caller:
    what passes, what is skipped. Then why. Write "ValidateRequired accepts every
    supplied document and returns nil", not "ValidateRequired returns nil".
12. **Doc comments.** A doc comment on an exported identifier starts with its exact
    name. Every doc comment is written in full sentences.
13. **One reading only.** Don't nest a clause or a parenthesis inside an item of a
    list. If a reader has to invert a condition to understand it, rewrite it.
14. **Language.** The words the Tone and style section of CLAUDE.md bans go. Vague
    qualifiers that name no tool or step go ("the validators downstream"). "Meemoo"
    is capitalized in prose.
15. **Never invent facts.** Check a comment's claim against the code before you
    rewrite it. If you can't tell why code exists, leave the comment and list it as
    a question in the report. If a comment contradicts the code and you can't tell
    which one is right, leave both and report it.

The style doc has before/after examples taken from this repository's code.

## Steps

1. List the target files and read each one in full. Search may find candidates, but
   read every comment next to its code before you change it.
2. Edit comments in place, file by file. Never delete comments by pattern, with
   `sed` or a regular expression.
3. Re-read each comment you rewrote as someone new to the code, and read it aloud.
   If a sentence can be read two ways, if you have to invert a condition, or if
   you have to reread it to find where a clause or list item ends, rewrite it.
   For each sentence after the first, ask: does a reader need this to call or
   change the code? If it argues, speculates or repeats the type's doc, delete it.
4. Run `gofmt -l` on the touched files and `go vet ./...`. Both must be clean.
5. Read the full diff. Every change must be a comment.
6. Report in chat: per file, a short summary of what was deleted or rewritten, and
   each comment you kept with a question. Don't list every deletion. Propose a
   one-line commit message starting with `Changed:` and wait for the user's go
   before committing.
