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
// package's schemas/ dir — only the writer knows where a document lands.
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
// Profiles differ in these values, not in build logic — one engine
// (Builder.Build) reads them; a name looks up values in the registry.
type Definition struct {
	Name                     string
	Family                   Family
	DescriptiveName          string // emitted filename of the descriptive document under metadata/descriptive/
	LocalIdentifierScheme    string // scheme for MEEMOO-LOCAL-ID extraction; "" disables
	EmitPackagePremis        bool
	EmitRepresentationPremis bool
	Mets                     sip.Spec

	// RequiredElements are the descriptive elements the profile's spec makes
	// mandatory at package level: meemoo's basic profile mandates four,
	// plain E-ARK only the input convention's identity MUSTs.
	RequiredElements []string
	// RequiredLang, when set, requires every language-tagged element in
	// the descriptive terms to include a value in this language — meemoo
	// demands an entry in Dutch wherever a lang-tagged element appears.
	RequiredLang string
	// EnforceCardinality binds the vocabulary table's repeatability marks
	// (meemoo's 0..1/1..1 restrictions); plain E-ARK has no such rule.
	EnforceCardinality bool
}

// validateDescriptive checks the input's descriptive terms against the
// profile's conformance rules — requiredness, required language, and the
// vocabulary's cardinality marks, all Definition data. Findings are joined
// so one failed build names every gap at once.
func (d Definition) validateDescriptive(in *Input) error {
	errs := []error{
		in.Descriptive.ValidateRequired(d.RequiredElements...),
		in.Descriptive.ValidateRequiredLang(d.RequiredLang),
	}
	if d.EnforceCardinality {
		errs = append(errs, in.Descriptive.ValidateCardinality())
		for _, r := range in.Representations {
			if err := r.Descriptive.ValidateCardinality(); err != nil {
				errs = append(errs, fmt.Errorf("representation %q: %w", r.Label, err))
			}
		}
	}
	return errors.Join(errs...)
}

// WithSubmitter returns a copy of the definition whose METS agents include
// the submitting organization. The submitter is operator identity, not
// profile data, so the registry entries omit it and the caller supplies it
// (the CLI from SIP_SUBMITTER_* config, an embedding system as arguments).
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
	// Clone before appending: d.Mets.Agents shares its backing array with
	// the registry entry, and append must never write into it.
	d.Mets.Agents = append(slices.Clone(d.Mets.Agents), agent)
	return d, nil
}

var registry = map[string]Definition{
	"basic": {
		Name:   "basic",
		Family: FamilyMeemoo,
		// The filename meemoo's basic profile expects for the descriptive
		// document; kept from the pre-terms era so packages stay identical.
		DescriptiveName:          "dc+schema.xml",
		LocalIdentifierScheme:    "dcterms",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		// meemoo's basic content profile: the vocabulary table's Required
		// marks bind, a Dutch entry wherever a lang-tagged element appears,
		// and the table's cardinalities bind.
		RequiredElements:   metadata.RequiredElements(),
		RequiredLang:       "nl",
		EnforceCardinality: true,
		Mets: sip.Spec{
			// meemoo SIP 1.2, the stable spec (docs/archive/meemoo-12.md):
			// 1.2 mandates the unversioned E-ARK SIP profile URL and the
			// 1.2 profile URI as OTHERCONTENTINFORMATIONTYPE.
			ProfileURL:                  "https://earksip.dilcis.eu/profile/E-ARK-SIP.xml",
			Type:                        "Photographs – Digital", // legal 1.2 content-category vocabulary; operator-selectable someday (TODO)
			ContentInformationType:      "OTHER",
			OtherContentInformationType: "https://data.hetarchief.be/id/sip/1.2/basic",
			DescriptiveMDType:           "DC",
			// Only the software agent: the submitting ORGANIZATION agent is
			// operator identity, not profile data — WithSubmitter appends it.
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1.", NoteType: "SOFTWARE VERSION"},
			},
		},
	},
	"eark": {
		Name:   "eark",
		Family: FamilyEARK,
		// Honest name for the simple-DC document RODA renders; no meemoo
		// naming convention applies to the eark family.
		DescriptiveName:          "dc.xml",
		LocalIdentifierScheme:    "", // MEEMOO-LOCAL-ID is a meemoo concept
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false, // RODA drops non-agent/event package PREMIS; v1 is essence + descriptive
		// Only the input convention's identity MUSTs; no required language,
		// and meemoo's cardinality marks don't bind a plain E-ARK package.
		RequiredElements: []string{"dcterms:identifier", "dcterms:title"},
		Mets: sip.Spec{
			// The version-pinned profile URL: commons-ip's SIP2 check for
			// spec 2.2.0 compares against this exact value (its error
			// message misleadingly prints the unversioned URL).
			ProfileURL:               "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
			Type:                     "Mixed", // CSIP content-category vocabulary
			ContentInformationType:   "MIXED", // becomes the AIP type in RODA
			DescriptiveMDType:        "DC",
			DescriptiveMDTypeVersion: "SimpleDC20021212", // the shape RODA renders natively
			// Only the software agent — see the basic entry: the submitting
			// ORGANIZATION agent comes from WithSubmitter.
			Agents: []sip.Agent{
				{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1.", NoteType: "SOFTWARE VERSION"},
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
