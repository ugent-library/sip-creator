package build

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// Definition declares a profile as data: its metadata model, which
// metadata it emits, and the values its METS documents carry. Profiles
// differ in these values, not in build logic: Builder.Build reads the same
// fields for every profile.
type Definition struct {
	// Name identifies the profile, such as meemoo/basic or ugent/basic.
	Name string
	// Model is the profile's metadata model: the description type it
	// accepts and how a description is written as a document. It must be
	// set.
	Model MetadataModel
	// RequireSubmitterORID makes WithSubmitter require the submitting
	// organization's Meemoo OR-id and record it as the agent's
	// IDENTIFICATIONCODE note (Meemoo SIP 1.2, metsHdr).
	RequireSubmitterORID bool
	// MinRepresentations is the number of representations a package needs
	// at least. Zero allows a package without any, which carries metadata
	// only, as CSIP58 and the E-ARK SIP 2.2.0 introduction allow.
	MinRepresentations int
	// MaxRepresentations is the number of representations a package may
	// have at most. Zero sets no limit. With MinRepresentations, a minimum
	// and maximum of 1 mean exactly one, as Meemoo SIP 1.2's basic profile
	// requires: "The IE MUST be represented by exactly one representation."
	MaxRepresentations int
	// Representations maps each name a representation may have to the
	// profile's definition of a representation of that name. When it is
	// nil, any name that passes ValidateRepresentationName is allowed. When
	// it is set, a representation's name must be one of its keys, the type
	// follows from the name, and SourceRepresentation.Type must be empty.
	// Names are unique within a package, so a package holds at most one
	// representation of each name. The UGent profiles allow preservation,
	// archival and access (docs/profiles/ugent-basic.md §4: "A
	// representation's name MUST be one of preservation, archival and
	// access, exactly, in lowercase.").
	Representations map[string]RepresentationDefinition
	// RecordStatuses lists the record statuses a package under the profile
	// may state. When it is nil, every status sip.RecordStatus.IsValid
	// accepts is allowed. The UGent profiles allow NEW and UPDATE, the two
	// statuses RODA acts on (ADR-0038). A package without a record status
	// is always allowed: the E-ARK SIP specification reads it as NEW.
	RecordStatuses []sip.RecordStatus
	// AllowRepresentationDescriptions allows a representation to carry its
	// own description. When it is false, only the package carries a
	// description. Meemoo SIP 1.2's basic profile leaves it false: "There
	// MUST NOT be any descriptive metadata at the representation level."
	AllowRepresentationDescriptions bool
	// DocumentName is the file name of the descriptive document under
	// metadata/descriptive/: the package's convention for naming a document
	// in the model's format, such as dc+schema.xml or mods.xml.
	DocumentName string
	// EmitPackagePremis makes the package carry a generated PREMIS document
	// at metadata/preservation/premis.xml.
	EmitPackagePremis bool
	// EmitRepresentationPremis makes each representation carry a generated
	// PREMIS document at metadata/preservation/premis.xml.
	EmitRepresentationPremis bool
	// EmitRepresentationType declares each representation's type in that
	// representation's METS, in place of the profile's fixed content
	// typing: TYPE="Other" with the type as csip:OTHERTYPE, and
	// CONTENTINFORMATIONTYPE="OTHER" with the type as
	// csip:OTHERCONTENTINFORMATIONTYPE (ADR-0013). Under
	// Representations, the type is the one its definition of the name
	// states.
	// Otherwise it is SourceRepresentation.Type, which defaults to the
	// label and then to the name. The package METS keeps the profile
	// declaration unchanged.
	EmitRepresentationType bool
	// Declaration carries the METS values the profile's documents declare.
	Declaration sip.MetsDeclaration
}

// RepresentationDefinition is what a profile states about a representation
// of one name.
type RepresentationDefinition struct {
	// Type is the type the representation's METS declares when the profile
	// sets EmitRepresentationType (ADR-0013).
	Type string
	// Status is the status the representation's METS states, as the TYPE of
	// the main division of its structMap, or empty for none. METS and CSIP
	// leave that attribute free. commons-ip reads ORIGINAL there as the
	// original representation, and RODA keeps it as the representation's
	// original flag (ADR-0037).
	Status string
}

// representationType returns the type sr declares: the type its definition
// in Representations states when that is set, otherwise its resolved type.
func (d Definition) representationType(sr SourceRepresentation) string {
	if d.Representations != nil {
		return d.Representations[sr.Name].Type
	}
	return sr.resolvedType()
}

// representationDeclaration returns the declaration a representation's METS
// document carries. When the profile emits representation types, the
// declaration names typ as the representation's type, as
// EmitRepresentationType describes. Otherwise it is the package's
// declaration.
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
// the submitting organization. The submitter is not part of the profile,
// because one profile serves every organization that submits with it. The
// organization that runs the build adds itself here. When
// RequireSubmitterORID is set, the agent carries orID as an
// IDENTIFICATIONCODE note (Meemoo SIP 1.2, metsHdr). Otherwise it carries
// the name alone. It returns an error if name is empty, if name or orID
// holds text XML cannot carry, or if the profile requires an OR-id and
// orID is empty.
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

// ValidateSource checks that the source package is one the profile
// accepts. First it checks that every description is in the profile's
// metadata model. A supplied document passes only when the model
// implements DocumentFormat and accepts its root element. Then it checks
// the package against MinRepresentations, MaxRepresentations,
// Representations, RecordStatuses and AllowRepresentationDescriptions. It returns an
// error if a description is not in the model. Otherwise it returns an
// error naming each profile rule the package breaks, or nil. It writes
// nothing, so a program can check a source package against a profile
// without building it. SourcePackage.Validate checks the rules every
// package follows whatever its profile.
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
	if d.Representations != nil {
		names := slices.Sorted(maps.Keys(d.Representations))
		for _, r := range source.Representations {
			if _, ok := d.Representations[r.Name]; !ok {
				errs = append(errs, fmt.Errorf("profile %q names its representations %s; %q is not one of them", d.Name, strings.Join(names, ", "), r.Name))
			}
			if r.Type != "" {
				errs = append(errs, fmt.Errorf("under profile %q a representation's type follows from its name and cannot be set; %q has type %q", d.Name, r.Name, r.Type))
			}
		}
	}
	if d.RecordStatuses != nil && source.RecordStatus != "" && !slices.Contains(d.RecordStatuses, source.RecordStatus) {
		statuses := make([]string, len(d.RecordStatuses))
		for i, s := range d.RecordStatuses {
			statuses[i] = string(s)
		}
		errs = append(errs, fmt.Errorf("profile %q allows the record statuses %s; %q is not one of them", d.Name, strings.Join(statuses, ", "), source.RecordStatus))
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
