package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type File struct {
	Identifier     string
	Name           string
	Checksum       string
	Format         *Format
	Size           string
	Created        string
	Path           string
	Representation *Representation
}

type Format struct {
	FormatRegistry *FormatRegistry
	// FormatDesignation FormatDesignation
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

func (f *File) GetRepresentation() *Representation {
	return f.Representation
}

func NewFile() *File {
	return &File{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
