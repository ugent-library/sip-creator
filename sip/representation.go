package sip

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/ugent-library/sip-creator/encoders/metadata"
)

type Representation struct {
	Entity *Entity
	// Name is the package-side name: the directory under representations/,
	// the representation METS's OBJID, and the fileSec/structMap paths.
	// The profile decides it (meemoo: representation_N).
	Name string
	// Label is the producer's human-readable name for this version
	// (input spec §2: "keeps your label as the human-readable name"),
	// emitted as the representation METS's mets/@LABEL.
	Label      string
	Identifier string
	Files      []*File
	// Description optionally describes this version of the content only
	// (e.g. a license that differs between master and access copy);
	// the work's identity stays on the Entity.
	Description     metadata.Terms
	DescriptionFile *File
	PremisFile      *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input (vendor/lab PREMIS). They are copied into the package as
	// received, never parsed or merged (input spec §5).
	ReceivedPremisFiles []*File
	// DocumentationFiles document this representation only (input spec §4).
	DocumentationFiles []*File
	MetsFile           *File
}

func (r *Representation) AddFile(f *File) {
	r.Files = append(r.Files, f)
}

func (r *Representation) AddDescriptionFile(f *File) {
	r.DescriptionFile = f
}

func (r *Representation) AddPremisFile(f *File) {
	r.PremisFile = f
}

func (r *Representation) AddReceivedPremisFiles(files []*File) {
	r.ReceivedPremisFiles = files
}

func (r *Representation) AddDocumentationFiles(files []*File) {
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

func (r *Representation) AddMetsFile(f *File) {
	r.MetsFile = f
}

func (r *Representation) SetEntity(e *Entity) {
	r.Entity = e
}

func NewRepresentation(name string) *Representation {
	return &Representation{
		Name:       name,
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
