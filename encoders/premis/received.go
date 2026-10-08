package premis

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
)

// Namespace is the PREMIS 3 XML namespace. The generated documents declare
// it, and received documents must declare it.
const Namespace = "http://www.loc.gov/premis/v3"

// ValidateReceived parses r as XML and checks that the root element is
// premis:premis in the PREMIS 3 namespace. It returns an error if this
// check fails. It does not validate against the PREMIS schema (ADR-0003).
func ValidateReceived(r io.Reader) error {
	root, err := xmldoc.Root(r)
	if err != nil {
		return err
	}
	if root.Name.Space != Namespace || root.Name.Local != "premis" {
		return fmt.Errorf("root element is {%s}%s, expected a premis:premis document in the PREMIS 3 namespace (%s)",
			root.Name.Space, root.Name.Local, Namespace)
	}
	return nil
}
