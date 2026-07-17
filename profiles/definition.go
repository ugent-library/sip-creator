package profiles

import (
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/sip"
)

// Family selects which output family a profile emits: the same assembled
// graph and the same canonical writer, different encodings (ADR-0007).
type Family string

// FamilyMeemoo emits E-ARK CSIP with the meemoo SIP specialization.
const FamilyMeemoo Family = "meemoo"

// FamilyEARK emits a plain E-ARK SIP.
const FamilyEARK Family = "eark"

// descriptiveEncoder is the one behavioral choice a family makes today.
// It grows into a struct of choices when families make more (ADR-0007).
type descriptiveEncoder func(io.Writer, sip.Descriptive) error

func (f Family) descriptiveEncoder() (descriptiveEncoder, error) {
	switch f {
	case FamilyMeemoo:
		return func(w io.Writer, d sip.Descriptive) error { return d.Encode(w) }, nil
	case FamilyEARK:
		return func(w io.Writer, d sip.Descriptive) error {
			dd, ok := d.(*metadata.Description)
			if !ok {
				return fmt.Errorf("eark descriptive encoding needs *metadata.Description, got %T", d)
			}
			return metadata.EncodeDC(w, dd)
		}, nil
	default:
		return nil, fmt.Errorf("unknown output family %q", f)
	}
}

// Definition declares a profile as data: what descriptive source it reads,
// which metadata it emits, and the values its METS documents carry.
// Profiles differ in these values, not in build logic — one engine
// (Builder.Build) reads them. Mirrors the formats/ registry pattern: a
// name looks up values.
type Definition struct {
	Name                     string
	Family                   Family
	DescriptiveSource        string // input filename of the descriptive metadata
	LocalIdentifierScheme    string // scheme for MEEMOO-LOCAL-ID extraction; "" disables
	EmitPackagePremis        bool
	EmitRepresentationPremis bool
	Mets                     sip.Spec
}

var registry = map[string]Definition{
	"basic": {
		Name:                     "basic",
		Family:                   FamilyMeemoo,
		DescriptiveSource:        "dc+schema.json",
		LocalIdentifierScheme:    "dcterms",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		Mets: sip.Spec{
			ProfileURL:                  "https://earkcsip.dilcis.eu/profile/E-ARK-CSIP.xml",
			Type:                        "Photographs – Digital", // known-wrong value, preserved; a one-line fix per the CSIP vocabulary
			ContentInformationType:      "OTHER",
			OtherContentInformationType: "https://data.hetarchief.be/id/sip/2.0/basic",
			DescriptiveMDType:           "DC",
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1."},
				{Role: "CREATOR", OtherRole: "OTHERROLE", Type: "ORGANIZATION", Name: "Universiteitsbibliotheek Gent"},
			},
		},
	},
	"eark": {
		Name:                     "eark",
		Family:                   FamilyEARK,
		DescriptiveSource:        "dc+schema.json",
		LocalIdentifierScheme:    "", // MEEMOO-LOCAL-ID is a meemoo concept
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false, // RODA drops non-agent/event package PREMIS; v1 is essence + descriptive
		Mets: sip.Spec{
			// The version-pinned profile URL: commons-ip's SIP2 check for
			// spec 2.2.0 compares against this exact value (its error
			// message misleadingly prints the unversioned URL).
			ProfileURL:               "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
			Type:                     "Mixed",  // CSIP content-category vocabulary
			ContentInformationType:   "MIXED",  // becomes the AIP type in RODA
			DescriptiveMDType:        "DC",
			DescriptiveMDTypeVersion: "SimpleDC20021212", // the shape RODA renders natively
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1."},
				{Role: "CREATOR", Type: "ORGANIZATION", Name: "Universiteitsbibliotheek Gent"},
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
