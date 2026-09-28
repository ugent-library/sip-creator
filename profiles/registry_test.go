package profiles

import (
	"maps"
	"slices"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/schemas"
)

// shipped is the sorted, deduplicated set of XSDs a profile's packages ship:
// what the METS documents point at plus what its descriptive document
// points at, the same sum the assembler makes.
func shipped(t *testing.T, name string) []string {
	t.Helper()
	def, ok := Get(name)
	if !ok {
		t.Fatalf("no %q definition registered", name)
	}
	return slices.Compact(slices.Sorted(slices.Values(slices.Concat(mets.Schemas, def.Encoder.Schemas()))))
}

// Every XSD a profile ships is bundled, so a typo in an encoder's list fails
// here rather than at the first build, and each profile ships exactly what
// its documents point at: basic the whole bundle (meemoo's descriptive
// schema imports the Dublin Core family), eark the METS set plus dc.xsd.
func TestRegistrySchemas(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Names() {
		for _, xsd := range shipped(t, name) {
			if _, ok := bundle[xsd]; !ok {
				t.Errorf("profile %q ships %q, which is not bundled", name, xsd)
			}
		}
	}

	if got, all := shipped(t, "basic"), slices.Sorted(maps.Keys(bundle)); !slices.Equal(got, all) {
		t.Errorf("basic ships %v, want the whole bundle %v", got, all)
	}
	want := slices.Sorted(slices.Values(append(slices.Clone(mets.Schemas), "dc.xsd")))
	if got := shipped(t, "eark"); !slices.Equal(got, want) {
		t.Errorf("eark ships %v, want the METS set plus dc.xsd %v", got, want)
	}
}

// Every registry entry names a descriptive encoder; the engine refuses a
// definition without one before any write.
func TestRegistryEntriesNameAnEncoder(t *testing.T) {
	for _, name := range Names() {
		def, _ := Get(name)
		if def.Encoder == nil {
			t.Errorf("profile %q has no descriptive encoder", name)
		}
		if def.Name != name {
			t.Errorf("profile %q is registered under Name %q", name, def.Name)
		}
	}
}

// Every encoder's NewDescription builds what its Check accepts: the CLI hands
// decoded rows to the one, and the engine runs the other on the result.
func TestRegistryEncodersBuildWhatTheyCheck(t *testing.T) {
	for _, name := range Names() {
		def, _ := Get(name)
		if err := def.Encoder.Check(def.Encoder.NewDescription(nil)); err != nil {
			t.Errorf("profile %q: Check refuses what NewDescription built: %v", name, err)
		}
	}
}
