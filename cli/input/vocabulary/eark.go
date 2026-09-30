package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Eark is the eark profile's vocabulary: the fifteen Simple Dublin Core
// elements, the table in profiles/eark. The document is a flat list, so a
// key is the model: the rows become eark.Terms as stated, and which keys
// exist and what a row may say are the terms' own rules, run by the reader
// on the result.
type Eark struct{}

// Description wraps the rows as Simple Dublin Core terms in row order. A
// Simple DC document has no place for a record's copies, so item rows are
// a finding.
func (Eark) Description(rows []input.Row, items []input.ItemRow) (sip.Description, []input.Finding) {
	return eark.Terms(statements(rows)), noItems(items, eark.Definition.Name)
}
