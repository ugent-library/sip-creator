package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's vocabulary: the keys of meemoo's dc+schema
// table in profiles/meemoo. The document is a flat list, so a key is the
// model: the rows become meemoo.Terms as stated, and which keys exist and
// what a row may say are the terms' own rules, run by the reader on the
// result.
type Meemoo struct{}

// Description wraps the rows as meemoo terms in row order. A dc+schema
// document has no place for a record's copies, so item rows are a finding.
func (Meemoo) Description(rows []input.Row, items []input.ItemRow) (sip.Description, []input.Finding) {
	return meemoo.Terms(statements(rows)), noItems(items, meemoo.Definition.Name)
}
