package sip

import (
	"fmt"
	"uuid"
)

type File struct {
	Identifier string
	Name       string
	Checksum   string
	Format     *Format
	Size       string
	Created    string
	Source     string
	Path       string
	// Mime is the IANA media type METS declares for this file (@MIMETYPE is
	// a MUST: CSIP62/26/40). Never empty by write time, and never a guess:
	// a characterizer's assertion, a type true by construction (generated
	// XML), or application/octet-stream as the admitted unknown.
	Mime           string
	Representation *Representation
}

type Format struct {
	FormatRegistry *FormatRegistry
}

type FormatRegistry struct {
	Name string
	Key  string
	Role string
}

func NewFormatRegistry() *FormatRegistry {
	return &FormatRegistry{
		Role: "specification",
	}
}

func (f *File) SetRepresentation(r *Representation) {
	f.Representation = r
}

func NewFile() *File {
	return &File{
		Identifier: fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
	}
}
