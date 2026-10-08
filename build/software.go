package build

import (
	"runtime/debug"

	"github.com/ugent-library/sip-creator/sip"
)

// softwareAgent returns the metsHdr agent naming the software that built
// the package. E-ARK CSIP requires it in every package, so Builder.Build
// adds it rather than leaving it to each profile.
func softwareAgent() sip.Agent {
	return sip.Agent{
		Role:      "CREATOR",
		Type:      "OTHER",
		OtherType: "SOFTWARE",
		Name:      "SIP Creator",
		Note:      softwareVersion(),
		NoteType:  "SOFTWARE VERSION",
	}
}

// softwareVersion returns the module version the Go toolchain stamped into
// the binary: the tag for a tagged go install, otherwise a pseudo-version
// carrying the commit, with +dirty when the working tree had uncommitted
// changes. For a binary without that information, such as a test binary,
// it returns (devel), Go's own word for an unversioned build.
func softwareVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
