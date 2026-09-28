// Package profiles is the registry of the profiles a package can be built
// to. Each profile lives in its own package under profiles/, which holds
// everything the profile knows about its descriptive metadata (the terms
// type, its vocabulary, its rules, its template, the XSDs its document
// points at) next to the build.Definition naming the rest as data. This
// package only hands definitions out by name; the set is closed here.
package profiles

import (
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
)

var registry = map[string]build.Definition{
	"basic": meemoo.Definition,
	"eark":  eark.Definition,
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
