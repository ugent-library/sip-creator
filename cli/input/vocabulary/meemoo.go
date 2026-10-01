package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's vocabulary: the keys of Meemoo's dc+schema
// table in profiles/meemoo. Each statement becomes one term of
// meemoo.Terms unchanged; the terms' Validate, which the reader runs on
// the result, decides which keys exist and what a statement may say.
type Meemoo struct{}

// Description wraps the statements as Meemoo terms in statement order.
func (Meemoo) Description(statements []input.Statement) (sip.Description, []error) {
	return meemoo.Terms(terms(statements)), nil
}
