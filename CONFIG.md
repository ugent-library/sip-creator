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
 - Build provenance, stamped by the deployment environment. Unused by the CLI itself.
   - `SOURCE_BRANCH` - Git branch this build was made from.
   - `SOURCE_COMMIT` - Git commit this build was made from.
   - `IMAGE_NAME` - Name of the container image carrying this build.

