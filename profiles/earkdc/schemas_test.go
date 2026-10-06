package eark

import (
	"testing"
)

// Every schema the simpledc document points at is bundled, so a package
// can ship it.
func TestSchemasBundled(t *testing.T) {
	for _, s := range (simpledc{}).Schemas() {
		if len(s.Content) == 0 {
			t.Errorf("%s is not in the schema bundle", s.Name)
		}
	}
}
