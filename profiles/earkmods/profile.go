package earkmods

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the eark-mods profile: the eark profile's plain E-ARK SIP
// (spec 2.2.0) for RODA-class repositories, writing mods.xml from a Record
// where eark writes dc.xml from Simple Dublin Core terms, and no PREMIS.
// Every other value is eark's. The registry in profiles/ hands it out under
// the name "eark-mods".
var Definition = build.Definition{
	Name:    "eark-mods",
	Encoder: mods{},
	// Named after the MODS document it holds.
	DescriptiveName: "mods.xml",
	// No PREMIS, as for eark: RODA drops package PREMIS that does not
	// describe agents or events.
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
		Type:                     "Mixed", // CSIP content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
		ContentInformationType:   "MIXED", // package METS value; RODA reads it as the AIP type
		DescriptiveMDType:        "MODS",
		DescriptiveMDTypeVersion: "3.7",
		// Only the software agent; WithSubmitter appends the
		// submitting organization.
		Agents: []sip.Agent{
			{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1", NoteType: "SOFTWARE VERSION"},
		},
	},
}
