package input

import (
	"github.com/ugent-library/sip-creator/sip"
)

// Mapper maps the terms of a description.csv onto the profile's
// description: which keys exist and where each term lands. The
// implementations live in cli/input/mapping, one per profile, so this
// package imports no profile.
type Mapper interface {
	// Map builds the profile's description from the terms of one level, in
	// file order. It reports each term it cannot place as a *sip.TermError
	// naming the term's index in terms; any other error concerns the file
	// as a whole. It returns a description even then, and for no terms at
	// all, so the description's own rules can still run. Where the
	// description is a list of terms it keeps their order and their
	// number, a refused term kept as written, so the index in a
	// *sip.TermError from the description's Validate names the same term;
	// Read drops the description's finding on a term Map refused.
	Map(terms []sip.Term) (sip.Description, []error)
}
