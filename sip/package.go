package sip

import (
	"fmt"
	"path/filepath"

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
	// input. They are copied as received, never parsed (input spec §5).
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
// reference: the generated PREMIS (when emitted) first, then the
// received ones. Each gets one digiprovMD, all in one amdSec.
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

// DescriptiveFiles lists every descriptive document the package METS must
// reference: sub-entities first, then the root. Sub-entities never nest
// deeper than one level today (nothing assembles them yet); revisit this
// when they do.
func (p *Package) DescriptiveFiles() []*File {
	var files []*File
	for _, e := range p.Root.Entities {
		files = append(files, e.DescriptionFile)
	}
	return append(files, p.Root.DescriptionFile)
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
		Location:   filepath.Join(baseDir, identifier),
	}
}
