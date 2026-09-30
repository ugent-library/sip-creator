package vocabulary

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/sip"
)

// EarkMods is the eark-mods profile's vocabulary: the MODS keys. The
// record is still a list of statements plus items, so for now the rows
// become the record's terms as stated, and which keys exist and what a row
// may say are the record's own rules, run by the reader on the result. The
// typed record and the key table that maps rows onto its fields follow
// (descriptive-model plan, S3).
type EarkMods struct{}

// Description wraps the rows as a record's terms in row order. Item rows
// are a finding until the record takes them (S3).
func (EarkMods) Description(rows []input.Row, items []input.ItemRow) (sip.Description, []input.Finding) {
	return earkmods.Record{Terms: statements(rows)}, noItems(items, earkmods.Definition.Name)
}
