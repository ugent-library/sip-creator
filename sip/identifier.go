package sip

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ValidateIdentifier returns why id is not a package-local identifier:
// every identifier in this model takes the form uuid-<uuid>.
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

type Identifier interface {
	GetType() string
	GetValue() string
}

type UUID struct {
	Type  string
	Value string
}

func NewUUID() *UUID {
	return &UUID{
		Type:  "uuid",
		Value: fmt.Sprintf("uuid-%s", uuid.New().String()),
	}
}

func (u *UUID) GetType() string {
	return u.Type
}

func (u *UUID) GetValue() string {
	return u.Value
}
