package build

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/sip"
)

// DescriptionEncoder is what a profile plugs into the engine for its
// descriptive metadata: which description type it accepts and how it
// writes the document for it. Each profile package under profiles/
// implements it for its own description type, so everything that needs the
// concrete type stays there and the engine speaks sip.Description only. A
// caller with its own profile implements it too (ADR-0022).
// Building a description is not the encoder's business: callers construct
// the profile's type themselves.
type DescriptionEncoder interface {
	// Check returns why d is not a description this encoder takes: a
	// description of another type. It runs before validation and before
	// any write, and guarantees the type assertions the encoder's other
	// methods make.
	Check(d sip.Description) error
	// Encode writes d as the profile's descriptive document. schemasDir is
	// the path of the package's schemas/ directory relative to the document
	// being written, for the document's schema-location hint; only the
	// writer knows where a document lands. Which XSD file the hint names is
	// the encoder's own, and Schemas lists it.
	Encode(w io.Writer, d sip.Description, schemasDir string) error
	// Schemas lists the bundled XSD file names the encoded document points
	// at, plus what those import by relative path. The package ships them
	// under schemas/ next to the ones the METS documents point at, and
	// nothing else: an XSD no document references is noise to whoever
	// reads the package later.
	Schemas() []string
}

// IdentifierSwapper is the optional part of a DescriptionEncoder whose
// spec links descriptive and preservation metadata by a shared identifier
// (meemoo's). Swap replaces the identifier in d with id and returns the
// producer's identifier it replaced; the assembler records that as the
// entity's MEEMOO-LOCAL-ID. An encoder without it keeps the producer's
// identifier in the document (ADR-0012).
type IdentifierSwapper interface {
	Swap(d sip.Description, id string) (local string)
}

// DescriptiveDocumentChecker is the optional part of a DescriptionEncoder whose
// profile takes a supplied descriptive document (DescriptiveDocument) next to its own
// description type. CheckDescriptiveDocument returns why root, the document's root
// element as the engine read it, is not the profile's standard: another
// element or namespace, or a version other than the one the METS declares.
// An encoder without it takes no supplied document; meemoo's is one,
// because its document must carry the entity identifier the build mints,
// which Swap writes into terms (ADR-0021).
type DescriptiveDocumentChecker interface {
	CheckDescriptiveDocument(root xml.StartElement) error
}

// checkDescriptions returns why a description in the source package is not one
// the encoder takes, the package's or a representation's. A missing
// package description is SourcePackage.Validate's finding, not this check's.
func checkDescriptions(enc DescriptionEncoder, source *SourcePackage) error {
	if source.Description != nil {
		if err := checkDescription(enc, source.Description); err != nil {
			return err
		}
	}
	for _, r := range source.Representations {
		if r.Description == nil {
			continue
		}
		if err := checkDescription(enc, r.Description); err != nil {
			return fmt.Errorf("representation %q: %w", r.Name, err)
		}
	}
	return nil
}

// checkDescription returns why one description is not one the encoder
// takes: a model of another type, or a supplied document for a profile that
// takes none or whose root is another standard's. Reading the document is
// the engine's; the encoder sees the root element only.
func checkDescription(enc DescriptionEncoder, d sip.Description) error {
	doc, ok := d.(DescriptiveDocument)
	if !ok {
		return enc.Check(d)
	}
	checker, ok := enc.(DescriptiveDocumentChecker)
	if !ok {
		return fmt.Errorf("a supplied descriptive document is not accepted: this profile takes its own description type only")
	}
	root, err := doc.Root()
	if err != nil {
		return err
	}
	if err := checker.CheckDescriptiveDocument(root); err != nil {
		return fmt.Errorf("supplied descriptive document %s: %w", doc.Source, err)
	}
	return nil
}
