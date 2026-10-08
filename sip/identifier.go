package sip

import (
	"fmt"
	"strings"
	"uuid"
)

// ValidateIdentifier checks that id takes the form uuid-<uuid>, the form of
// every identifier in this model. It returns an error if it does not. The
// prefix makes a UUID valid as a METS @ID: an xsd:ID may not start with a
// digit, and an unprefixed UUID can.
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
