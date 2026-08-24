package sip

import (
	"fmt"
	"path/filepath"
	"uuid"
)

// Package is one assembled SIP: the graph of entities, representations,
// and file nodes the writer emits.
type Package struct {
	// Location is the package directory on disk: the destination dir
	// joined with the identifier.
	Location string
	// Identifier is the package identifier (uuid-<uuid>): the mets/@OBJID,
	// the directory name, and the zip name.
	Identifier string
	// Declaration carries the profile-level METS values; set by the assembler.
	Declaration *MetsDeclaration
	// Root is the intellectual entity the package describes.
	Root *Entity
	// PremisFile is the generated preservation document; nil when the
	// profile emits none.
	PremisFile *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input. They are copied as received, never parsed.
	ReceivedPremisFiles []*File
	// MetsFile is the node for the generated package METS.
	MetsFile *File
	// SchemaFiles are the bundled XSDs, copied into schemas/.
	SchemaFiles []*File
	// DocumentationFiles document the whole package.
	DocumentationFiles []*File
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
		identifier = fmt.Sprintf("uuid-%s", uuid.NewV4().String())
	}
	return &Package{
		Identifier: identifier,
		Location:   filepath.Join(baseDir, identifier),
	}
}
