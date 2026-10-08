// Package ugent holds the profiles UGent Library defines for its own RODA
// instance, with the metadata models they use. A profile is a content
// type, named after its owner and the type (ADR-0034). ugent/basic is for
// resources the library has not necessarily catalogued, described as
// Simple Dublin Core Terms. ugent/bibliographic is for its catalogued
// holdings, described as a MODS Record. Each profile's rules are written in
// docs/profiles/.
package ugent

import (
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Basic is the ugent/basic profile (docs/profiles/ugent-basic.md): a plain
// E-ARK SIP (spec 2.2.0), writing dc.xml from Terms and no PREMIS.
var Basic = build.Definition{
	Name:  "ugent/basic",
	Model: simpledc{},
	// Named after the Simple DC document it holds.
	DocumentName: "dc.xml",
	// docs/profiles/ugent-basic.md §4: "A representation's name MUST be
	// one of preservation, archival and access, exactly, in lowercase."
	// The name is also the representation's type.
	RepresentationTypes: []string{"preservation", "archival", "access"},
	// docs/profiles/ugent-basic.md §4: "A package MAY hold no
	// representation at all: a description of an intellectual entity whose
	// content is not, or not yet, in the archive." CSIP58 allows a package
	// without file references to content.
	MinRepresentations: 0,
	// docs/profiles/ugent-basic.md §3: "A representation MAY carry its own
	// description, in the same standard, for what is true of that
	// representation only, such as a license."
	AllowRepresentationDescriptions: true,
	// docs/profiles/ugent-basic.md §6: "The tool generates no PREMIS for
	// this profile. Without agents or events a generated PREMIS document
	// would only repeat the fixity the METS already declares." Supplied
	// PREMIS documents are still carried.
	EmitPackagePremis:        false,
	EmitRepresentationPremis: false,
	// Each representation's type goes into its METS content typing
	// (ADR-0013).
	EmitRepresentationType: true,
	Declaration: sip.MetsDeclaration{
		// The profile URL for E-ARK SIP 2.2.0. commons-ip's SIP2 check for
		// that version compares against this exact value, although its
		// error message prints the unversioned URL.
		ProfileURL: "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		// The default content category, from the CSIP vocabulary. A
		// package's SourcePackage.ContentCategory replaces it.
		Type: "Mixed",
		// The package METS declares the profile as its content type
		// (docs/profiles/ugent-basic.md §2: "The package METS MUST declare
		// ugent/basic as its content information type."). CSIP6: "When the
		// csip:CONTENTINFORMATIONTYPE has the value OTHER the
		// csip:OTHERCONTENTINFORMATIONTYPE must state the content
		// information type." A representation METS declares its type
		// instead (EmitRepresentationType).
		ContentInformationType:      "OTHER",
		OtherContentInformationType: "ugent/basic",
		// No agents: Builder.Build adds the software agent, and
		// WithSubmitter the submitting organization.
	},
}

// Bibliographic is the ugent/bibliographic profile
// (docs/profiles/ugent-bibliographic.md): Basic's plain E-ARK SIP, writing
// mods.xml from a Record where Basic writes dc.xml. Every other value
// is Basic's.
var Bibliographic = build.Definition{
	Name:         "ugent/bibliographic",
	Model:        mods{},
	DocumentName: "mods.xml",
	// docs/profiles/ugent-bibliographic.md §4: "A representation's name
	// MUST be one of preservation, archival and access, exactly, in
	// lowercase." The name is also the representation's type.
	RepresentationTypes: []string{"preservation", "archival", "access"},
	// As for Basic: a package may hold no representation at all
	// (docs/profiles/ugent-bibliographic.md §4).
	MinRepresentations: 0,
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
		ProfileURL: "https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml",
		Type:       "Mixed",
		// The package METS declares the profile as its content type
		// (docs/profiles/ugent-bibliographic.md §2: "The package METS MUST
		// declare ugent/bibliographic as its content information type."),
		// as CSIP6 asks for OTHER (see Basic).
		ContentInformationType:      "OTHER",
		OtherContentInformationType: "ugent/bibliographic",
	},
}
