package input

import (
	"os"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// readDocument reads the supplied descriptive document at src as the
// level's description: a file the package copies as it is. It must parse
// as XML (xmldoc.Root) with the profile's standard as its root, which the
// vocabulary judges. Nothing else in the document is checked (ADR-0003):
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
	if err := w.document.ValidateDocumentRoot(root); err != nil {
		w.violate("%s: %v", rel, err)
	}
	return build.EncodedDescription{Source: src}
}
