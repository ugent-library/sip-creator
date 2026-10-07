// Package profiles is the registry of the profiles a package can be built
// to. Each profile lives in its own package under profiles/, which holds
// everything the profile knows about its descriptive metadata (its
// description type and rules, its template, the XSDs its document points
// at, and for Meemoo and eark the table of keys) next to the
// build.Definition naming the rest as data. This
// package only hands definitions out by name; the set is closed here.
package profiles

import (
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/earkdc"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
)

var registry = map[string]build.Definition{
	"meemoo/basic": meemoo.Definition,
	"eark/dc":      earkdc.Definition,
	"eark/mods":    earkmods.Definition,
}

// Get resolves a profile name to its definition.
func Get(name string) (build.Definition, bool) {
	def, ok := registry[name]
	return def, ok
}

// Names lists the registered profiles, sorted, for CLI error messages.
func Names() []string {
	return slices.Sorted(maps.Keys(registry))
}
