package meemoo

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the meemoo/basic profile: Meemoo SIP 1.2's basic content
// profile, writing dc+schema.xml from Terms.
var Definition = build.Definition{
	Name:  "meemoo/basic",
	Model: dcschema{},
	// Meemoo identifies the submitting organization by its OR-id
	// (Meemoo SIP 1.2, metsHdr agent note).
	RequireSubmitterORID: true,
	// The basic profile allows exactly one representation: "The IE MUST be
	// represented by exactly one representation." It also leaves
	// AllowRepresentationDescriptions false: "There MUST NOT be any
	// descriptive metadata at the representation level."
	MinRepresentations: 1,
	MaxRepresentations: 1,
	// The filename Meemoo's basic profile expects for the descriptive
	// document.
	DocumentName:             "dc+schema.xml",
	EmitPackagePremis:        true,
	EmitRepresentationPremis: true,
	Declaration: sip.MetsDeclaration{
		// Meemoo SIP 1.2, the stable spec (docs/archive/meemoo-12.md):
		// 1.2 requires the unversioned E-ARK SIP profile URL and the
		// 1.2 profile URI as OTHERCONTENTINFORMATIONTYPE.
		ProfileURL: "https://earksip.dilcis.eu/profile/E-ARK-SIP.xml",
		// The default content category, from the 1.2 vocabulary. A
		// package's SourcePackage.ContentCategory replaces it.
		Type:                        "Photographs – Digital",
		ContentInformationType:      "OTHER",
		OtherContentInformationType: "https://data.hetarchief.be/id/sip/1.2/basic",
		// No agents: Builder.Build adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}
