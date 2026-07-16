package profiles

import (
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/sip"
)

// Definition declares a profile as data: what descriptive source it reads,
// which metadata it emits, and the values its METS documents carry.
// Profiles differ in these values, not in build logic — one engine
// (Builder.Build) reads them. Mirrors the formats/ registry pattern: a
// name looks up values.
type Definition struct {
	Name                     string
	DescriptiveSource        string // input filename of the descriptive metadata
	LocalIdentifierScheme    string // scheme for MEEMOO-LOCAL-ID extraction; "" disables
	EmitPackagePremis        bool
	EmitRepresentationPremis bool
	Mets                     sip.Spec
}

var registry = map[string]Definition{
	"basic": {
		Name:                     "basic",
		DescriptiveSource:        "dc+schema.json",
		LocalIdentifierScheme:    "dcterms",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		Mets: sip.Spec{
			ProfileURL:                  "https://earkcsip.dilcis.eu/profile/E-ARK-CSIP.xml",
			Type:                        "Photographs – Digital", // known-wrong value, preserved; a one-line fix per the CSIP vocabulary
			ContentInformationType:      "OTHER",
			OtherContentInformationType: "https://data.hetarchief.be/id/sip/2.0/basic",
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1."},
				{Role: "CREATOR", OtherRole: "OTHERROLE", Type: "ORGANIZATION", Name: "Universiteitsbibliotheek Gent"},
			},
		},
	},
}

// Get resolves a profile name to its definition.
func Get(name string) (Definition, bool) {
	def, ok := registry[name]
	return def, ok
}

// Names lists the registered profiles, sorted, for CLI error messages.
func Names() []string {
	return slices.Sorted(maps.Keys(registry))
}
