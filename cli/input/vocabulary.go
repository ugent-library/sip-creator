package input

import (
	"github.com/ugent-library/sip-creator/sip"
)

// Vocabulary gives the terms of a description.csv their meaning under one
// profile: which keys exist and how the terms become the profile's
// description. The implementations live in cli/input/vocabulary, one per
// profile, so this package imports no profile.
type Vocabulary interface {
	// Description builds the profile's description from the terms of one
	// level, in file order. It reports each term it cannot place as a
	// *sip.TermError naming the term's index in terms; any other error
	// concerns the file as a whole. It returns a description even then,
	// and for no terms at all, so the description's own rules can still
	// run. Where the description is a list of terms it keeps their order,
	// so the index in a *sip.TermError from the description's Validate
	// names the same term.
	Description(terms []sip.Term) (sip.Description, []error)
}
