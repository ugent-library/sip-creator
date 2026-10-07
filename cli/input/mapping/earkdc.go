package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/earkdc"
	"github.com/ugent-library/sip-creator/sip"
)

// EarkDC is the eark/dc profile's mapping: the fifteen Simple Dublin Core
// elements, the table in profiles/earkdc. The terms become earkdc.Terms
// unchanged; their Validate decides which keys exist and what a term may
// say.
type EarkDC struct{}

// Map takes the terms as Simple Dublin Core terms, in order: the keys are
// the elements' own names, so the mapping is the identity.
func (EarkDC) Map(terms []sip.Term) (sip.Description, []error) {
	return earkdc.Terms(terms), nil
}
