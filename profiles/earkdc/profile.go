// Package earkdc is the eark/dc profile: a plain E-ARK SIP whose
// descriptive standard is Simple Dublin Core.
package earkdc

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the eark/dc profile: a plain E-ARK SIP (spec 2.2.0), writing
// dc.xml from simpledc.Terms and no PREMIS. The registry in profiles/ hands
// it out under the name "eark/dc".
var Definition = build.Definition{
	Name:  "eark/dc",
	Model: simpledc.Model,
	// Named after the simple-DC document it holds; Meemoo's naming
	// convention doesn't apply to the eark/dc profile.
	DocumentName: "dc.xml",
	// A representation may carry its own description, such as a license
	// that holds for one version only; CSIP has no rule against it.
	AllowRepresentationDescriptions: true,
	// No PREMIS: E-ARK SIP makes it optional, and without agents or events
	// a PREMIS document only repeats the fixity the METS already declares
	// (ADR-0022).
	EmitPackagePremis:        false,
	EmitRepresentationPremis: false,
	// Each representation's type goes into its METS content typing, where
	// an ingest system reads it as the representation's type (ADR-0013).
	EmitRepresentationType: true,
	Declaration: sip.MetsDeclaration{
		// The version-pinned profile URL: commons-ip's SIP2 check for
		// spec 2.2.0 compares against this exact value (its error
		// message misleadingly prints the unversioned URL).
		ProfileURL:             "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		Type:                   "Mixed", // CSIP content-category vocabulary
		ContentInformationType: "MIXED", // CSIP content information type; the package METS value
		// No agents: the engine adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}
