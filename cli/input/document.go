package input

import "github.com/ugent-library/sip-creator/build"

// DocumentSpec describes the descriptive document an input folder may
// supply in place of description.csv, as the profile defines it: the file
// name the document must have, and the metadata model that judges its
// root. It describes the document, it is not one. When the model takes
// supplied documents (implements build.DocumentFormat), as the two eark
// profiles' models do, a file with that name is read as the level's
// description. When the model takes none, as basic's does, a file with
// that name is a violation. Other names are content like any other file:
// a dc.xml under meemoo/basic, or the zero DocumentSpec's empty name.
type DocumentSpec struct {
	// Name is the file name of the document, such as dc.xml or mods.xml:
	// the profile's build.Definition.DocumentName.
	Name string
	// Model is the profile's metadata model: build.Definition.Model.
	Model build.MetadataModel
}
