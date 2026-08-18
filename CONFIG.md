# Environment Variables

## config

Application config: the CLI's operator contract, read from the
environment. The library never sees it — embedding systems supply their
logger and identificator as data (profiles.Config), not via env vars.

 - Format identification tool (see formats/). Optional: leaving NAME
empty disables format identification — premis:format is a SHOULD,
and essence fixity is computed natively during the copy.
   - `SIP_FILE_FORMAT_NAME` - Name of the format identification tool; empty disables format
identification. Valid values: siegfried.
   - `SIP_FILE_FORMAT_COMMAND` - Path to the tool's binary on this system. Required when NAME is set.
   - `SIP_FILE_FORMAT_ARGS` - Extra arguments passed to the tool (e.g. "-json" for siegfried).
 - The submitting organization, stamped into every package's METS as a
CREATOR agent. `create` requires NAME for every profile and OR_ID for
meemoo profiles; how required each is depends on the profile, so the
check lives at profile resolution, not here.
   - `SIP_SUBMITTER_NAME` - Name of the submitting organization, e.g. "Universiteitsbibliotheek Gent".
   - `SIP_SUBMITTER_OR_ID` - The organization's meemoo OR-id (its identifier in meemoo's
organization register), e.g. "OR-a1b2c3d". Required for meemoo
profiles, where it becomes the agent's IDENTIFICATIONCODE note.
 - Build provenance, stamped by the deployment environment. Unused by the CLI itself.
   - `SOURCE_BRANCH` - Git branch this build was made from.
   - `SOURCE_COMMIT` - Git commit this build was made from.
   - `IMAGE_NAME` - Name of the container image carrying this build.

