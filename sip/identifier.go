package sip

import (
	"fmt"

	"github.com/google/uuid"
)

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
