package vocabulary

import (
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Eark is the eark profile's vocabulary: the fifteen Simple Dublin Core
// elements, the table in profiles/eark. The terms become eark.Terms
// unchanged; their Validate decides which keys exist and what a term may
// say.
type Eark struct{}

// Description takes the terms as Simple Dublin Core terms, in order.
func (Eark) Description(terms []sip.Term) (sip.Description, []error) {
	return eark.Terms(terms), nil
}
