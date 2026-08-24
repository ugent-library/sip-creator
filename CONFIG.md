# Environment Variables

## config

Application config: the CLI's operator contract, read from the
environment. The library never reads it: embedding systems pass a
profiles.Config and per-build profiles.Input instead, and format info
arrives via the siegfried.json sidecar (ADR-0009), not configuration.

 - The submitting organization, stamped into every package's METS as a
CREATOR agent. `create` requires NAME for every profile and OR_ID for
meemoo profiles.
   - `SIP_SUBMITTER_NAME` - Name of the submitting organization, e.g. "Universiteitsbibliotheek Gent".
   - `SIP_SUBMITTER_OR_ID` - The organization's meemoo OR-id (its identifier in meemoo's
organization register), e.g. "OR-a1b2c3d". Required for meemoo
profiles, where it becomes the agent's IDENTIFICATIONCODE note.
 - `SIP_CONTENT_CATEGORY` - Default content category for created packages (mets/@TYPE, CSIP
content-category vocabulary), e.g. "Photographs – Digital". Empty
means the profile's registry value; --content-category overrides
both per run.

