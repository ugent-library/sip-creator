package sip

import (
	"fmt"
	"uuid"
)

// Representation is one version of the content, such as a master or an
// access copy.
type Representation struct {
	// Entity is the intellectual entity this representation represents.
	Entity *Entity
	// Name is the representation's name in the package: the directory under
	// representations/, the representation METS's mets/@OBJID, and the
	// fileSec/structMap paths. It comes from the name the producer gave the
	// representation.
	Name string
	// Label is the producer's human-readable name for this version, written
	// as mets/@LABEL in the representation METS. It equals Name when the
	// producer gives no label.
	Label string
	// Identifier identifies the representation in METS and PREMIS
	// (uuid-<uuid>).
	Identifier string
	// Files are the essence files, in packaging order.
	Files []*File
	// Description describes this version of the content only, such as a
	// license that differs between master and access copy. It is nil when
	// the version has no description of its own. The work's identity stays
	// on the Entity.
	Description Description
	// DescriptionFile is the node for the descriptive document in the
	// package. It is set only when Description is set.
	DescriptionFile *File
	// PremisFile is the generated PREMIS document. It is set only when the
	// profile writes one.
	PremisFile *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input, such as PREMIS from a vendor or a digitization lab. They are
	// copied into the package as received, never parsed or merged.
	ReceivedPremisFiles []*File
	// DocumentationFiles document this representation only.
	DocumentationFiles []*File
	// MetsFile is the node for the generated representation METS.
	MetsFile *File
	// Declaration holds the profile-level values the representation METS
	// declares.
	Declaration *MetsDeclaration
}

// PremisFiles returns every preservation document the representation METS
// references: the generated PREMIS document first, if there is one, then
// the received ones.
func (r *Representation) PremisFiles() []*File {
	var files []*File
	if r.PremisFile != nil {
		files = append(files, r.PremisFile)
	}
	return append(files, r.ReceivedPremisFiles...)
}

// NewRepresentation mints a Representation named name, with a fresh
// uuid-<uuid> identifier.
func NewRepresentation(name string) *Representation {
	return &Representation{
		Name:       name,
		Identifier: fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
	}
}
