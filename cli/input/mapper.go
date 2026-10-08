package input

import (
	"github.com/ugent-library/sip-creator/sip"
)

// Mapper maps the terms of a description.csv onto the profile's
// description: which keys exist and where each term lands. It is an
// interface so that this package imports no profile.
type Mapper interface {
	// Map builds the profile's description from the terms of one level, in
	// file order. It reports each term it cannot place as a *sip.TermError
	// with the term's index in terms. Any other error concerns the file as
	// a whole. Map returns a description even when it reports errors, and
	// when terms is empty, so the description's own rules can still run.
	// When the description is a list of terms, Map keeps their order and
	// their number, and keeps a refused term as written. The index in a
	// *sip.TermError from the description's Validate then names the same
	// term.
	Map(terms []sip.Term) (sip.Description, []error)
}
