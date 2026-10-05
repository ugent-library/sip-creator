package input

import "github.com/ugent-library/sip-creator/build"

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
