package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type Representation struct {
	Entity     *Entity
	Label      string
	Identifier string
	Files      []*File
	PremisFile *File
	MetsFile   *File
}

func (r *Representation) AddFile(f *File) {
	r.Files = append(r.Files, f)
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

func NewRepresentation(label string) *Representation {
	return &Representation{
		Label:      label,
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
