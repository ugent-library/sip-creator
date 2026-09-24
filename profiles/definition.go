package profiles

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/encoders/dcschema"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition declares a profile as data: what descriptive source it reads,
// which metadata it emits, and the values its METS documents carry.
// Profiles differ in these values, not in build logic: one engine
// (Builder.Build) reads them; a name looks up values in the registry.
// Definitions come from the registry (Get): the descriptive standard a
// profile writes is a closed set and cannot be set from outside the package.
type Definition struct {
	// Name is the registry key: what --profile selects.
	Name string
	// descriptive is the descriptive standard the profile accepts and the
	// document it writes: one of the values in descriptive.go.
	descriptive descriptive
	// RequireSubmitterORID requires the submitting organization's meemoo
	// OR-id, emitted as the agent's IDENTIFICATIONCODE note (meemoo SIP
	// 1.2, metsHdr); WithSubmitter needs the OR-id when set.
	RequireSubmitterORID bool
	// DescriptiveName is the emitted filename of the descriptive document
	// under metadata/descriptive/.
	DescriptiveName string
	// EmitLocalIdentifier lifts the producer's identifier onto the entity
	// as MEEMOO-LOCAL-ID.
	EmitLocalIdentifier bool
	// SwapObjectIdentifier replaces the descriptive dcterms:identifier with
	// the entity (or representation) identifier in the emitted document.
	// When false the document keeps the producer's own identifier
	// (ADR-0012).
	SwapObjectIdentifier bool
	// EmitPackagePremis emits the generated package PREMIS document.
	EmitPackagePremis bool
	// EmitRepresentationPremis emits a generated PREMIS document per
	// representation.
	EmitRepresentationPremis bool
	// EmitRepresentationType declares each representation's resolved type
	// (SourceRepresentation.Type, defaulting to the label, then the name)
	// in that representation's METS instead of the profile's fixed content
	// typing: TYPE="Other" with the type as csip:OTHERTYPE, and
	// CONTENTINFORMATIONTYPE="OTHER" with the type as
	// csip:OTHERCONTENTINFORMATIONTYPE. Ingest systems read one of those
	// pairs as the representation's type (ADR-0013). The package METS keeps
	// the profile declaration unchanged.
	EmitRepresentationType bool
	// Declaration carries the METS values the profile's documents declare.
	Declaration sip.MetsDeclaration

	// RequiredKeys are the vocabulary keys the profile's spec makes required
	// at package level (identifier, title, ...), which the profile's
	// descriptive standard resolves through its own table: meemoo's basic
	// profile requires four, plain E-ARK only the input convention's
	// identity MUSTs.
	RequiredKeys []string
}

// validateDescriptive checks the package-level description against the
// profile's required keys, Definition data. Requiredness applies at
// package level only: identity lives there, and a representation's
// description is optional. Every other rule of the descriptive standard
// (term validity, meemoo's cardinality and language rules) is the
// standard's own and runs in Input.Validate. Findings are joined so one
// failed build names every gap at once.
func (d Definition) validateDescriptive(in *Input) error {
	return in.Descriptive.ValidateRequired(d.RequiredKeys...)
}

// representationDeclaration returns the declaration a representation's METS
// document carries: the profile declaration as-is, or, when the profile
// emits representation types, a copy whose content typing names the resolved
// type. The type lands in both the TYPE and the CONTENTINFORMATIONTYPE pair
// because ingest systems disagree on which pair they read as the
// representation's type (ADR-0013).
func (d Definition) representationDeclaration(typ string) *sip.MetsDeclaration {
	decl := d.Declaration
	if d.EmitRepresentationType {
		decl.Type = "Other"
		decl.OtherType = typ
		decl.ContentInformationType = "OTHER"
		decl.OtherContentInformationType = typ
	}
	return &decl
}

