package build

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// DescriptiveDocument is a finished descriptive document supplied as a file, a dc.xml
// or mods.xml prepared elsewhere, copied into the package as it is
// (ADR-0021). It is a Description for the profiles whose encoder is also a
// DescriptiveDocumentChecker, the two eark profiles. The basic profile takes none:
// meemoo's document must carry the entity identifier the build mints, and
// the tool does not edit XML. The tool checks what the package's integrity
// needs and nothing more: the file parses as XML (Validate) and its root
// element is the profile's standard (the engine reads the root, the
// profile's CheckDescriptiveDocument judges it). The writer copies the file with the
// store's streamed copy, fixity computed on the way, as it copies essence.
// Validity against the schema and the document's content are the
// producer's, and the validators downstream check them (ADR-0003).
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

// ValidateRequired trusts the document. Nothing in the profiles that
// accept one reads an identifier or a title back out of it, and what a
// document must state is its schema's business, not the tool's.
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
