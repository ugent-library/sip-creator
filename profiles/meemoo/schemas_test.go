package meemoo

import (
	"testing"
)

// Every schema the dc+schema document points at, and every schema those
// import by relative path, is bundled, so a package can ship the set.
func TestSchemasBundled(t *testing.T) {
	for _, s := range (dcschema{}).Schemas() {
		if len(s.Content) == 0 {
			t.Errorf("%s is not in the schema bundle", s.Name)
		}
	}
}
