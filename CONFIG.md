# Environment Variables

## Config

Application config

 - Format identification tool (see formats/).
   - `SIP_FILE_FORMAT_NAME` (**required**, non-empty) - Name of the format identification tool. Valid values: siegfried.
   - `SIP_FILE_FORMAT_COMMAND` (**required**, non-empty) - Path to the tool's binary on this system.
   - `SIP_FILE_FORMAT_ARGS` (**required**, non-empty) - Extra arguments passed to the tool (e.g. "-hash md5 -json" for siegfried).
 - Build provenance, stamped by the deployment environment. Unused by the CLI itself.
   - `SOURCE_BRANCH` - Git branch this build was made from.
   - `SOURCE_COMMIT` - Git commit this build was made from.
   - `IMAGE_NAME` - Name of the container image carrying this build.

