package sip

import (
	"fmt"
	"strings"
	"uuid"
)

// ValidateIdentifier returns why id is not a package-local identifier:
// every identifier in this model takes the form uuid-<uuid>. The prefix
// makes a UUID valid as a METS @ID (an xsd:ID may not start with a
// digit; an unprefixed UUID can).
func ValidateIdentifier(id string) error {
	rest, ok := strings.CutPrefix(id, "uuid-")
	if !ok {
		return fmt.Errorf("identifier %q does not take the uuid-<uuid> form", id)
	}
	if _, err := uuid.Parse(rest); err != nil {
		return fmt.Errorf("identifier %q does not carry a valid UUID: %v", id, err)
	}
	return nil
}
