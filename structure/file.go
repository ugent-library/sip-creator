package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type File struct {
	Identifier     string
	Name           string
	Checksum       string
	Format         string
	Size           string
	Created        string
	Path           string
	Representation *Representation
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
