package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Eark is the eark profile's mapping: the fifteen Simple Dublin Core
// elements, the table in profiles/eark. The terms become eark.Terms
// unchanged; their Validate decides which keys exist and what a term may
// say.
type Eark struct{}

// Map takes the terms as Simple Dublin Core terms, in order: the keys are
// the elements' own names, so the mapping is the identity.
func (Eark) Map(terms []sip.Term) (sip.Description, []error) {
	return eark.Terms(terms), nil
}
