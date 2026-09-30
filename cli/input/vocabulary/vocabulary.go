// Package vocabulary holds the profiles' vocabularies for the input
// folder: one input.Vocabulary per profile, giving the rows of a
// description.csv their meaning under it and building the profile's
// description from them. It is the one place in the CLI that imports the
// profile packages for descriptive metadata; the reader in cli/input
// imports none and takes a vocabulary as a value.
package vocabulary

import (
	"fmt"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// byProfile pairs each profile, by the name the registry in profiles/
// hands it out under, with its vocabulary. Keyed by the definitions' own
// names so a renamed profile cannot leave a stale entry behind.
var byProfile = map[string]input.Vocabulary{
	meemoo.Definition.Name:   Meemoo{},
	eark.Definition.Name:     Eark{},
	earkmods.Definition.Name: EarkMods{},
}

// For returns the vocabulary of the profile named name and whether one
// exists. Every registered profile has one; a test pins that.
func For(name string) (input.Vocabulary, bool) {
	v, ok := byProfile[name]
	return v, ok
}

// statements turns rows into the statements a flat world's description is
// a list of, in row order, so a term error's index names the row.
func statements(rows []input.Row) []sip.Term {
	terms := make([]sip.Term, len(rows))
	for i, r := range rows {
		terms[i] = sip.Term{Key: r.Key, Lang: r.Lang, Value: r.Value}
	}
	return terms
}

// noItems reports item rows handed to a profile whose document has no
// place for a record's copies: one finding about the file, none when
// there are no rows.
func noItems(items []input.ItemRow, profile string) []input.Finding {
	if len(items) == 0 {
		return nil
	}
	return []input.Finding{{Err: fmt.Errorf("items.csv: the %s profile's document has no place for a record's copies; remove the file", profile)}}
}
