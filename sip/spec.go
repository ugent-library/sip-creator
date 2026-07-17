package sip

// Spec holds the profile-level values a METS document declares: the
// profile URL, the content typing, and the agents responsible for the
// package. Profiles differ in these values, not in build logic — Spec is
// data the templates read, set once on the Package at assembly.
type Spec struct {
	ProfileURL                  string // mets/@PROFILE
	Type                        string // mets/@TYPE (TODO: draw from the CSIP content-category vocabulary)
	ContentInformationType      string // mets/@csip:CONTENTINFORMATIONTYPE
	OtherContentInformationType string // mets/@csip:OTHERCONTENTINFORMATIONTYPE; rendered only when set
	DescriptiveMDType           string // dmdSec mdRef @MDTYPE
	DescriptiveMDTypeVersion    string // dmdSec mdRef @MDTYPEVERSION; rendered only when set
	Agents                      []Agent
}

// Agent is one metsHdr agent entry.
type Agent struct {
	Role      string // agent/@ROLE
	OtherRole string // agent/@OTHERROLE
	Type      string // agent/@TYPE
	OtherType string // agent/@OTHERTYPE
	Name      string
	Note      string // optional; rendered with csip:NOTETYPE="SOFTWARE VERSION"
}
