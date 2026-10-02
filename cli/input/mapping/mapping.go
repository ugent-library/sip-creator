// Package mapping holds the profiles' mappings for the input folder: one
// input.Mapper per profile, mapping the terms of a description.csv onto
// the profile's description. It is the one place in the CLI that imports
// the profile packages for descriptive metadata; Read in cli/input imports
// none and takes a mapper as a value.
package mapping

import (
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
)

// byProfile pairs each profile, by the name the registry in profiles/
// hands it out under, with its mapper. Keyed by the definitions' own
// names so a renamed profile cannot leave a stale entry behind.
var byProfile = map[string]input.Mapper{
	meemoo.Definition.Name:   Meemoo{},
	eark.Definition.Name:     Eark{},
	earkmods.Definition.Name: EarkMods{},
}

// For returns the mapper of the profile named name and whether one
// exists. Every registered profile has one; a test pins that.
func For(name string) (input.Mapper, bool) {
	v, ok := byProfile[name]
	return v, ok
}
