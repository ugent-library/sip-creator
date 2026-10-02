package input

import (
	"os"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// Document is the profile's descriptive document as the input folder may
// supply it in place of description.csv: the file name the package gives
// the document, and the metadata model that judges a supplied one. The
// name is reserved, and a supplied document read, only when the model
// takes supplied documents (implements build.DocumentFormat), as the two
// eark profiles' models do. The zero Document takes none: a dc.xml under
// basic is content like any other file.
type Document struct {
	// Name is the file name of the document, such as dc.xml or mods.xml:
	// the profile's build.Definition.DocumentName.
	Name string
	// Model is the profile's metadata model: build.Definition.Model.
	Model build.MetadataModel
}

// readDocument reads the supplied descriptive document at src as the
// level's description: a file the package copies as it is. It must parse
// as XML (xmldoc.Root) with a root in the model's document format, which
// the model judges as the engine will. Nothing else in the document is checked (ADR-0003):
// schema validity is left to the validators downstream.
func (w *folderWalker) readDocument(src string) sip.Description {
	rel := w.rel(src)

	f, err := os.Open(src)
	if err != nil {
		w.violate("%s: %v", rel, err)
		return nil
	}
	defer f.Close()

	root, err := xmldoc.Root(f)
	if err != nil {
		w.violate("%s: %v", rel, err)
		return nil
	}
	if err := w.documentFormat.ValidateDocumentRoot(root); err != nil {
		w.violate("%s: %v", rel, err)
	}
	return build.EncodedDescription{Source: src}
}
