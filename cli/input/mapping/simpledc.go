package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

// SimpleDC is the ugent/basic profile's mapping: the fifteen Simple Dublin Core
// elements, the table in profiles/ugent. The terms become ugent.Terms
// unchanged; their Validate decides which keys exist and what a term may
// say.
type SimpleDC struct{}

// Map takes the terms as Simple Dublin Core terms, in order: the keys are
// the elements' own names, so the mapping is the identity.
func (SimpleDC) Map(terms []sip.Term) (sip.Description, []error) {
	return ugent.Terms(terms), nil
}
