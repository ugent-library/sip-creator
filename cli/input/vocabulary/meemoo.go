package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's vocabulary: the keys of meemoo's dc+schema
// table in profiles/meemoo. The document is a flat list, so a key is the
// model: the statements become meemoo.Terms as stated, and which keys
// exist and what a statement may say are the terms' own rules, run by the
// reader on the result.
type Meemoo struct{}

// Description wraps the statements as meemoo terms in statement order.
func (Meemoo) Description(statements []input.Statement) (sip.Description, []error) {
	return meemoo.Terms(terms(statements)), nil
}
