package meemoo

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the basic profile: Meemoo SIP 1.2's basic content profile
// on the E-ARK SIP profile of its era, writing dc+schema.xml from Terms.
// The registry in profiles/ hands it out under the name "basic".
var Definition = build.Definition{
	Name:  "basic",
	Model: dcschema{},
	// Meemoo identifies the submitting organization by its OR-id
	// (Meemoo SIP 1.2, metsHdr agent note).
	RequireSubmitterORID: true,
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
		DescriptiveMDType:           "DC",
		// Only the software agent; WithSubmitter appends the
		// submitting organization.
		Agents: []sip.Agent{
			{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1", NoteType: "SOFTWARE VERSION"},
		},
	},
}
