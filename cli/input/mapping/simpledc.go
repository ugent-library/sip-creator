package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

// SimpleDC is the eark/dc profile's mapping: the fifteen Simple Dublin Core
// elements, the table in profiles/simpledc. The terms become simpledc.Terms
// unchanged; their Validate decides which keys exist and what a term may
// say.
type SimpleDC struct{}

// Map takes the terms as Simple Dublin Core terms, in order: the keys are
// the elements' own names, so the mapping is the identity.
func (SimpleDC) Map(terms []sip.Term) (sip.Description, []error) {
	return simpledc.Terms(terms), nil
}
