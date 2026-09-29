package profiles

import (
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

// withMETS returns the sorted set a profile ships when its descriptive
// document points at names: the METS set plus those.
func withMETS(names ...string) []string {
	return slices.Sorted(slices.Values(append(slices.Clone(mets.Schemas), names...)))
}

// Every XSD a profile ships is bundled, so a typo in an encoder's list fails
// here rather than at the first build, and each profile ships exactly what
// its documents point at: basic the METS set plus meemoo's descriptive
// schema and what it imports (the Dublin Core family, EDTF, schema.org,
// xml.xsd), eark the METS set plus dc.xsd, eark-mods the METS set plus
// mods-3-7.xsd. The bundle is the union of what the profiles ship, so no
// profile ships all of it.
func TestRegistrySchemas(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Names() {
		for _, xsd := range shipped(t, name) {
			if _, ok := bundle[xsd]; !ok {
				t.Errorf("profile %q ships %q, which is not bundled", name, xsd)
			}
		}
	}

	basic := withMETS("descriptive_basic.xsd", "dc.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd")
	if got := shipped(t, "basic"); !slices.Equal(got, basic) {
		t.Errorf("basic ships %v, want the METS set plus meemoo's descriptive schemas %v", got, basic)
	}
	eark := withMETS("dc.xsd")
	if got := shipped(t, "eark"); !slices.Equal(got, eark) {
		t.Errorf("eark ships %v, want the METS set plus dc.xsd %v", got, eark)
	}
	earkmods := withMETS("mods-3-7.xsd")
	if got := shipped(t, "eark-mods"); !slices.Equal(got, earkmods) {
		t.Errorf("eark-mods ships %v, want the METS set plus mods-3-7.xsd %v", got, earkmods)
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

// Every profile's NewDescription builds what its encoder's Check accepts:
// the CLI hands decoded rows to the one, and the engine runs the other on
// the result. A profile without the function could not read a folder.
func TestRegistryProfilesBuildWhatTheyCheck(t *testing.T) {
	for _, name := range Names() {
		def, _ := Get(name)
		if def.NewDescription == nil {
			t.Errorf("profile %q names no NewDescription", name)
			continue
		}
		if err := def.Encoder.Check(def.NewDescription(nil)); err != nil {
			t.Errorf("profile %q: Check refuses what NewDescription built: %v", name, err)
		}
	}
}
