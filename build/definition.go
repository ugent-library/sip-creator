package build

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// Definition declares a profile as data: its metadata model, which
// metadata it emits, and the values its METS documents carry.
// Profiles differ in these values, not in build logic: one engine
// (Builder.Build) reads them. Each profile package under profiles/ builds
// its own definition, and the registry in profiles/ hands them out by name.
type Definition struct {
	// Name is the registry key: what --profile selects.
	Name string
	// Model is the profile's metadata model: the description type it
	// accepts and how a description is written as a document. A definition
	// without one is refused before any write.
	Model MetadataModel
	// RequireSubmitterORID requires the submitting organization's Meemoo
	// OR-id, emitted as the agent's IDENTIFICATIONCODE note (Meemoo SIP
	// 1.2, metsHdr); WithSubmitter needs the OR-id when set.
	RequireSubmitterORID bool
	// MinRepresentations is the number of representations a package needs
	// at least; zero allows a package without any, which carries metadata
	// only (CSIP58: "In the case that a package only contains metadata
	// updates, i.e. exclusively metadata files, then no file references need
	// to be added to this section."; the E-ARK SIP 2.2.0 introduction: "A
	// package with zero representations means that it only contains
	// metadata.").
	MinRepresentations int
	// MaxRepresentations is the number of representations a package may
	// have at most; zero sets no limit. With MinRepresentations, a minimum
	// and maximum of 1 mean exactly one, as Meemoo SIP 1.2's basic profile
	// requires: "The IE MUST be represented by exactly one representation."
	MaxRepresentations int
	// RepresentationTypes is the closed set of names a representation may
	// have; nil sets none. Under a set, a representation's name is also its
	// type: an empty SourceRepresentation.Type resolves to the name, and a
	// Type that differs from it is refused. Names are unique within a
	// package, so a package holds at most one representation of each type.
	// The UGent profiles name preservation, archival and access
	// (docs/profiles/ugent-basic.md §4: "A representation's name MUST be one
	// of preservation, archival and access, exactly, in lowercase.").
	RepresentationTypes []string
	// AllowRepresentationDescriptions allows a representation to carry its
	// own description; false allows one at the package level only. Meemoo
	// SIP 1.2's basic profile leaves it false: "There MUST NOT be any
	// descriptive metadata at the representation level."
	AllowRepresentationDescriptions bool
	// DocumentName is the file name of the descriptive document under
	// metadata/descriptive/: the package's convention for naming a document
	// in the model's format, such as dc+schema.xml or mods.xml.
	DocumentName string
	// EmitPackagePremis emits the generated package PREMIS document.
	EmitPackagePremis bool
	// EmitRepresentationPremis emits a generated PREMIS document per
	// representation.
	EmitRepresentationPremis bool
	// EmitRepresentationType declares each representation's resolved type
	// (SourceRepresentation.Type, defaulting to the label, then the name;
	// under RepresentationTypes, the name)
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

// representationType returns the type sr declares: its name under a
// vocabulary of representation types, otherwise its resolved type.
func (d Definition) representationType(sr SourceRepresentation) string {
	if d.RepresentationTypes != nil {
		return sr.Name
	}
	return sr.resolvedType()
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
// RequireSubmitterORID decides its shape: Meemoo requires the
// organization's OR-id as an IDENTIFICATIONCODE note (Meemoo SIP 1.2,
// metsHdr); plain E-ARK carries the name alone.
func (d Definition) WithSubmitter(name, orID string) (Definition, error) {
	if name == "" {
		return Definition{}, fmt.Errorf("profile %q requires the submitting organization's name", d.Name)
	}
	if err := ValidateXMLText(name); err != nil {
		return Definition{}, fmt.Errorf("submitting organization's name: %w", err)
	}
	if err := ValidateXMLText(orID); err != nil {
		return Definition{}, fmt.Errorf("submitting organization's OR-id: %w", err)
	}
	agent := sip.Agent{Role: "CREATOR", Type: "ORGANIZATION", Name: name}
	if d.RequireSubmitterORID {
		if orID == "" {
			return Definition{}, fmt.Errorf("profile %q requires the submitting organization's Meemoo OR-id", d.Name)
		}
		agent.Note = orID
		agent.NoteType = "IDENTIFICATIONCODE"
	}
	// Clone before appending: d.Declaration.Agents shares its backing array
	// with the profile package's value, and append must never write into it.
	d.Declaration.Agents = append(slices.Clone(d.Declaration.Agents), agent)
	return d, nil
}

// ValidateSource returns why the source package is not one the profile
// accepts: a description that is not in the profile's metadata model, or a
// package that breaks the profile's own rules (MinRepresentations,
// MaxRepresentations, RepresentationTypes,
// AllowRepresentationDescriptions). The profile's rules are joined, one error each,
// so all of them can be reported at once. It writes nothing, so a source
// package can be checked against the profile without building it. What
// every package needs regardless of profile is SourcePackage.Validate's.
func (d Definition) ValidateSource(source *SourcePackage) error {
	if err := checkDescriptions(d.Model, source); err != nil {
		return fmt.Errorf("profile %q: %w", d.Name, err)
	}

	var errs []error
	if len(source.Representations) < d.MinRepresentations {
		errs = append(errs, fmt.Errorf("profile %q needs at least %d representation(s), the package has %d", d.Name, d.MinRepresentations, len(source.Representations)))
	}
	if d.MaxRepresentations > 0 && len(source.Representations) > d.MaxRepresentations {
		errs = append(errs, fmt.Errorf("profile %q allows at most %d representation(s), the package has %d", d.Name, d.MaxRepresentations, len(source.Representations)))
	}
	if d.RepresentationTypes != nil {
		for _, r := range source.Representations {
			if !slices.Contains(d.RepresentationTypes, r.Name) {
				errs = append(errs, fmt.Errorf("profile %q names its representations %s; %q is not one of them", d.Name, strings.Join(d.RepresentationTypes, ", "), r.Name))
			}
			if r.Type != "" && r.Type != r.Name {
				errs = append(errs, fmt.Errorf("under profile %q a representation's type is its name; %q has type %q", d.Name, r.Name, r.Type))
			}
		}
	}
	if !d.AllowRepresentationDescriptions {
		for _, r := range source.Representations {
			if r.Description != nil {
				errs = append(errs, fmt.Errorf("representation %q has a description; profile %q allows one at the package level only", r.Name, d.Name))
			}
		}
	}
	return errors.Join(errs...)
}
