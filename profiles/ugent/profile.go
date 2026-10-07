// Package ugent holds the profiles UGent Library defines for its own RODA
// instance. A profile is a content type, named after its owner and the
// type (ADR-0034): ugent/basic for resources the library has not
// necessarily catalogued, ugent/bibliographic for its catalogued holdings.
// Each profile's rules are written in docs/profiles/.
package ugent

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

// Basic is the ugent/basic profile (docs/profiles/ugent-basic.md): a plain
// E-ARK SIP (spec 2.2.0), writing dc.xml from simpledc.Terms and no PREMIS.
// The registry in profiles/ hands it out under the name "ugent/basic".
var Basic = build.Definition{
	Name:  "ugent/basic",
	Model: simpledc.Model,
	// Named after the Simple DC document it holds; Meemoo's naming
	// convention doesn't apply here.
	DocumentName: "dc.xml",
	// A representation may carry its own description, such as a license
	// that holds for one version only; CSIP has no rule against it.
	AllowRepresentationDescriptions: true,
	// No PREMIS: E-ARK SIP makes it optional, and without agents or events
	// a PREMIS document only repeats the fixity the METS already declares
	// (ADR-0022). Supplied PREMIS documents are still carried.
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
		Type:                   "Mixed", // CSIP content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
		ContentInformationType: "MIXED", // CSIP content information type; the package METS value
		// No agents: the engine adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}

// Bibliographic is the ugent/bibliographic profile
// (docs/profiles/ugent-bibliographic.md): Basic's plain E-ARK SIP, writing
// mods.xml from a mods.Record where Basic writes dc.xml. Every other value
// is Basic's. The registry in profiles/ hands it out under the name
// "ugent/bibliographic".
var Bibliographic = build.Definition{
	Name:  "ugent/bibliographic",
	Model: mods.Model,
	// Named after the MODS document it holds.
	DocumentName: "mods.xml",
	// As for Basic: a representation may carry its own description.
	AllowRepresentationDescriptions: true,
	// No PREMIS, as for Basic.
	EmitPackagePremis:        false,
	EmitRepresentationPremis: false,
	// As for Basic: each representation's type goes into its METS content
	// typing (ADR-0013).
	EmitRepresentationType: true,
	Declaration: sip.MetsDeclaration{
		// As for Basic.
		ProfileURL:             "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		Type:                   "Mixed", // CSIP content-category vocabulary; --content-category and SIP_CONTENT_CATEGORY override it
		ContentInformationType: "MIXED", // CSIP content information type; the package METS value
	},
}
