package sip

import (
	"fmt"
	"uuid"
)

// File is one file in the package: essence, documentation, a schema, or a
// metadata document.
type File struct {
	// Identifier identifies the file in METS and PREMIS (uuid-<uuid>).
	Identifier string
	// Name is the file's basename, emitted as premis:originalName.
	Name string
	// Checksum, Size, and Created describe the file as written into the
	// package; the writer back-fills them as the file lands.
	Checksum string
	Size     string
	Created  string
	// Format is the file's characterization result; nil when no report
	// was supplied or the report found no match.
	Format *Format
	// Source is the absolute path the file is copied from; empty for
	// generated documents (METS, PREMIS, descriptive).
	Source string
	// Path is the href relative to the METS document that references the
	// file: package-relative for package-level files, representation-relative
	// for files inside a representation.
	Path string
	// Mime is the IANA media type METS declares for this file (@MIMETYPE is
	// a MUST: CSIP62/26/40). Never empty by write time, and never a guess:
	// the characterization report's assertion, the known type of a generated
	// document, or application/octet-stream when the type is unknown.
	Mime string
	// Representation is the owning representation; nil for package-level
	// files.
	Representation *Representation
}

// Format is a file's premis:format assertion, taken from the
// characterization report.
type Format struct {
	FormatRegistry *FormatRegistry
}

// FormatRegistry identifies a format by its entry in a registry,
// e.g. PRONOM.
type FormatRegistry struct {
	// Name is the registry name, e.g. PRONOM.
	Name string
	// Key is the format's key in the registry, e.g. fmt/43.
	Key string
	// Role is the formatRegistryRole vocabulary value, normally
	// "specification".
	Role string
}

// NewFormatRegistry returns a registry entry with the role defaulted to
// "specification".
func NewFormatRegistry() *FormatRegistry {
	return &FormatRegistry{
		Role: "specification",
	}
}

func (f *File) SetRepresentation(r *Representation) {
	f.Representation = r
}

// NewFile mints a File with a fresh uuid-<uuid> identifier.
func NewFile() *File {
	return &File{
		Identifier: fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
	}
}
