package profiles

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/encoders/dc"
	"github.com/ugent-library/sip-creator/encoders/dcschema"
	"github.com/ugent-library/sip-creator/sip"
)

// descriptive is what a profile does with descriptive metadata: which
// standard's type it accepts, whether it swaps the entity identifier in,
// and how it writes the document. It is the one behavioral choice a
// profile makes; every other difference between profiles is a value on
// Definition, and all profiles share one writer (ADR-0007, ADR-0015). The
// set is closed: registry entries pick one of the values below, and the
// Definition field holding it is unexported. Everything that needs the
// concrete terms type lives in the closures here, so the rest of the
// package speaks sip.Description only.
type descriptive struct {
	// check returns why d is not a description of this standard.
	check func(d sip.Description) error
	// encode writes a description of this standard as the profile's
	// descriptive document.
	encode descriptiveEncoder
	// swap replaces the identifier in d with id and returns the producer's
	// identifier it replaced. Set only for a standard whose spec links
	// descriptive and preservation metadata by a shared identifier
	// (meemoo's); nil means the document keeps the producer's identifier
	// (ADR-0012).
	swap func(d sip.Description, id string) (local string)
}

// descriptiveEncoder writes d as a descriptive document. schemas is the
// relative path from the document being written to the package's schemas/
// dir; only the writer knows where a document lands.
type descriptiveEncoder func(w io.Writer, d sip.Description, schemas string) error

// meemooDC is meemoo's dc+schema document: Dublin Core terms plus schema.org
// properties in the meemoo namespace, written from dcschema.Terms.
var meemooDC = descriptive{
	check: func(d sip.Description) error {
		if _, ok := d.(dcschema.Terms); !ok {
			return fmt.Errorf("descriptive metadata is %T, not meemoo dc+schema terms (dcschema.Terms)", d)
		}
		return nil
	},
	encode: func(w io.Writer, d sip.Description, schemas string) error {
		// checkInput ran before anything else, so the assertion holds.
		return dcschema.Encode(w, d.(dcschema.Terms), schemas)
	},
	// meemoo SIP 1.2 links dc+schema.xml to the PREMIS object by a shared
	// UUID: the document carries the entity identifier, and the producer's
	// own identifier travels as a MEEMOO-LOCAL-ID object identifier. The
	// terms hold one identifier slot, so the producer's value is read
	// before the swap overwrites it.
	swap: func(d sip.Description, id string) string {
		terms := d.(dcschema.Terms) // checkInput ran before anything else
		local := terms.LocalIdentifier()
		terms.SetObjectIdentifier(id)
		return local
	},
}

// simpleDC is the Simple Dublin Core document RODA renders natively, written
// from dc.Terms. It never swaps: dc.xml keeps the producer's identifier,
// because CSIP has no rule tying it to the package identifier and the
// ingesting catalogue indexes dc.xml, so operators find the package by the
// identifier they know (ADR-0012).
var simpleDC = descriptive{
	check: func(d sip.Description) error {
		if _, ok := d.(dc.Terms); !ok {
			return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (dc.Terms)", d)
		}
		return nil
	},
	encode: func(w io.Writer, d sip.Description, schemas string) error {
		// checkInput ran before anything else, so the assertion holds.
		return dc.Encode(w, d.(dc.Terms), schemas)
	},
}

// checkInput returns why a description in the input is not of this
// standard, the package's or a representation's. A missing package
// description is Input.Validate's finding, not this check's.
func (s descriptive) checkInput(in *Input) error {
	if in.Descriptive != nil {
		if err := s.check(in.Descriptive); err != nil {
			return err
		}
	}
	for _, r := range in.Representations {
		if r.Descriptive == nil {
			continue
		}
		if err := s.check(r.Descriptive); err != nil {
			return fmt.Errorf("representation %q: %w", r.Name, err)
		}
	}
	return nil
}
