package build

import (
	"fmt"
	"slices"

	"github.com/ugent-library/sip-creator/sip"
)

// Definition declares a profile as data: the encoder for its descriptive
// document, which metadata it emits, and the values its METS documents
// carry.
// Profiles differ in these values, not in build logic: one engine
// (Builder.Build) reads them. Each profile package under profiles/ builds
// its own definition, and the registry in profiles/ hands them out by name.
type Definition struct {
	// Name is the registry key: what --profile selects.
	Name string
	// Encoder writes the profile's descriptive document from the
	// description it accepts. A definition without one is refused before
	// any write.
	Encoder DescriptionEncoder
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
}

// representationDeclaration returns the declaration a representation's METS
// document carries: the package's declaration as-is, or, when the profile
// emits representation types, a copy whose content typing names typ as
// EmitRepresentationType describes.
func (d Definition) representationDeclaration(base sip.MetsDeclaration, typ string) *sip.MetsDeclaration {
	decl := base
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
// profile data: one profile serves every organization that submits with
// it, so the profile packages omit the submitter and the organization
// running the build adds it here.
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
	// with the profile package's value, and append must never write into it.
	d.Declaration.Agents = append(slices.Clone(d.Declaration.Agents), agent)
	return d, nil
}
