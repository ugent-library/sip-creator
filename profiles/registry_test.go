package profiles

import (
	"maps"
	"slices"
	"testing"

	"github.com/ugent-library/sip-creator/schemas"
)

// Every registry entry's schema list names bundled files, so a typo fails
// here rather than at the first build (a name may repeat across the lists
// a profile concatenates; the assembler ships it once). Both profiles ship
// the whole bundle today; a profile shipping a subset gets its own line.
func TestRegistrySchemas(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Names() {
		def, _ := Get(name)
		for _, xsd := range def.Schemas {
			if _, ok := bundle[xsd]; !ok {
				t.Errorf("profile %q ships %q, which is not bundled", name, xsd)
			}
		}
	}

	all := slices.Sorted(maps.Keys(bundle))
	for _, name := range []string{"basic", "eark"} {
		def, _ := Get(name)
		if got := slices.Compact(slices.Sorted(slices.Values(def.Schemas))); !slices.Equal(got, all) {
			t.Errorf("profile %q Schemas = %v, want the whole bundle %v", name, got, all)
		}
	}
}

// Every registry entry names a descriptive standard; the engine refuses a
// definition without one before any write.
func TestRegistryEntriesNameAStandard(t *testing.T) {
	for _, name := range Names() {
		def, _ := Get(name)
		if def.Descriptive == nil {
			t.Errorf("profile %q has no descriptive standard", name)
		}
		if def.Name != name {
			t.Errorf("profile %q is registered under Name %q", name, def.Name)
		}
	}
}
