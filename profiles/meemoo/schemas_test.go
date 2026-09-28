package meemoo

import (
	"testing"

	"github.com/ugent-library/sip-creator/schemas"
)

// Every schema the dc+schema document points at, and every schema those
// import by relative path, is bundled, so a package can ship the set.
func TestSchemasBundled(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range (dcschema{}).Schemas() {
		if _, ok := bundle[name]; !ok {
			t.Errorf("%s is not in the schema bundle", name)
		}
	}
}
