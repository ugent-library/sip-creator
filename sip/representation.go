package sip

import (
	"fmt"
	"uuid"

	"github.com/ugent-library/sip-creator/encoders/metadata"
)

// Representation is one version of the content, e.g. a master or an
// access copy.
type Representation struct {
	// Entity is the intellectual entity this representation represents.
	Entity *Entity
	// Name is the package-side name: the directory under representations/,
	// the representation METS's OBJID, and the fileSec/structMap paths.
	// The assembler sets it from the producer's label.
	Name string
	// Label is the producer's human-readable name for this version,
	// emitted as the representation METS's mets/@LABEL.
	Label string
	// Identifier identifies the representation in METS and PREMIS
	// (uuid-<uuid>).
	Identifier string
	// Files are the essence files, in packaging order.
	Files []*File
	// Description optionally describes this version of the content only
	// (e.g. a license that differs between master and access copy);
	// the work's identity stays on the Entity.
	Description metadata.Terms
	// DescriptionFile is the node for the generated descriptive document;
	// nil when Description is nil.
	DescriptionFile *File
	// PremisFile is the generated preservation document; nil when the
	// profile emits none.
	PremisFile *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input (vendor/lab PREMIS). They are copied into the package as
	// received, never parsed or merged.
	ReceivedPremisFiles []*File
	// DocumentationFiles document this representation only.
	DocumentationFiles []*File
	// MetsFile is the node for the generated representation METS.
	MetsFile *File
	// Declaration carries the METS values the representation METS declares;
	// set by the assembler.
	Declaration *MetsDeclaration
}

func (r *Representation) AddFile(f *File) {
	r.Files = append(r.Files, f)
}

func (r *Representation) SetDescriptionFile(f *File) {
	r.DescriptionFile = f
}

func (r *Representation) SetPremisFile(f *File) {
	r.PremisFile = f
}

func (r *Representation) SetReceivedPremisFiles(files []*File) {
	r.ReceivedPremisFiles = files
}

func (r *Representation) SetDocumentationFiles(files []*File) {
	r.DocumentationFiles = files
}

// PremisFiles lists every preservation document the representation METS
// must reference: the generated PREMIS (when emitted) first, then the
// received ones. Each gets one digiprovMD, all in one amdSec.
func (r *Representation) PremisFiles() []*File {
	var files []*File
	if r.PremisFile != nil {
		files = append(files, r.PremisFile)
	}
	return append(files, r.ReceivedPremisFiles...)
}

func (r *Representation) SetMetsFile(f *File) {
	r.MetsFile = f
}

func (r *Representation) SetEntity(e *Entity) {
	r.Entity = e
}

// NewRepresentation mints a Representation named name, with a fresh
// uuid-<uuid> identifier.
func NewRepresentation(name string) *Representation {
	return &Representation{
		Name:       name,
		Identifier: fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
	}
}
