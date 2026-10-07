package meemoo

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the basic profile: Meemoo SIP 1.2's basic content profile
// on the E-ARK SIP profile of its era, writing dc+schema.xml from Terms.
// The registry in profiles/ hands it out under the name "meemoo/basic".
var Definition = build.Definition{
	Name:  "meemoo/basic",
	Model: dcschema{},
	// Meemoo identifies the submitting organization by its OR-id
	// (Meemoo SIP 1.2, metsHdr agent note).
	RequireSubmitterORID: true,
	// The basic profile allows exactly one representation: a maximum of 1
	// here, with the one every package needs (SourcePackage.Validate). It
	// allows no descriptive metadata at the representation level
	// (AllowRepresentationDescriptions left false).
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
		ProfileURL:                  "https://earksip.dilcis.eu/profile/E-ARK-SIP.xml",
		Type:                        "Photographs – Digital", // 1.2 content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
		ContentInformationType:      "OTHER",
		OtherContentInformationType: "https://data.hetarchief.be/id/sip/1.2/basic",
		// No agents: the engine adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}
