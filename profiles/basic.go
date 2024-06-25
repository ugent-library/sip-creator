package profiles

import (
	"fmt"

	"github.com/ugent-library/sip-creator/structure"
)

func (p *Profile) Basic() {
	// Create skeleton
	packageDirs := []string{
		fmt.Sprintf("%s/metadata/descriptive", p.BaseDir),
		fmt.Sprintf("%s/metadata/preservation", p.BaseDir),
		fmt.Sprintf("%s/representations", p.BaseDir),
	}

	for _, pd := range packageDirs {
		createDir(pd)
	}

	// Step 1: Compose and parse input

	// Create a new package
	pkg := structure.NewPackage()

	// Create an entity & a representation
	entity := p.createIntellectualEntity(fmt.Sprintf("%s/dc+schema.json", p.InDir))

	// representation := p.createRepresentation(fmt.Sprintf("%s/representation_1", p.InDir), "representation_1")

	p.eachDirectory(func(dir string, r *structure.Representation) {
		p.eachFile(dir, r.Label, func(f *structure.File) {
			r.AddFile(f)
		})

		entity.AddRepresentation(r)
	})

	// entity.AddRepresentation(representation)
	pkg.AddRootEntity(entity)

	// Step 2: Generate metadata files

	// Generate package description file
	// p.createDescriptionFile(fmt.Sprintf("%s/metadata/descriptive/dc+schema.xml", p.BaseDir), entity)

	// Generate package premis file
	// p.createPremisPackage(fmt.Sprintf("%s/metadata/preservation/premis.xml", p.BaseDir), pkg, entity)

	// Iterate over representations
	// entity.EachRepresentation(func(r *structure.Representation) error {
	// 	p.createPremisRepresentation(fmt.Sprintf("%s/representations/%s/metadata/preservation/premis.xml", p.BaseDir, r.Label), r, entity)

	// 	p.createMetsRepresentation(fmt.Sprintf("%s/representations/%s/mets.xml", p.BaseDir, r.Label), r)
	// 	return nil
	// })

	// // Generate package mets file
	// p.createMetsPackage(fmt.Sprintf("%s/mets.xml", p.BaseDir), pkg)
}
