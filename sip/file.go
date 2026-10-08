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
	// Name is the file's base name. For essence, PREMIS records it as
	// premis:originalName.
	Name string
	// Checksum is the file's MD5, hex-encoded. When a characterization
	// report supplies an MD5, Checksum is that value, taken as given.
	// Otherwise it is computed from the bytes written into the package.
	Checksum string
	// Size is the file's size in bytes as written into the package.
	Size string
	// Created is the file's creation time as written into the package.
	Created string
	// Format is the file's characterization result. It is set only when a
	// characterization report was supplied and matched the file to a format.
	Format *Format
	// Source is the absolute path of the producer's file that is copied into
	// the package. It is set for essence, documentation and received PREMIS
	// files. A supplied descriptive document is copied from the path its
	// description carries, not from Source.
	Source string
	// Content is the bytes of a schema, written into the package as they
	// are. It is set only for schema files.
	Content []byte
	// Path is the file's path relative to the METS document that references
	// it. For a package-level file, it is relative to the package root. For a
	// file inside a representation, it is relative to the representation's
	// directory.
	Path string
	// Mime is the IANA media type METS declares for this file in @MIMETYPE,
	// which CSIP26, CSIP40 and CSIP62 require. It is the type the
	// characterization report asserts, the known type of a generated
	// document, or application/octet-stream when the type is unknown.
	Mime string
	// MDType is the mdRef/@MDTYPE the METS dmdSec declares for a
	// descriptive document, a value of the METS vocabulary such as DC or
	// MODS. It is set only for descriptive documents.
	MDType string
	// MDTypeVersion is the descriptive document's mdRef/@MDTYPEVERSION. It
	// is set only when the metadata model declares a version.
	MDTypeVersion string
	// OtherMDType is the descriptive document's mdRef/@OTHERMDTYPE: the name
	// of a format the MDTYPE vocabulary does not list. It is set only when
	// MDType is OTHER.
	OtherMDType string
	// Representation is the representation the file belongs to. It is set
	// only for files inside a representation.
	Representation *Representation
}

// Format is a file's premis:format assertion, taken from the
// characterization report.
type Format struct {
	// FormatRegistry is the format's entry in a format registry.
	FormatRegistry *FormatRegistry
}

// FormatRegistry identifies a format by its entry in a registry, such
// as PRONOM.
type FormatRegistry struct {
	// Name is the registry name, such as PRONOM.
	Name string
	// Key is the format's key in the registry, such as fmt/43.
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

// NewFile mints a File with a fresh uuid-<uuid> identifier.
func NewFile() *File {
	return &File{
		Identifier: fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
	}
}
