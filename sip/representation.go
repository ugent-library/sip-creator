package sip

import (
	"fmt"

	"github.com/google/uuid"
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
	Description     Descriptive
	DescriptionFile *File
	PremisFile      *File
	MetsFile        *File
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

func (r *Representation) AddMetsFile(f *File) {
	r.MetsFile = f
}

func (r *Representation) SetEntity(e *Entity) {
	r.Entity = e
}

func (r *Representation) GetEntity() *Entity {
	return r.Entity
}

func NewRepresentation(name string) *Representation {
	return &Representation{
		Name:       name,
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
