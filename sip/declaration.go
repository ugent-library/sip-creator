package sip

import (
	"fmt"
	"strings"
)

// MetsDeclaration holds the profile-level values a METS document declares:
// the profile URL, the content typing, and the agents responsible for the
// package.
type MetsDeclaration struct {
	// ProfileURL is mets/@PROFILE.
	ProfileURL string
	// Type is mets/@TYPE, the content category.
	Type string
	// OtherType is mets/@csip:OTHERTYPE, which CSIP requires when Type is
	// "Other".
	OtherType string
	// ContentInformationType is mets/@csip:CONTENTINFORMATIONTYPE.
	ContentInformationType string
	// OtherContentInformationType is mets/@csip:OTHERCONTENTINFORMATIONTYPE.
	OtherContentInformationType string
	// RecordStatus is metsHdr/@RECORDSTATUS. Only the package METS
	// carries it, and only when it is set, because the E-ARK SIP
	// specification reads an absent status as NEW.
	RecordStatus RecordStatus
	// Agents are the metsHdr agent entries. Builder.Build adds the
	// software agent, and Definition.WithSubmitter the submitting
	// organization. A profile lists only the other agents its archive asks
	// for.
	Agents []Agent
}

// RecordStatus is metsHdr/@RECORDSTATUS: what the package does to the
// archive's holdings. The values the tool knows are the E-ARK SIP
// vocabulary (SIP3) and UPDATE. A profile can narrow them to the ones its
// archive acts on (build.Definition.RecordStatuses). SIP3 also lists
// OTHER, which the tool leaves out: METS has no attribute that says which
// other status is meant, so an archive cannot act on it.
type RecordStatus string

// The SIP3 vocabulary.
const (
	RecordStatusNew         RecordStatus = "NEW"
	RecordStatusSupplement  RecordStatus = "SUPPLEMENT"
	RecordStatusReplacement RecordStatus = "REPLACEMENT"
	RecordStatusTest        RecordStatus = "TEST"
	RecordStatusVersion     RecordStatus = "VERSION"
	RecordStatusDelete      RecordStatus = "DELETE"
)

// RecordStatusUpdate is the status that makes RODA update an existing AIP.
// It is not in SIP3. commons-ip and RODA read the record status against
// their own list, which has UPDATE and lacks SUPPLEMENT, VERSION and
// DELETE (ADR-0038, keeps/commons-ip#389).
const RecordStatusUpdate RecordStatus = "UPDATE"

// ParseRecordStatus returns the record status that text names, in any
// case. It returns an error if text names none of the values the tool
// knows.
func ParseRecordStatus(text string) (RecordStatus, error) {
	s := RecordStatus(strings.ToUpper(text))
	if !s.IsValid() {
		return "", fmt.Errorf("record status %q is not one of NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE (SIP3) and UPDATE", text)
	}
	return s, nil
}

// IsValid reports whether s is one of the values the tool knows, in the
// same case. Case matters because the METS template writes the value as it
// is.
func (s RecordStatus) IsValid() bool {
	switch s {
	case RecordStatusNew, RecordStatusSupplement, RecordStatusReplacement,
		RecordStatusTest, RecordStatusVersion, RecordStatusDelete,
		RecordStatusUpdate:
		return true
	}
	return false
}

// IsUpdate reports whether s declares the package an update of an earlier
// one: a package that must reuse the original's identifier as its
// mets/@OBJID.
func (s RecordStatus) IsUpdate() bool {
	switch s {
	case RecordStatusSupplement, RecordStatusReplacement, RecordStatusVersion, RecordStatusDelete,
		RecordStatusUpdate:
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
	// Name is agent/name.
	Name string
	// Note is agent/note, or empty when the agent has none.
	Note string
	// NoteType is note/@csip:NOTETYPE, required when Note is set.
	NoteType string
}
