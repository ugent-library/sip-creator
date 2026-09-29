package build

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/sip"
)

// DescriptionEncoder is what a profile plugs into the engine for its
// descriptive metadata: which description type it accepts, how to build
// one from flat statements, and how it writes the document for it. Each
// profile package under profiles/ implements it for its own terms type,
// so everything that needs the concrete type stays there and the engine
// speaks sip.Description only. The registry in profiles/ is the closed
// set of encoders a build can use.
type DescriptionEncoder interface {
	// Check returns why d is not a description this encoder takes: a
	// description of another type. It runs before validation and before
	// any write, and guarantees the type assertions the encoder's other
	// methods make.
	Check(d sip.Description) error
	// NewDescription builds this encoder's description from flat statements,
	// the type Check takes.
	// It is how a transport that decodes statements without knowing the
	// standard, such as the CLI's rows file, builds the profile's
	// description; the engine itself never calls it.
	NewDescription(terms []sip.Term) sip.Description
	// Encode writes d as the profile's descriptive document. schemas is
	// the relative path from the document being written to the package's
	// schemas/ dir; only the writer knows where a document lands.
	Encode(w io.Writer, d sip.Description, schemas string) error
	// Schemas lists the bundled XSD file names the encoded document points
	// at, plus what those import by relative path. The package ships them
	// under schemas/ next to the ones the METS documents point at, and
	// nothing else: an XSD no document references is noise to whoever
	// reads the package later.
	Schemas() []string
}

// IdentifierSwapper is the optional part of a DescriptionEncoder whose
// spec links descriptive and preservation metadata by a shared identifier
// (meemoo's). Swap replaces the identifier in d with id and returns the
// producer's identifier it replaced; the assembler records that as the
// entity's MEEMOO-LOCAL-ID. An encoder without it keeps the producer's
// identifier in the document (ADR-0012).
type IdentifierSwapper interface {
	Swap(d sip.Description, id string) (local string)
}

// checkDescriptions returns why a description in the material is not one
// the encoder takes, the package's or a representation's. A missing
// package description is Material.Validate's finding, not this check's.
func checkDescriptions(enc DescriptionEncoder, m *Material) error {
	if m.Description != nil {
		if err := enc.Check(m.Description); err != nil {
			return err
		}
	}
	for _, r := range m.Representations {
		if r.Description == nil {
			continue
		}
		if err := enc.Check(r.Description); err != nil {
			return fmt.Errorf("representation %q: %w", r.Name, err)
		}
	}
	return nil
}
