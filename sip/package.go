package sip

import (
	"fmt"

	"github.com/google/uuid"
)

type Package struct {
	Location   string
	Identifier string
	// Spec carries the profile-level METS values; set by the assembler.
	Spec       *Spec
	Root       *Entity
	PremisFile *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input — copied as received, never parsed (input spec §5).
	ReceivedPremisFiles []*File
	MetsFile            *File
	SchemaFiles         []*File
	DocumentationFiles  []*File
}

func (p *Package) AddRootEntity(e *Entity) {
	p.Root = e
}

func (p *Package) AddPremisFile(f *File) {
	p.PremisFile = f
}

func (p *Package) AddReceivedPremisFiles(files []*File) {
	p.ReceivedPremisFiles = files
}

// PremisFiles lists every preservation document the package METS must
// reference: the generated PREMIS (when emitted) first, then the received
// ones — one digiprovMD each, in one amdSec.
func (p *Package) PremisFiles() []*File {
	var files []*File
	if p.PremisFile != nil {
		files = append(files, p.PremisFile)
	}
	return append(files, p.ReceivedPremisFiles...)
}

func (p *Package) AddMetsFile(f *File) {
	p.MetsFile = f
}

func (p *Package) AddSchemaFiles(files []*File) {
	p.SchemaFiles = files
}

func (p *Package) AddDocumentationFiles(files []*File) {
	p.DocumentationFiles = files
}

func (p *Package) DescriptiveFiles() []*File {
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

// NewPackage roots a package under baseDir. A caller-supplied identifier
// is reused as the package identifier (how an update keeps the original's
// mets/@OBJID); empty means mint a fresh one.
func NewPackage(baseDir, identifier string) *Package {
	if identifier == "" {
		identifier = fmt.Sprintf("uuid-%s", uuid.New().String())
	}
	return &Package{
		Identifier: identifier,
		Location:   fmt.Sprintf("%s/%s", baseDir, identifier),
	}
}