// WithSubmitter returns a copy of the definition whose METS agents include
// the submitting organization. The submitter is operator identity, not
// profile data, so the registry entries omit it and the caller supplies it.
// RequireSubmitterORID decides its shape: meemoo requires the
// organization's OR-id as an IDENTIFICATIONCODE note (meemoo SIP 1.2,
// metsHdr); plain E-ARK carries the name alone.
func (d Definition) WithSubmitter(name, orID string) (Definition, error) {
	if name == "" {
		return Definition{}, fmt.Errorf("profile %q requires the submitting organization's name", d.Name)
	}
	agent := sip.Agent{Role: "CREATOR", Type: "ORGANIZATION", Name: name}
	if d.RequireSubmitterORID {
		if orID == "" {
			return Definition{}, fmt.Errorf("profile %q requires the submitting organization's meemoo OR-id", d.Name)
		}
		agent.Note = orID
		agent.NoteType = "IDENTIFICATIONCODE"
	}
	// Clone before appending: d.Declaration.Agents shares its backing array
	// with the registry entry, and append must never write into it.
	d.Declaration.Agents = append(slices.Clone(d.Declaration.Agents), agent)
	return d, nil
}

var registry = map[string]Definition{
	"basic": {
		Name:        "basic",
		descriptive: meemooDC,
		// meemoo identifies the submitting organization by its OR-id
		// (meemoo SIP 1.2, metsHdr agent note).
		RequireSubmitterORID: true,
		// The filename meemoo's basic profile expects for the descriptive
		// document.
		DescriptiveName: "dc+schema.xml",
		// meemoo links descriptive to preservation metadata by a shared
		// UUID: dc+schema.xml carries the entity identifier, and the
		// producer's own identifier travels as a MEEMOO-LOCAL-ID PREMIS
		// object identifier.
		EmitLocalIdentifier:      true,
		SwapObjectIdentifier:     true,
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		// meemoo's basic content profile: the vocabulary table's required
		// keys. Its cardinality limits and Dutch-language rule are the
		// dcschema standard's own (dcschema.Terms.Validate).
		RequiredKeys: dcschema.RequiredKeys(),
		Declaration: sip.MetsDeclaration{
			// meemoo SIP 1.2, the stable spec (docs/archive/meemoo-12.md):
			// 1.2 requires the unversioned E-ARK SIP profile URL and the
			// 1.2 profile URI as OTHERCONTENTINFORMATIONTYPE.
			ProfileURL:                  "https://earksip.dilcis.eu/profile/E-ARK-SIP.xml",
			Type:                        "Photographs – Digital", // 1.2 content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
			ContentInformationType:      "OTHER",
			OtherContentInformationType: "https://data.hetarchief.be/id/sip/1.2/basic",
			DescriptiveMDType:           "DC",
			// Only the software agent; WithSubmitter appends the
			// submitting organization.
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1", NoteType: "SOFTWARE VERSION"},
			},
		},
	},
	"eark": {
		Name:        "eark",
		descriptive: simpleDC,
		// Named after the simple-DC document it holds; meemoo's naming
		// convention doesn't apply to the eark profile.
		DescriptiveName:     "dc.xml",
		EmitLocalIdentifier: false, // MEEMOO-LOCAL-ID is a meemoo concept
		// dc.xml keeps the producer's identifier: CSIP has no rule tying it
		// to the package identifier (mets/@OBJID carries that), and the
		// ingesting catalogue indexes dc.xml, so operators find the package
		// by the identifier they know (ADR-0012).
		SwapObjectIdentifier: false,
		// The eark profile emits no PREMIS: RODA drops package PREMIS that
		// does not describe agents or events.
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false,
		// Only the input convention's identity MUSTs.
		RequiredKeys: []string{"identifier", "title"},
		// RODA shows each representation's type from the representation
		// METS's content typing (ADR-0013).
		EmitRepresentationType: true,
		Declaration: sip.MetsDeclaration{
			// The version-pinned profile URL: commons-ip's SIP2 check for
			// spec 2.2.0 compares against this exact value (its error
			// message misleadingly prints the unversioned URL).
			ProfileURL:               "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
			Type:                     "Mixed", // CSIP content-category vocabulary
			ContentInformationType:   "MIXED", // package METS value; RODA reads it as the AIP type
			DescriptiveMDType:        "DC",
			DescriptiveMDTypeVersion: "SimpleDC20021212", // the shape RODA renders natively
			// Only the software agent; WithSubmitter appends the
			// submitting organization.
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1", NoteType: "SOFTWARE VERSION"},
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
