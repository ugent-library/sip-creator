package sip

import (
	"fmt"
)

// Spec holds the profile-level values a METS document declares: the
// profile URL, the content typing, and the agents responsible for the
// package. Profiles differ in these values, not in build logic: Spec is
// data the templates read, set once on the Package at assembly.
type Spec struct {
	// ProfileURL is mets/@PROFILE.
	ProfileURL string
	// Type is mets/@TYPE, the content category; overridable per run.
	Type string
	// ContentInformationType is mets/@csip:CONTENTINFORMATIONTYPE.
	ContentInformationType string
	// OtherContentInformationType is mets/@csip:OTHERCONTENTINFORMATIONTYPE,
	// rendered only when set.
	OtherContentInformationType string
	// DescriptiveMDType is the dmdSec mdRef @MDTYPE.
	DescriptiveMDType string
	// DescriptiveMDTypeVersion is the dmdSec mdRef @MDTYPEVERSION, rendered
	// only when set.
	DescriptiveMDTypeVersion string
	// RecordStatus is metsHdr/@RECORDSTATUS (SIP3), rendered only when set:
	// the E-ARK SIP spec defines an absent status as equal to NEW.
	RecordStatus string
	// Agents are the metsHdr agent entries.
	Agents []Agent
}

// recordStatuses is the SIP3 vocabulary for metsHdr/@RECORDSTATUS.
var recordStatuses = map[string]bool{
	"NEW": true, "SUPPLEMENT": true, "REPLACEMENT": true,
	"TEST": true, "VERSION": true, "DELETE": true,
}

// ValidateRecordStatus returns why status is not a legal
// metsHdr/@RECORDSTATUS value (SIP3 vocabulary).
func ValidateRecordStatus(status string) error {
	if !recordStatuses[status] {
		return fmt.Errorf("record status %q is not in the SIP3 vocabulary (NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE)", status)
	}
	return nil
}

// IsUpdateRecordStatus reports whether status declares this package an
// update of an earlier one: a package that must reuse the original's
// identifier as its mets/@OBJID. Like ValidateRecordStatus, it expects the
// uppercase SIP3 form; callers normalize case before calling either.
func IsUpdateRecordStatus(status string) bool {
	switch status {
	case "SUPPLEMENT", "REPLACEMENT", "VERSION", "DELETE":
		return true
	}
	return false
}

// Agent is one metsHdr agent entry.
type Agent struct {
	// Role is agent/@ROLE.
	Role string
	// OtherRole is agent/@OTHERROLE.
	OtherRole string
	// Type is agent/@TYPE.
	Type string
	// OtherType is agent/@OTHERTYPE.
	OtherType string
	// Name is the agent's name.
	Name string
	// Note is the agent's optional note.
	Note string
	// NoteType is note/@csip:NOTETYPE; required when Note is set.
	NoteType string
}
