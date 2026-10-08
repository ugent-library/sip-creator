package sip

import (
	"fmt"
	"path/filepath"
	"uuid"
)

// Package is one assembled SIP: the graph of its entity, representations
// and file nodes.
type Package struct {
	// Location is the package directory on disk: the destination directory
	// joined with the identifier.
	Location string
	// Identifier is the package identifier (uuid-<uuid>): the mets/@OBJID,
	// the directory name, and the zip name.
	Identifier string
	// Declaration holds the profile-level values the package METS declares.
	Declaration *MetsDeclaration
	// Root is the intellectual entity the package describes.
	Root *Entity
	// PremisFile is the generated PREMIS document. It is set only when the
	// profile writes one.
	PremisFile *File
	// ReceivedPremisFiles are preservation documents delivered with the
	// input. They are copied as received, never parsed.
	ReceivedPremisFiles []*File
	// MetsFile is the node for the generated package METS.
	MetsFile *File
	// SchemaFiles are the bundled XSDs this package ships, copied into
	// schemas/.
	SchemaFiles []*File
	// DocumentationFiles document the whole package.
	DocumentationFiles []*File
}

// PremisFiles returns every preservation document the package METS
// references: the generated PREMIS document first, if there is one, then
// the received ones.
func (p *Package) PremisFiles() []*File {
	var files []*File
	if p.PremisFile != nil {
		files = append(files, p.PremisFile)
	}
	return append(files, p.ReceivedPremisFiles...)
}

// DescriptiveFiles returns every descriptive document the package METS
// references. A package has one entity, so that is the root entity's
// document alone.
func (p *Package) DescriptiveFiles() []*File {
	return []*File{p.Root.DescriptionFile}
}

// NewPackage returns a package whose directory is baseDir joined with its
// identifier. It reuses identifier when one is given, so that an update
// keeps the original's mets/@OBJID. It mints a fresh one when identifier is
// empty.
func NewPackage(baseDir, identifier string) *Package {
	if identifier == "" {
		identifier = fmt.Sprintf("uuid-%s", uuid.NewV4().String())
	}
	return &Package{
		Identifier: identifier,
		Location:   filepath.Join(baseDir, identifier),
	}
}
