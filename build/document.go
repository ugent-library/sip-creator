package build

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// DescriptiveDocument is a finished descriptive document supplied as a
// file, such as a dc.xml or mods.xml prepared elsewhere. The package
// carries an unchanged copy of it (ADR-0021). The two eark profiles accept
// one; the basic profile does not, because meemoo's document must carry
// the entity identifier the build mints, and the tool does not edit XML.
// The tool checks only that the file parses as XML and that its root
// element is the profile's standard. Schema validity and content are left
// to the producer and to the validators downstream (ADR-0003).
type DescriptiveDocument struct {
	// Source is the absolute path of the file on disk.
	Source string
}

var _ sip.Description = DescriptiveDocument{}

// Validate reports why the file is not a document at all: unreadable, or
// not parsable as XML (xmldoc.Root). Which root element it must have is
// the profile's rule, checked through its DescriptiveDocumentChecker.
func (d DescriptiveDocument) Validate() error {
	_, err := d.Root()
	return err
}

// ValidateRequired reports nothing: no profile that accepts a document
// reads an identifier or a title from it, and the document's schema says
// what it must state.
func (DescriptiveDocument) ValidateRequired() error { return nil }

// Root reads the whole file with xmldoc.Root and returns its root
// element for the profile's check of the standard. A build reads the
// file once per check and once more to copy it: descriptive documents are
// small, and rereading keeps the type a plain value with nothing cached.
func (d DescriptiveDocument) Root() (xml.StartElement, error) {
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
