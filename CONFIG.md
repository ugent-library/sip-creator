# Environment Variables

## config

The CLI's configuration, read from the environment and an optional
.env file. The library never reads it. A program that embeds the library
passes a build.Config, and a build.SourcePackage per build, instead.
Format information arrives in siegfried.json (ADR-0009), not in
configuration.

 - The submitting organization, written into every package's METS as a
CREATOR agent.
   - `SIP_SUBMITTER_NAME` - Name of the submitting organization, for example "Example
Organization". `create` requires it for every profile.
   - `SIP_SUBMITTER_OR_ID` - The organization's Meemoo OR-id (its identifier in Meemoo's
organization register), for example "OR-a1b2c3d". `create`
requires it for Meemoo profiles, where it becomes the agent's
IDENTIFICATIONCODE note.
 - `SIP_CONTENT_CATEGORY` - Default content category for created packages (mets/@TYPE, from the
CSIP content-category vocabulary), for example "Photographs – Digital".
When it is empty, the profile's value applies. The --content-category
flag overrides both for one run.

