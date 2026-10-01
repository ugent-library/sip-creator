// Package xmldoc is the one XML reader the tool has. It reads a whole
// document and returns its root element, so the checks on documents the
// tool did not write (received PREMIS files, supplied descriptive
// documents) share one notion of "parses as XML, with this root".
// Deliberately not schema validation, which stays external (ADR-0003);
// the reader only keeps the tool from packaging something that is not an
// XML document of the expected kind at all.
package xmldoc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// Root reads r to its end and returns the document's first element, the
// root, with its attributes. It refuses what encoding/xml refuses: bad
// syntax and unclosed or mismatched tags, reported as "not well-formed
// XML", and input that holds no element at all, reported as "not an XML
// document". encoding/xml does not refuse a second top-level element or
// text after the root, so neither does Root.
func Root(r io.Reader) (xml.StartElement, error) {
	dec := xml.NewDecoder(r)

	var root *xml.StartElement
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return xml.StartElement{}, fmt.Errorf("not well-formed XML: %v", err)
		}
		if start, ok := tok.(xml.StartElement); ok && root == nil {
			root = &start
			// Keep reading: well-formedness of the whole document matters,
			// not just the prologue.
		}
	}
	if root == nil {
		return xml.StartElement{}, errors.New("not an XML document")
	}
	return *root, nil
}

// Attr returns the value of the root's attribute named local, outside any
// namespace, and whether the root carries it.
func Attr(root xml.StartElement, local string) (string, bool) {
	for _, a := range root.Attr {
		if a.Name.Space == "" && a.Name.Local == local {
			return a.Value, true
		}
	}
	return "", false
}
