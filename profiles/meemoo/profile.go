package meemoo

import (
	"fmt"
	"io"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the basic profile: meemoo SIP 1.2's basic content profile
// on the E-ARK SIP profile of its era, writing dc+schema.xml from Terms.
// The registry in profiles/ hands it out under the name "basic".
var Definition = build.Definition{
	Name:        "basic",
	Descriptive: standard{},
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
	Schemas: slices.Concat(mets.Schemas, Schemas),
}

// standard is the dc+schema document as the engine sees it: it accepts
// Terms, writes them with Encode, and swaps the entity identifier in.
type standard struct{}

func (standard) Check(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not meemoo dc+schema terms (meemoo.Terms)", d)
	}
	return nil
}

func (standard) Encode(w io.Writer, d sip.Description, schemas string) error {
	return Encode(w, d.(Terms), schemas) // Check ran before anything else
}

// Swap implements build.IdentifierSwapper. meemoo SIP 1.2 links
// dc+schema.xml to the PREMIS object by a shared UUID: the document carries
// the entity identifier, and the producer's own identifier travels as a
// MEEMOO-LOCAL-ID object identifier. The terms hold one identifier slot, so
// the producer's value is read before the swap overwrites it.
func (standard) Swap(d sip.Description, id string) string {
	terms := d.(Terms) // Check ran before anything else
	local := terms.LocalIdentifier()
	terms.SetObjectIdentifier(id)
	return local
}
