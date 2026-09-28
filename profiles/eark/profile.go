package eark

import (
	"fmt"
	"io"
	"slices"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the eark profile: a plain E-ARK SIP (spec 2.2.0) for
// RODA-class repositories, writing dc.xml from Terms and no PREMIS. The
// registry in profiles/ hands it out under the name "eark".
var Definition = build.Definition{
	Name:        "eark",
	Descriptive: standard{},
	// Named after the simple-DC document it holds; meemoo's naming
	// convention doesn't apply to the eark profile.
	DescriptiveName: "dc.xml",
	// The eark profile emits no PREMIS: RODA drops package PREMIS that
	// does not describe agents or events.
	EmitPackagePremis:        false,
	EmitRepresentationPremis: false,
	// RODA shows each representation's type from the representation
	// METS's content typing (ADR-0013).
	EmitRepresentationType: true,
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
	// What the METS documents and dc.xml point at, plus the meemoo
	// dc+schema schemas, which no eark document references: they ship so
	// the output stays as it was, and dropping them is a deliberate output
	// change to make on its own.
	Schemas: slices.Concat(mets.Schemas, Schemas,
		[]string{"descriptive_basic.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd"}),
}

// standard is the simpledc document as the engine sees it: it accepts
// Terms and writes them with Encode. It never swaps: dc.xml keeps the
// producer's identifier, because CSIP has no rule tying it to the package
// identifier and the ingesting catalogue indexes dc.xml, so operators find
// the package by the identifier they know (ADR-0012).
type standard struct{}

func (standard) Check(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (eark.Terms)", d)
	}
	return nil
}

func (standard) Encode(w io.Writer, d sip.Description, schemas string) error {
	return Encode(w, d.(Terms), schemas) // Check ran before anything else
}
