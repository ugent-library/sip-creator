// Package earkmods is the eark/mods profile: a plain E-ARK SIP whose
// descriptive standard is MODS 3.7.
package earkmods

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/sip"
)

// Definition is the eark/mods profile: the eark/dc profile's plain E-ARK SIP
// (spec 2.2.0), writing mods.xml from a mods.Record where eark/dc writes
// dc.xml from Simple Dublin Core terms, and no PREMIS.
// Every other value is eark/dc's. The registry in profiles/ hands it out under
// the name "eark/mods".
var Definition = build.Definition{
	Name:  "eark/mods",
	Model: mods.Model,
	// Named after the MODS document it holds.
	DocumentName: "mods.xml",
	// As for eark: a representation may carry its own description.
	AllowRepresentationDescriptions: true,
	// No PREMIS, as for eark: E-ARK SIP makes it optional, and without
	// agents or events a PREMIS document only repeats the fixity the METS
	// already declares (ADR-0022).
	EmitPackagePremis:        false,
	EmitRepresentationPremis: false,
	// As for eark: each representation's type goes into its METS content
	// typing, where an ingest system reads it (ADR-0013).
	EmitRepresentationType: true,
	Declaration: sip.MetsDeclaration{
		// The version-pinned profile URL: commons-ip's SIP2 check for
		// spec 2.2.0 compares against this exact value (its error
		// message misleadingly prints the unversioned URL).
		ProfileURL:             "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		Type:                   "Mixed", // CSIP content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
		ContentInformationType: "MIXED", // CSIP content information type; the package METS value
		// No agents: the engine adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}
