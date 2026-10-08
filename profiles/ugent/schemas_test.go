package ugent

import (
	"testing"

	"github.com/ugent-library/sip-creator/build"
)

// Every schema the simpledc and the MODS document point at is bundled, so
// a package can ship it.
func TestSchemasBundled(t *testing.T) {
	for _, model := range []build.MetadataModel{simpledc{}, mods{}} {
		for _, s := range model.Schemas() {
			if len(s.Content) == 0 {
				t.Errorf("%s is not in the schema bundle", s.Name)
			}
		}
	}
}
