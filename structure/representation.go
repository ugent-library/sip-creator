package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type Representation struct {
	Identifier       string
	Label            string
	Files            []*File
	DescriptiveFiles []*File
	PremisFile       *File
	MetsFile         *File
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

func NewRepresentation(label string) *Representation {
	return &Representation{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
		Label:      label,
	}
}
