package eark

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the eark profile: a plain E-ARK SIP (spec 2.2.0) for
// RODA-class repositories, writing dc.xml from Terms and no PREMIS. The
// registry in profiles/ hands it out under the name "eark".
var Definition = build.Definition{
	Name:  "eark",
	Model: simpledc{},
	// Named after the simple-DC document it holds; Meemoo's naming
	// convention doesn't apply to the eark profile.
	DocumentName: "dc.xml",
	// A representation may carry its own description, such as a license
	// that holds for one version only; CSIP has no rule against it.
	AllowRepresentationDescriptions: true,
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
		ProfileURL:             "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		Type:                   "Mixed", // CSIP content-category vocabulary
		ContentInformationType: "MIXED", // package METS value; RODA reads it as the AIP type
		// Only the software agent; WithSubmitter appends the
		// submitting organization.
		Agents: []sip.Agent{
			{Role: "CREATOR", Type: "OTHER", OtherType: "SOFTWARE", Name: "SIP creator", Note: "0.1", NoteType: "SOFTWARE VERSION"},
		},
	},
}
