package vocabulary

import (
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's vocabulary: the keys of Meemoo's dc+schema
// table in profiles/meemoo. The terms become meemoo.Terms unchanged; their
// Validate decides which keys exist and what a term may say.
type Meemoo struct{}

// Description takes the terms as Meemoo terms, in order.
func (Meemoo) Description(terms []sip.Term) (sip.Description, []error) {
	return meemoo.Terms(terms), nil
}
