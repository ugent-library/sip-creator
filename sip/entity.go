package sip

import (
	"fmt"
	"uuid"
)

// Entity is one intellectual entity: the work the package describes. A
// package has one.
type Entity struct {
	// Identifier identifies the entity (uuid-<uuid>). The package PREMIS
	// document carries it when the profile writes one. The descriptive
	// document carries it when the profile's description puts it in place of
	// the producer's identifier, as Meemoo's does. The producer's identifier
	// then becomes MEEMOO-LOCAL-ID. A profile that does neither leaves it out
	// of the package.
	Identifier string
	// AdditionalIdentifiers are extra PREMIS object identifiers, keyed by
	// type, such as MEEMOO-LOCAL-ID.
	AdditionalIdentifiers map[string]string
	// Representations are the versions of the content.
	Representations []*Representation
	// Description is the entity's descriptive metadata.
	Description Description
	// DescriptionFile is the node for the descriptive document in the
	// package.
	DescriptionFile *File
}

// EachRepresentation calls fn for every representation, in order. It stops
// at the first error fn returns and returns that error.
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
