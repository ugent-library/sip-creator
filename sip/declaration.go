package sip

import (
	"fmt"
	"strings"
)

// MetsDeclaration holds the profile-level values a METS document declares:
// the profile URL, the content typing, and the agents responsible for the
// package. Profiles differ in these values, not in build logic, so a profile
// difference in the METS reaches the templates as data here.
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
	// RecordStatus is metsHdr/@RECORDSTATUS (SIP3). Only the package METS
	// carries it, and only when it is set, because the E-ARK SIP
	// specification reads an absent status as NEW.
	RecordStatus RecordStatus
	// Agents are the metsHdr agent entries.
	Agents []Agent
}

// RecordStatus is metsHdr/@RECORDSTATUS in the E-ARK SIP vocabulary (SIP3):
// what the package does to the archive's holdings. The vocabulary is
// closed.
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

// ParseRecordStatus returns the vocabulary value that text names, in any
// case. It returns an error if text names none.
func ParseRecordStatus(text string) (RecordStatus, error) {
	s := RecordStatus(strings.ToUpper(text))
	if !s.IsValid() {
		return "", fmt.Errorf("record status %q is not in the SIP3 vocabulary (NEW, SUPPLEMENT, REPLACEMENT, TEST, VERSION, DELETE)", text)
	}
	return s, nil
}

// IsValid reports whether s is one of the vocabulary's values, in the same
// case. Case matters because the METS template writes the value as it is.
func (s RecordStatus) IsValid() bool {
	switch s {
	case RecordStatusNew, RecordStatusSupplement, RecordStatusReplacement,
		RecordStatusTest, RecordStatusVersion, RecordStatusDelete:
		return true
	}
	return false
}

// IsUpdate reports whether s declares the package an update of an earlier
// one: a package that must reuse the original's identifier as its
// mets/@OBJID.
func (s RecordStatus) IsUpdate() bool {
	switch s {
	case RecordStatusSupplement, RecordStatusReplacement, RecordStatusVersion, RecordStatusDelete:
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
