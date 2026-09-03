package profiles

import (
	"errors"
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
// schemas is the relative path from the document being written to the
// package's schemas/ dir; only the writer knows where a document lands.
type descriptiveEncoder func(w io.Writer, t metadata.Terms, schemas string) error

func (f Family) descriptiveEncoder() (descriptiveEncoder, error) {
	switch f {
	case FamilyMeemoo:
		return metadata.EncodeTerms, nil
	case FamilyEARK:
		return metadata.EncodeDCTerms, nil
	default:
		return nil, fmt.Errorf("unknown output family %q", f)
	}
}

// Definition declares a profile as data: what descriptive source it reads,
// which metadata it emits, and the values its METS documents carry.
// Profiles differ in these values, not in build logic: one engine
// (Builder.Build) reads them; a name looks up values in the registry.
type Definition struct {
	// Name is the registry key: what --profile selects.
	Name string
	// Family selects which encodings the package uses.
	Family Family
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
	// RepresentationTypeFromLabel types each representation METS by the
	// producer's label instead of the profile's fixed content typing:
	// TYPE="Other" with the label as csip:OTHERTYPE, and
	// CONTENTINFORMATIONTYPE="OTHER" with the label as
	// csip:OTHERCONTENTINFORMATIONTYPE. Ingest systems read one of those
	// pairs as the representation's type (ADR-0013). The package METS keeps
	// the profile declaration unchanged.
	RepresentationTypeFromLabel bool
	// Declaration carries the METS values the profile's documents declare.
	Declaration sip.MetsDeclaration

	// RequiredElements are the descriptive elements the profile's spec makes
	// required at package level: meemoo's basic profile requires four,
	// plain E-ARK only the input convention's identity MUSTs.
	RequiredElements []string
	// RequiredLang, when set, requires every language-tagged element in
	// the descriptive terms to include a value in this language; meemoo
	// requires a Dutch entry wherever a lang-tagged element appears.
	RequiredLang string
	// EnforceCardinality applies the vocabulary table's cardinality limits
	// (meemoo's 0..1/1..1 restrictions); plain E-ARK has no such rule.
	EnforceCardinality bool
}

// validateDescriptive checks the input's descriptive terms against the
// profile's conformance rules (requiredness, required language, and the
// vocabulary's cardinality limits), all Definition data. Requiredness applies
// at package level only (identity lives there; representation descriptive
// is optional), but the language and cardinality rules constrain what any
// emitted document may say, so they cover representation descriptive too.
// Findings are joined so one failed build names every gap at once.
func (d Definition) validateDescriptive(in *Input) error {
	errs := []error{
		in.Descriptive.ValidateRequired(d.RequiredElements...),
		in.Descriptive.ValidateRequiredLang(d.RequiredLang),
	}
	if d.EnforceCardinality {
		errs = append(errs, in.Descriptive.ValidateCardinality())
	}
	for _, r := range in.Representations {
		repErrs := []error{r.Descriptive.ValidateRequiredLang(d.RequiredLang)}
		if d.EnforceCardinality {
			repErrs = append(repErrs, r.Descriptive.ValidateCardinality())
		}
		for _, err := range repErrs {
			if err != nil {
				errs = append(errs, fmt.Errorf("representation %q: %w", r.Label, err))
			}
		}
	}
	return errors.Join(errs...)
}

// representationDeclaration returns the declaration a representation's METS
// document carries: the profile declaration as-is, or, when the profile types
// representations by label, a copy whose content typing names the label.
// The label lands in both the TYPE and the CONTENTINFORMATIONTYPE pair
// because ingest systems disagree on which pair they read as the
// representation's type (ADR-0013).
func (d Definition) representationDeclaration(label string) *sip.MetsDeclaration {
	decl := d.Declaration
	if d.RepresentationTypeFromLabel {
		decl.Type = "Other"
		decl.OtherType = label
		decl.ContentInformationType = "OTHER"
		decl.OtherContentInformationType = label
	}
	return &decl
}

// WithSubmitter returns a copy of the definition whose METS agents include
// the submitting organization. The submitter is operator identity, not
// profile data, so the registry entries omit it and the caller supplies it.
// The family decides its shape: meemoo requires the organization's OR-id
// as an IDENTIFICATIONCODE note (meemoo SIP 1.2, metsHdr); plain E-ARK
// carries the name alone.
func (d Definition) WithSubmitter(name, orID string) (Definition, error) {
	if name == "" {
		return Definition{}, fmt.Errorf("profile %q requires the submitting organization's name", d.Name)
	}
	agent := sip.Agent{Role: "CREATOR", Type: "ORGANIZATION", Name: name}
	if d.Family == FamilyMeemoo {
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
		Name:   "basic",
		Family: FamilyMeemoo,
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
		// elements, a Dutch value for every lang-tagged element, and the
		// table's cardinality limits.
		RequiredElements:   metadata.RequiredElements(),
		RequiredLang:       "nl",
		EnforceCardinality: true,
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
		Name:   "eark",
		Family: FamilyEARK,
		// Named after the simple-DC document it holds; meemoo's naming
		// convention doesn't apply to the eark family.
		DescriptiveName:     "dc.xml",
		EmitLocalIdentifier: false, // MEEMOO-LOCAL-ID is a meemoo concept
		// dc.xml keeps the producer's identifier: CSIP has no rule tying it
		// to the package identifier (mets/@OBJID carries that), and the
		// ingesting catalogue indexes dc.xml, so operators find the package
		// by the identifier they know (ADR-0012).
		SwapObjectIdentifier: false,
		// The eark family emits no PREMIS: RODA drops package PREMIS that
		// does not describe agents or events.
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false,
		// Only the input convention's identity MUSTs; no required language,
		// and meemoo's cardinality limits don't apply to a plain E-ARK package.
		RequiredElements: []string{"dcterms:identifier", "dcterms:title"},
		// RODA shows each representation's type from the representation
		// METS's content typing; the label is the only per-representation
		// name we carry (ADR-0013).
		RepresentationTypeFromLabel: true,
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
