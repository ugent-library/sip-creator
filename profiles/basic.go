package profiles

import (
	"github.com/ugent-library/sip-creator/sip"
)

// Basic builds a meemoo SIP 2.0 basic-profile package. It survives only
// until the CLI resolves profiles from the registry itself (plan Step 9).
func (b *Builder) Basic() (*sip.Package, error) {
	def, _ := Get("basic") // registered in this package; cannot miss
	return b.Build(def)
}
