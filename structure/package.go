package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type Package struct {
	Identifier              string
	Root                    *Entity
	PremisFile              *File
	MetsFile                *File
	DescriptiveFiles        []*File
	RepresentationMetsFiles []*File
}

func (p *Package) AddRootEntity(e *Entity) {
	p.Root = e
}

func (p *Package) AddPremisFile(f *File) {
	p.PremisFile = f
}

func (p *Package) AddMetsFile(f *File) {
	p.MetsFile = f
}

func (p *Package) GetDescriptiveFiles() []*File {
	var tmp []*File
	var fn func(e *Entity)

	fn = func(e *Entity) {
		for _, v := range e.Entities {
			fn(v)
		}

		tmp = append(tmp, e.DescriptionFile)
	}

	fn(p.Root)

	return tmp
}

func NewPackage() *Package {
	return &Package{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
