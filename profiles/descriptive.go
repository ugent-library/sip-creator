package profiles

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/encoders/dc"
	"github.com/ugent-library/sip-creator/sip"
)

// descriptive is what a profile does with descriptive metadata: which
// standard's type it accepts and how it writes the document. It is the one
// behavioral choice a profile makes; every other difference between
// profiles is a value on Definition, and all profiles share one writer
// (ADR-0007, ADR-0015). The set is closed: registry entries pick one of the
// values below, and the Definition field holding it is unexported.
type descriptive struct {
	// check returns why d is not a description of this standard.
	check func(d sip.Description) error
	// encode writes a description of this standard as the profile's
	// descriptive document.
	encode descriptiveEncoder
}

// descriptiveEncoder writes d as a descriptive document. schemas is the
// relative path from the document being written to the package's schemas/
// dir; only the writer knows where a document lands.
type descriptiveEncoder func(w io.Writer, d sip.Description, schemas string) error

// The two documents the registered profiles write from dc.Terms: meemoo's
// dc+schema shape (Dublin Core terms plus schema.org elements in the meemoo
// namespace) and the simple DC shape RODA renders natively.
var (
	meemooDC = fromDCTerms(dc.EncodeTerms)
	simpleDC = fromDCTerms(dc.EncodeDCTerms)
)

// fromDCTerms is a standard whose descriptions must be the dc package's
// terms model, written out by encode. The name says what the input must
// be, not what the document holds: the two documents above differ in
// exactly that.
func fromDCTerms(encode func(io.Writer, dc.Terms, string) error) descriptive {
	return descriptive{
		check: func(d sip.Description) error {
			if _, ok := d.(dc.Terms); !ok {
				return fmt.Errorf("descriptive metadata is %T, not Dublin Core terms (dc.Terms)", d)
			}
			return nil
		},
		encode: func(w io.Writer, d sip.Description, schemas string) error {
			// checkInput ran before anything else, so the assertion holds.
			return encode(w, d.(dc.Terms), schemas)
		},
	}
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
