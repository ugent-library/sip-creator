// Package xmldoc is the one XML reader the tool has. It reads a whole
// document and returns its root element. Every check on a document the
// tool did not write, such as a received PREMIS file or a supplied
// descriptive document, reads it this way, so all of them accept the same
// XML. It does not validate against a schema (ADR-0003).
package xmldoc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

// Root reads r to its end and returns the root element with its
// attributes. It returns a "not well-formed XML" error for bad syntax and
// for unclosed or mismatched tags. It returns a "not an XML document" error
// for input without any element. Like encoding/xml, it accepts a second
// top-level element and text after the root.
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
			// Keep reading: the whole document must be well-formed, not
			// only the part up to the root's start tag.
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
