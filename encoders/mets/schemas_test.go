package mets

import (
	"testing"

	"github.com/ugent-library/sip-creator/schemas"
)

// Every schema the METS templates point at is bundled, so a package can
// ship it.
func TestSchemasBundled(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Schemas {
		if _, ok := bundle[name]; !ok {
			t.Errorf("%s is not in the schema bundle", name)
		}
	}
}
