package profiles

import (
	"fmt"

	"github.com/ugent-library/sip-creator/structure"
)

func (p *Profile) Basic() {
	// Step 1: Compose and parse input

	// Create skeleton
	pkg := p.createPackage()

	// Create an entity & associate with a description file
	e := p.createIntellectualEntity(fmt.Sprintf("%s/dc+schema.json", p.InDir))

	// Loop over all "representation_*" directories and parse them
	//   in other profiles, we may require custom logic to hook a representation to a specific
	//   sub-entity.
	p.eachDirectory(func(dir string, r *structure.Representation) {
		p.eachFile(dir, r.Label, func(f *structure.File) {
			f.SetRepresentation(r)
			r.AddFile(f)
		})

		// in other profiles, here we might create dedicated dc+schema, dc or mods
		// files on a representation level, overriding the package level metadata (e.g. licenses)

		r.SetEntity(e)
		e.AddRepresentation(r)
	})

	pkg.AddRootEntity(e)

	// Step 2: Generate metadata files

	pkg.Root.EachRepresentation(func(r *structure.Representation) error {
		pr := p.generateRepresentationPremis(r)
		r.AddPremisFile(pr)

		mts := p.generateRepresentationMets(r)
		r.AddMetsFile(mts)

		return nil
	})

	pr := p.generatePackagePremis(pkg.Root)
	pkg.AddPremisFile(pr)

	mts := p.generatePackageMets(pkg)
	pkg.AddMetsFile(mts)
}
