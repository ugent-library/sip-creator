# Environment Variables

## config

Application config: the CLI's operator contract, read from the
environment. The library never sees it — embedding systems supply their
destination and logger as data (profiles.Config), and each package's
material — characterization report included — as a profiles.Input, not
via env vars. Format info comes from the siegfried.json sidecar in the
input folder (ADR-0009), not from configuration.

 - The submitting organization, stamped into every package's METS as a
CREATOR agent. `create` requires NAME for every profile and OR_ID for
meemoo profiles; how required each is depends on the profile, so the
check lives at profile resolution, not here.
   - `SIP_SUBMITTER_NAME` - Name of the submitting organization, e.g. "Universiteitsbibliotheek Gent".
   - `SIP_SUBMITTER_OR_ID` - The organization's meemoo OR-id (its identifier in meemoo's
organization register), e.g. "OR-a1b2c3d". Required for meemoo
profiles, where it becomes the agent's IDENTIFICATIONCODE note.
 - `SIP_CONTENT_CATEGORY` - Default content category for created packages (mets/@TYPE, CSIP
content-category vocabulary), e.g. "Photographs – Digital". Empty
means the profile's registry value; --content-category overrides
both per run.

