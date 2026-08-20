package sip

import (
	"fmt"
	"strings"
)

// Spec holds the profile-level values a METS document declares: the
// profile URL, the content typing, and the agents responsible for the
// package. Profiles differ in these values, not in build logic — Spec is
// data the templates read, set once on the Package at assembly.
type Spec struct {
	ProfileURL                  string // mets/@PROFILE
	Type                        string // mets/@TYPE (content category; overridable per run)
	ContentInformationType      string // mets/@csip:CONTENTINFORMATIONTYPE
	OtherContentInformationType string // mets/@csip:OTHERCONTENTINFORMATIONTYPE; rendered only when set
	DescriptiveMDType           string // dmdSec mdRef @MDTYPE
	DescriptiveMDTypeVersion    string // dmdSec mdRef @MDTYPEVERSION; rendered only when set
	// RecordStatus is metsHdr/@RECORDSTATUS (SIP3), rendered only when set:
	// the E-ARK SIP spec defines an absent status as equal to NEW.
	RecordStatus string
	Agents       []Agent
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
// update of an earlier one — a package that must reuse the original's
// identifier as its mets/@OBJID.
func IsUpdateRecordStatus(status string) bool {
	switch strings.ToUpper(status) {
	case "SUPPLEMENT", "REPLACEMENT", "VERSION", "DELETE":
		return true
	}
	return false
}

// Agent is one metsHdr agent entry.
type Agent struct {
	Role      string // agent/@ROLE
	OtherRole string // agent/@OTHERROLE
	Type      string // agent/@TYPE
	OtherType string // agent/@OTHERTYPE
	Name      string
	Note      string // optional
	NoteType  string // note/@csip:NOTETYPE; required when Note is set
}
