package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type File struct {
	Identifier string
	Name       string
	Checksum   string
	Format     string
	Size       string
	Created    string
}

func NewFile() *File {
	return &File{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
