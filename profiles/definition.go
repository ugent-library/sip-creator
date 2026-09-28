package profiles

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ugent-library/sip-creator/encoders/dc"
	"github.com/ugent-library/sip-creator/encoders/dcschema"
	"github.com/ugent-library/sip-creator/encoders/mets"
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
	// Schemas are the XSD file names, from the bundle in schemas/, that the
	// profile's packages ship under schemas/: the lists each encoder
	// exports for the schemas its document points at (mets.Schemas,
	// dcschema.Schemas, dc.Schemas), concatenated. A name may appear in
	// more than one list; each ships once. Every name must be bundled.
	Schemas []string
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
		DescriptiveName:          "dc+schema.xml",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
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
		// What the METS documents and dc+schema.xml point at.
		Schemas: slices.Concat(mets.Schemas, dcschema.Schemas),
	},
	"eark": {
		Name:        "eark",
		descriptive: simpleDC,
		// Named after the simple-DC document it holds; meemoo's naming
		// convention doesn't apply to the eark profile.
		DescriptiveName: "dc.xml",
		// The eark profile emits no PREMIS: RODA drops package PREMIS that
		// does not describe agents or events.
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false,
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
		// What the METS documents and dc.xml point at, plus the dc+schema
		// list, which no eark document references: it ships so the output
		// stays as it was, and dropping it is a deliberate output change to
		// make on its own.
		Schemas: slices.Concat(mets.Schemas, dc.Schemas, dcschema.Schemas),
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
