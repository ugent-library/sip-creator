package build

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/sip"
)

// MetadataModel is the profile's metadata model as the engine sees it:
// Meemoo's dc+schema.org, Simple Dublin Core or MODS 3.7. It names the
// description type that holds a description in the model, and writes such
// a description as a document in the model's document format. Which fields
// the model has and what rules they follow live on the description type
// (its Validate), not here. Each profile package under profiles/
// implements it for its own description type, so everything that needs the
// concrete type stays there and the engine speaks sip.Description only. A
// profile written outside this module, for another model, implements it
// the same way (ADR-0022). The model never builds a description: it
// arrives in the SourcePackage already built, as a value of the model's
// description type, and the model only checks its type and writes it.
type MetadataModel interface {
	// ValidateType returns why d is not a description in this model: a
	// description of another type. It runs before validation and before
	// any write, and guarantees the type assertions the model's other
	// methods make.
	ValidateType(d sip.Description) error
	// Encode writes d as a document in the model's document format.
	// schemasDir is the path of the package's schemas/ directory relative
	// to the document being written, for the document's schema-location
	// hint; only the writer knows where a document lands. Which XSD file
	// the hint names is the model's own, and Schemas lists it. ValidateType
	// and the description's Validate have run before Encode is called, and
	// Encode repeats neither. A failed render writes nothing to w.
	Encode(w io.Writer, d sip.Description, schemasDir string) error
	// Schemas lists the bundled XSD file names the encoded document points
	// at, plus what those import by relative path. The package ships them
	// under schemas/ next to the ones the METS documents point at, also
	// when the description is a supplied document. List nothing else: an
	// XSD no document references is noise to whoever reads the package
	// later.
	Schemas() []string
}

// IdentifierSwapper is the optional part of a MetadataModel whose
// spec links descriptive and preservation metadata by a shared identifier
// (Meemoo's). Swap replaces the identifier in d with id and returns the
// producer's identifier it replaced; the assembler records that as the
// entity's MEEMOO-LOCAL-ID. A model without it keeps the producer's
// identifier in the document (ADR-0012).
type IdentifierSwapper interface {
	Swap(d sip.Description, id string) (local string)
}

// DocumentFormat is the optional part of a MetadataModel that accepts a
// finished document supplied as a file (DescriptiveDocument), next to its
// own description type: it recognizes a supplied file as a document in the
// model's document format. Every model has a document format; only a model
// that takes supplied documents implements this.
// ValidateDocumentRoot returns why root, the document's root element, is
// not a document in the format: another element or namespace, or a
// version other than the one the METS declares. A model without it takes
// no supplied document. Meemoo's model does not implement it, because
// Meemoo's document must carry the entity identifier the build mints,
// which Swap writes into the terms (ADR-0021).
type DocumentFormat interface {
	ValidateDocumentRoot(root xml.StartElement) error
}

// checkDescriptions returns why a description in the source package, the
// package's or a representation's, is not a description in the model. A
// missing package description is SourcePackage.Validate's finding, not
// this check's.
func checkDescriptions(model MetadataModel, source *SourcePackage) error {
	if source.Description != nil {
		if err := checkDescription(model, source.Description); err != nil {
			return err
		}
	}
	for _, r := range source.Representations {
		if r.Description == nil {
			continue
		}
		if err := checkDescription(model, r.Description); err != nil {
			return fmt.Errorf("representation %q: %w", r.Name, err)
		}
	}
	return nil
}

// checkDescription returns why one description is not a description in the
// model: a value of another type, or a supplied document for a profile that
// takes none or whose root is another format's. Reading the document is
// the engine's; the model sees the root element only.
func checkDescription(model MetadataModel, d sip.Description) error {
	doc, ok := d.(DescriptiveDocument)
	if !ok {
		return model.ValidateType(d)
	}
	format, ok := model.(DocumentFormat)
	if !ok {
		return fmt.Errorf("a supplied descriptive document is not accepted: this profile takes its own description type only")
	}
	root, err := doc.Root()
	if err != nil {
		return err
	}
	if err := format.ValidateDocumentRoot(root); err != nil {
		return fmt.Errorf("supplied descriptive document %s: %w", doc.Source, err)
	}
	return nil
}
