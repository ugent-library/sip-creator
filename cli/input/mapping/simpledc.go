package mapping

import (
	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

// SimpleDC is the ugent/basic profile's mapping. Its keys are the fifteen
// Simple Dublin Core elements, which profiles/ugent lists.
// ugent.Terms.Validate decides which keys exist and what a term may say.
type SimpleDC struct{}

// Map returns the terms unchanged and in order as ugent.Terms. The keys
// are the elements' own names, so there is nothing to rename.
func (SimpleDC) Map(terms []sip.Term) (sip.Description, []error) {
	return ugent.Terms(terms), nil
}
