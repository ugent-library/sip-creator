package build

import "github.com/ugent-library/sip-creator/sip"

// Assemble exposes the assembly phase to the external tests, which inspect
// the graph a build declares without writing anything.
func (b *Builder) Assemble(def Definition, in *Input) (*sip.Package, error) {
	return b.assemble(def, in)
}
