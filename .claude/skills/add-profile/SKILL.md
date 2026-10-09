---
name: add-profile
description: >-
  Add a profile to this repository's registry, so --profile and profiles.Get accept
  it: collect the decisions, then change the definition, registry, mapping, example,
  build.sh and docs as docs/development-adding-a-profile.md lists. Use when the user
  asks to add, create or register a new profile or content type.
---

# Add a profile

Add a profile by following docs/development-adding-a-profile.md. Read it first,
together with the files it names for the closest existing profile: its definition in
`profiles/<owner>/profile.go`, its entry in `cli/profile.go`, its example folder, and
for a UGent profile its page under `docs/profiles/`.

## Ask first

Every item under "Decisions" in the guide is the user's to make. Collect them before
you change a file:

1. Ask for all of them in one message. Suggest the value of the closest existing
   profile for each, and say which profile that is.
2. Don't invent a rule. If an answer leaves a value open, ask again. Where a value
   follows from a specification (the profile URL for an E-ARK version, a Meemoo
   requirement), quote the section.
3. For a UGent profile, the answers are the rules on its profile page. Draft that
   page from the answers and show it to the user before you write code.

Then propose a plan in chat: the steps from the guide that apply, the files each one
changes, and the commits you will make. Wait for the user's go. Start each later
step only on a go.

## Steps

1. Follow the steps of the guide in order. Skip step 2 when the profile reuses a
   model. Match the code next to it: the same comment style on the definition, the
   same test style in the registry tests.
2. Run `go test ./...` after steps 3, 4 and 5. Each of them adds a profile to a test
   that iterates every registered profile, so a missing piece fails there.
3. For step 8, use the `validate-profiles` skill. Every profile must report `VALID`
   with no warnings, and the existing profiles must still match their reference
   copies. Create the new reference copy only after the user has seen the result.
4. Read the full diff. Check it against the guide's list: every file it names for
   this kind of profile has changed, or you can say why not.

## Report

Report in chat: the profile's decisions in one short list, the files changed per
step, and the validation result per profile. Propose one-line commit messages
starting with `Added:` and wait for the user's go before committing.
