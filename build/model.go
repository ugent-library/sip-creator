package build

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// MetadataModel is a profile's metadata model as package build sees it.
// It names the description type that holds a description in the model,
// and writes such a description as a document in the model's document
// format. The fields of a description and the rules they follow belong to
// the description type and its Validate. A profile written outside this
// module implements MetadataModel for its own model (ADR-0022).
type MetadataModel interface {
	// ValidateType checks that d is a value of the model's description
	// type. It returns an error if d has another type. Encode and Swap
	// receive only a description that passed this check, so they may
	// assert its type.
	ValidateType(d sip.Description) error
	// Encode writes d as a document in the model's document format.
	// schemasDir is the path of the package's schemas/ directory relative
	// to the document, for the document's schema-location hint. The XSD the
	// hint names must be one that Schemas lists. d has passed ValidateType
	// and its own Validate, so Encode need not check it again. When Encode
	// fails, it writes nothing to w.
	Encode(w io.Writer, d sip.Description, schemasDir string) error
	// Schemas lists the XSDs the encoded document points at, plus what
	// those import by relative path, each with its contents. The package
	// ships them under schemas/ next to the ones the METS documents point
	// at, also when the description is a supplied document. List nothing
	// else. A profile outside this module embeds its own XSDs and returns
	// them here. BundledSchemas returns the XSDs this module bundles.
	Schemas() []Schema
	// ModelType returns the name of the document's format, such as DC,
	// MODS or EBUCore. A name the METS MDTYPE vocabulary lists, spelled as
	// the vocabulary spells it, is recorded as MDTYPE. Any other name is
	// recorded as MDTYPE OTHER with the name in OTHERMDTYPE. Together with
	// ModelTypeVersion it describes both the document Encode writes and the
	// document a supplied file must be. It belongs to the model rather than
	// to the profile's METS values, so a profile cannot pair a model with
	// another model's type.
	ModelType() string
	// ModelTypeVersion returns the version of the model the document
	// follows, such as 3.7 for MODS. METS records it in MDTYPEVERSION. It
	// returns an empty string when the model has no version to name.
	ModelTypeVersion() string
}

// Schema is one XSD file a package ships under schemas/.
type Schema struct {
	// Name is the file name under schemas/, such as mods-3-7.xsd. It is a
	// plain file name without directories, because every schema lands
	// directly in schemas/. The documents' schema-location hints point at
	// it by this name.
	Name string
	// Content is the file's bytes, written into the package as they are.
	// It must not be empty.
	Content []byte
}

// BundledSchemas returns the XSDs this module bundles under the given file
// names, such as dc.xsd or xml.xsd, for a metadata model to return from
// Schemas. A name the bundle does not hold comes back with empty Content,
// and Builder.Build refuses such a schema before it writes anything.
func BundledSchemas(names ...string) []Schema {
	bundle := schemas.Get()
	list := make([]Schema, 0, len(names))
	for _, name := range names {
		list = append(list, Schema{Name: name, Content: bundle[name]})
	}
	return list
}

// IdentifierSwapper is the optional part of a MetadataModel whose
// standard links descriptive and preservation metadata by a shared
// identifier, as Meemoo's does. Under a model without it, the document
// keeps the producer's identifier (ADR-0012).
type IdentifierSwapper interface {
	// Swap replaces the identifier in d with id and returns the producer's
	// identifier it replaced. Builder.Build records the identifier Swap
	// returns for the package description as the entity's MEEMOO-LOCAL-ID.
	Swap(d sip.Description, id string) (local string)
}

// DocumentFormat is the optional part of a MetadataModel that accepts a
// finished document supplied as a file (EncodedDescription), next to its
// own description type. A model without it accepts no supplied document
// (ADR-0021).
type DocumentFormat interface {
	// ValidateDocumentRoot checks that root, the root element of a
	// supplied document, is a document in the model's format: the right
	// element and namespace, and the version the METS declares. It returns
	// an error if it is not.
	ValidateDocumentRoot(root xml.StartElement) error
}

// checkDescriptions checks the package's description and each
// representation's description with checkDescription. It returns the
// first error. A missing package description is left to
// SourcePackage.Validate.
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

// checkDescription checks that d is a description in the model. A value
// of a description type goes to the model's ValidateType. For a supplied
// document, it reads the root element and passes it to the model's
// ValidateDocumentRoot. A model without DocumentFormat accepts no supplied
// document. It returns an error if the document cannot be read or a check
// fails.
func checkDescription(model MetadataModel, d sip.Description) error {
	doc, ok := d.(EncodedDescription)
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
