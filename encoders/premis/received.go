package premis

import (
	"fmt"
	"io"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
)

// Namespace is the PREMIS 3 XML namespace: the one the generated
// documents declare and received documents must declare.
const Namespace = "http://www.loc.gov/premis/v3"

// ValidateReceived reports why r is not acceptable received preservation
// metadata: it must parse as XML (xmldoc.Root) with premis:premis in the
// PREMIS 3 namespace as its root element. Deliberately not schema validation, which
// stays external (ADR-0003); this check only keeps the tool from packaging
// something that is not a PREMIS document at all.
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
