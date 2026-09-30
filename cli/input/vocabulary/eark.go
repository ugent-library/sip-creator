package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Eark is the eark profile's vocabulary: the fifteen Simple Dublin Core
// elements, the table in profiles/eark. The document is a flat list, so a
// key is the model: the statements become eark.Terms as stated, and which
// keys exist and what a statement may say are the terms' own rules, run
// by the reader on the result.
type Eark struct{}

// Description wraps the statements as Simple Dublin Core terms in
// statement order.
func (Eark) Description(statements []input.Statement) (sip.Description, []error) {
	return eark.Terms(terms(statements)), nil
}
