package build

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/sip"
)

// DescriptiveStandard is what a profile plugs into the engine for its
// descriptive metadata: which terms type it accepts and how it writes the
// document. Each profile package under profiles/ implements it for its own
// terms type, so everything that needs the concrete type stays there and
// the engine speaks sip.Description only. The registry in profiles/ is the
// closed set of standards a build can use.
type DescriptiveStandard interface {
	// Check returns why d is not a description of this standard: a
	// description of another type. It runs before validation and before
	// any write, and guarantees the type assertions the standard's other
	// methods make.
	Check(d sip.Description) error
	// Encode writes d as the profile's descriptive document. schemas is
	// the relative path from the document being written to the package's
	// schemas/ dir; only the writer knows where a document lands.
	Encode(w io.Writer, d sip.Description, schemas string) error
}

// IdentifierSwapper is the optional part of a DescriptiveStandard whose
// spec links descriptive and preservation metadata by a shared identifier
// (meemoo's). Swap replaces the identifier in d with id and returns the
// producer's identifier it replaced; the assembler records that as the
// entity's MEEMOO-LOCAL-ID. A standard without it keeps the producer's
// identifier in the document (ADR-0012).
type IdentifierSwapper interface {
	Swap(d sip.Description, id string) (local string)
}

// descriptiveEncoder writes d as a descriptive document; the writer takes
// the standard's Encode as one.
type descriptiveEncoder func(w io.Writer, d sip.Description, schemas string) error

// checkDescriptions returns why a description in the input is not of the
// standard s, the package's or a representation's. A missing package
// description is Input.Validate's finding, not this check's.
func checkDescriptions(s DescriptiveStandard, in *Input) error {
	if in.Descriptive != nil {
		if err := s.Check(in.Descriptive); err != nil {
			return err
		}
	}
	for _, r := range in.Representations {
		if r.Descriptive == nil {
			continue
		}
		if err := s.Check(r.Descriptive); err != nil {
			return fmt.Errorf("representation %q: %w", r.Name, err)
		}
	}
	return nil
}
