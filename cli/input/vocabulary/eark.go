package vocabulary

import (
	"encoding/xml"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Eark is the eark profile's vocabulary: the fifteen Simple Dublin Core
// elements, the table in profiles/eark. Each statement becomes one term of
// eark.Terms unchanged; the terms' Validate decides which keys exist and
// what a statement may say. A finished dc.xml may stand in for the rows.
type Eark struct{}

var _ input.DocumentVocabulary = Eark{}

// Description wraps the statements as Simple Dublin Core terms in
// statement order.
func (Eark) Description(statements []input.Statement) (sip.Description, []error) {
	return eark.Terms(terms(statements)), nil
}

// DocumentName is the file name of a supplied Simple Dublin Core document:
// dc.xml, the name the package gives the document.
func (Eark) DocumentName() string {
	return eark.Definition.DescriptiveName
}

// CheckDocument returns why root is not a simpledc document, as the eark
// profile's encoder judges it.
func (Eark) CheckDocument(root xml.StartElement) error {
	return checkDocument(eark.Definition, root)
}
