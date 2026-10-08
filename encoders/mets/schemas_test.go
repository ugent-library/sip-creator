package mets

import (
	"regexp"
	"slices"
	"testing"

	"github.com/ugent-library/sip-creator/schemas"
)

// Every schema the METS templates point at is bundled, so a package can
// ship it.
func TestSchemasBundled(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Schemas {
		if _, ok := bundle[name]; !ok {
			t.Errorf("%s is not in the schema bundle", name)
		}
	}
}

// The MDTYPE vocabulary is the one the bundled METS schema allows, value
// for value, so a name the schema lists is never recorded as OTHER.
func TestMDTypesMatchTheMETSSchema(t *testing.T) {
	xsd := string(schemas.Get()["mets1_12.xsd"])
	attribute := regexp.MustCompile(`(?s)<xsd:attribute name="MDTYPE".*?</xsd:attribute>`).FindString(xsd)
	if attribute == "" {
		t.Fatal("no MDTYPE attribute in mets1_12.xsd")
	}
	var allowed []string
	for _, m := range regexp.MustCompile(`value="([^"]*)"`).FindAllStringSubmatch(attribute, -1) {
		allowed = append(allowed, m[1])
	}
	if !slices.Equal(mdTypes, allowed) {
		t.Errorf("mdTypes = %v, want the schema's %v", mdTypes, allowed)
	}
}

// A listed name is the MDTYPE itself. Any other name is OTHER, with the
// name as OTHERMDTYPE. The vocabulary's spelling decides: dc is not DC.
func TestMDType(t *testing.T) {
	for _, tc := range []struct {
		format, mdType, other string
	}{
		{"DC", "DC", ""},
		{"MODS", "MODS", ""},
		{"OTHER", "OTHER", ""},
		{"EBUCore", "OTHER", "EBUCore"},
		{"dc", "OTHER", "dc"},
	} {
		mdType, other := MDType(tc.format)
		if mdType != tc.mdType || other != tc.other {
			t.Errorf("MDType(%q) = %q, %q; want %q, %q", tc.format, mdType, other, tc.mdType, tc.other)
		}
	}
}
