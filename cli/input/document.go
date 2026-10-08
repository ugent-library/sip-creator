package input

import "github.com/ugent-library/sip-creator/build"

// DocumentSpec describes the descriptive document a profile lets an input
// folder supply in place of description.csv: the file name the document
// must have, and the metadata model that checks its root element. When
// the model implements build.DocumentFormat, a file with that name is read
// as the level's description. When it does not, a file with that name is
// a violation. A document named for another profile is content, such as
// a dc.xml under meemoo/basic. The zero DocumentSpec has an empty name and
// reserves no name.
type DocumentSpec struct {
	// Name is the file name of the document, such as dc.xml or mods.xml:
	// the profile's build.Definition.DocumentName.
	Name string
	// Model is the profile's metadata model: build.Definition.Model.
	Model build.MetadataModel
}
