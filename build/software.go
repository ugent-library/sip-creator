package build

import (
	"runtime/debug"

	"github.com/ugent-library/sip-creator/sip"
)

// softwareAgent is the metsHdr agent naming the software that built the
// package. E-ARK CSIP requires it in every package, so the engine adds it
// rather than leaving it to each profile.
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

// softwareVersion is the module version the Go toolchain stamped into the
// binary: the tag for a tagged go install, otherwise a pseudo-version
// carrying the commit, with +dirty when the working tree had uncommitted
// changes. A binary without that information, such as a test binary,
// reports (devel), Go's own word for an unversioned build.
func softwareVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "(devel)"
	}
	return info.Main.Version
}
