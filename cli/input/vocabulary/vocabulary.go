// Package vocabulary holds the profiles' vocabularies for the input
// folder: one input.Vocabulary per profile, giving the statements of a
// description.csv their meaning under it and building the profile's
// description from them. It is the one place in the CLI that imports the
// profile packages for descriptive metadata; Read in cli/input imports
// none and takes a vocabulary as a value.
package vocabulary

import (
	"encoding/xml"
	"fmt"

	"github.com/ugent-library/sip-creator/build"
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

// validateDocumentRoot judges root with the profile's encoder. Only a
// vocabulary whose profile's encoder takes a supplied document implements
// input.DocumentFormat, so the refusal below is a programming error, not
// the operator's.
func validateDocumentRoot(def build.Definition, root xml.StartElement) error {
	format, ok := def.Encoder.(build.DocumentFormat)
	if !ok {
		return fmt.Errorf("profile %q takes no supplied descriptive document", def.Name)
	}
	return format.ValidateDocumentRoot(root)
}

// terms turns statements into terms, for the profiles whose description is
// a list of them, in statement order, so a term error's index names the
// row.
func terms(statements []input.Statement) []sip.Term {
	out := make([]sip.Term, len(statements))
	for i, s := range statements {
		out[i] = sip.Term{Key: s.Key, Lang: s.Lang, Value: s.Value}
	}
	return out
}
