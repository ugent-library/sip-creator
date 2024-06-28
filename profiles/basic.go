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
	e := p.createIntellectualEntity()

	src := fmt.Sprintf("%s/dc+schema.json", p.InDir)
	dest := fmt.Sprintf("%s/metadata/descriptive", p.BaseDir)
	file := p.createDescriptiveFile(src, dest, func(d Description) {
		d.SetObjectIdentifier(e.Identifier)
		// TODO This could be auto-detected based off the salience of the source metadata
		//   e.g. metadata properties which describe a group of additional identifiers.
		localId := d.GetLocalIdentifier("dcterms")
		e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", localId)
	})

	e.AddDescriptionFile(file)

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
		// Use p.createDescriptiveFile to create those rep level descriptive files

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
