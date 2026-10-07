// Package profiles is the registry of the profiles a package can be built
// to. Each owner of profiles has a package under profiles/ (meemoo,
// ugent) that holds its profiles' definitions and the metadata models they
// use. This package only hands definitions out by name; the set is closed
// here.
package profiles

import (
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/ugent"
)

var registry = map[string]build.Definition{
	"meemoo/basic":        meemoo.Definition,
	"ugent/basic":         ugent.Basic,
	"ugent/bibliographic": ugent.Bibliographic,
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
