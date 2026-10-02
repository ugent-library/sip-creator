package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's mapping: the keys of Meemoo's dc+schema
// table in profiles/meemoo. The terms become meemoo.Terms unchanged; their
// Validate decides which keys exist and what a term may say.
type Meemoo struct{}

// Map takes the terms as Meemoo terms, in order.
func (Meemoo) Map(terms []sip.Term) (sip.Description, []error) {
	return meemoo.Terms(terms), nil
}
