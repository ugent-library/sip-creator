package build

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// EncodedDescription is a description supplied as a finished document in
// the model's document format, such as a dc.xml or mods.xml prepared
// elsewhere. The package carries an unchanged copy of the file
// (ADR-0021). A profile accepts one only when its metadata model
// implements DocumentFormat. Package build checks only that the file
// parses as XML and that its root element is in the profile's format. It
// does not validate the file against its schema or check its content
// (ADR-0003).
type EncodedDescription struct {
	// Source is the absolute path of the file on disk.
	Source string
}

var _ sip.Description = EncodedDescription{}

// Validate reads the file and checks that it parses as XML. It returns an
// error if the file cannot be read or is not XML. The metadata model's
// ValidateDocumentRoot checks the root element.
func (d EncodedDescription) Validate() error {
	_, err := d.Root()
	return err
}

// ValidateRequired accepts every supplied document and returns nil.
// Package build does not read a document's content (ADR-0003), so it
// cannot tell whether the document states an identifier and a title.
func (EncodedDescription) ValidateRequired() error { return nil }

// Root reads the whole file with xmldoc.Root and returns its root element.
// It returns an error if the file cannot be opened or is not XML.
func (d EncodedDescription) Root() (xml.StartElement, error) {
	f, err := os.Open(d.Source)
	if err != nil {
		return xml.StartElement{}, fmt.Errorf("supplied descriptive document: %w", err)
	}
	defer f.Close()
	root, err := xmldoc.Root(f)
	if err != nil {
		return xml.StartElement{}, fmt.Errorf("supplied descriptive document %s: %w", d.Source, err)
	}
	return root, nil
}
