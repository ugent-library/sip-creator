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
// (Builder.Build) reads them; a name looks up values in the registry.
type Definition struct {
	Name                     string
	Family                   Family
	DescriptiveSource        string // input filename of the descriptive metadata
	CharacterizationSource   string // input filename of the sidecar characterization report; "" disables discovery (ADR-0009)
	LocalIdentifierScheme    string // scheme for MEEMOO-LOCAL-ID extraction; "" disables
	EmitPackagePremis        bool
	EmitRepresentationPremis bool
	Mets                     sip.Spec
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
		Name:                     "basic",
		Family:                   FamilyMeemoo,
		DescriptiveSource:        "dc+schema.json",
		CharacterizationSource:   "siegfried.json",
		LocalIdentifierScheme:    "dcterms",
		EmitPackagePremis:        true,
		EmitRepresentationPremis: true,
		Mets: sip.Spec{
			// meemoo SIP 1.2, the stable spec (docs/plans/meemoo-12.md):
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
		Name:                     "eark",
		Family:                   FamilyEARK,
		DescriptiveSource:        "dc+schema.json",
		CharacterizationSource:   "siegfried.json",
		LocalIdentifierScheme:    "", // MEEMOO-LOCAL-ID is a meemoo concept
		EmitPackagePremis:        false,
		EmitRepresentationPremis: false, // RODA drops non-agent/event package PREMIS; v1 is essence + descriptive
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
