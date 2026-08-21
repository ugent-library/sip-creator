package premis

import (
	"encoding/xml"
	"fmt"
	"io"
)

// Namespace is the PREMIS 3 XML namespace: the one the generated
// documents declare and received documents must declare.
const Namespace = "http://www.loc.gov/premis/v3"

// ValidateReceived reports why r is not acceptable received preservation
// metadata (input spec §5): it must be well-formed XML whose root element
// is premis:premis in the PREMIS 3 namespace. Deliberately not schema
// validation, which stays external (ADR-0003); this check only keeps the
// tool from packaging something that is not a PREMIS document at all.
func ValidateReceived(r io.Reader) error {
	dec := xml.NewDecoder(r)

	var root *xml.StartElement
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("not well-formed XML: %v", err)
		}
		if start, ok := tok.(xml.StartElement); ok && root == nil {
			root = &start
			// Keep reading: well-formedness of the whole document matters,
			// not just the prologue.
		}
	}

	switch {
	case root == nil:
		return fmt.Errorf("not an XML document")
	case root.Name.Space != Namespace || root.Name.Local != "premis":
		return fmt.Errorf("root element is {%s}%s, expected a premis:premis document in the PREMIS 3 namespace (%s)",
			root.Name.Space, root.Name.Local, Namespace)
	}
	return nil
}
