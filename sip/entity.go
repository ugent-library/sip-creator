package sip

import (
	"fmt"
	"uuid"
)

// Entity is one intellectual entity: the work the package describes. A
// package has one; sub-entities are not modeled.
type Entity struct {
	// Identifier identifies the entity in PREMIS and the emitted
	// descriptive document (uuid-<uuid>).
	Identifier string
	// AdditionalIdentifiers are extra PREMIS object identifiers, keyed by
	// type, e.g. MEEMOO-LOCAL-ID.
	AdditionalIdentifiers map[string]string
	// Representations are the versions of the content.
	Representations []*Representation
	// Description carries the decoded descriptive metadata until the writer
	// serializes it.
	Description Description
	// DescriptionFile is the node for the generated descriptive document.
	DescriptionFile *File
}

// EachRepresentation calls fn for every representation in order; the
// first error stops the walk and is returned.
func (e *Entity) EachRepresentation(fn func(r *Representation) error) error {
	for _, r := range e.Representations {
		err := fn(r)
		if err != nil {
			return err
		}
	}

	return nil
}

// NewEntity mints an Entity with a fresh uuid-<uuid> identifier.
func NewEntity() *Entity {
	return &Entity{
		Identifier:            fmt.Sprintf("uuid-%s", uuid.NewV4().String()),
		AdditionalIdentifiers: make(map[string]string),
	}
}
