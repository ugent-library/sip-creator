package structure

import (
	"fmt"

	"github.com/google/uuid"
)

type Entity struct {
	Identifier      string
	Representations []*Representation
	Entities        []*Entity
	DescriptionFile *File
}

func (e *Entity) AddDescriptionFile(f *File) {
	e.DescriptionFile = f
}

func (e *Entity) AddSubIntellectualEntity(sub *Entity) {
	e.Entities = append(e.Entities, sub)
}

func (e *Entity) AddRepresentation(r *Representation) {
	e.Representations = append(e.Representations, r)
}

func (e *Entity) EachRepresentation(fn func(r *Representation) error) error {
	for _, r := range e.Representations {
		err := fn(r)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewEntity() *Entity {
	return &Entity{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}
