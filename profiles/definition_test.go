package profiles

import (
	"slices"
	"testing"
)

func TestGetUnknown(t *testing.T) {
	if _, ok := Get("nope"); ok {
		t.Error(`Get("nope") reported an unregistered profile as known`)
	}
}

func TestNames(t *testing.T) {
	want := []string{"basic"}
	if got := Names(); !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

// An empty LocalIdentifierScheme disables local-identifier extraction —
// the first definition-driven behavior variation.
func TestAssembleWithoutLocalIdentifierScheme(t *testing.T) {
	b, _, _ := newTestBuilder(t)

	def := basicDef(t)
	def.LocalIdentifierScheme = ""

	pkg, err := b.assemble(def)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if _, ok := pkg.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
		t.Error("MEEMOO-LOCAL-ID extracted despite an empty scheme")
	}
}
