package structure

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type Entity interface {
	AddDescription(bts []byte)
	AddRepresentation(r *Representation)
	AddSubIntellectualEntity(e *Entity)
	EachRepresentation(fn func(r *Representation) error) error
}

type DublinCoreEntity struct {
	Identifier      string
	Label           string
	Representations []*Representation
	Entities        []*Entity
	Description     *DublinCore
}

func (e *DublinCoreEntity) AddDescription(bts []byte) {
	var dc *DublinCore
	if err := json.Unmarshal(bts, &dc); err != nil {
		panic(err)
	}

	e.Description = dc
}

func (e *DublinCoreEntity) AddSubIntellectualEntity(sub *Entity) {
	e.Entities = append(e.Entities, sub)
}

func (e *DublinCoreEntity) AddRepresentation(r *Representation) {
	e.Representations = append(e.Representations, r)
}

func (e *DublinCoreEntity) EachRepresentation(fn func(r *Representation) error) error {
	for _, r := range e.Representations {
		err := fn(r)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewDublinCoreEntity(label string) *DublinCoreEntity {
	return &DublinCoreEntity{
		Identifier: fmt.Sprintf("uuid-%s", uuid.New().String()),
		Label:      label,
	}
}
