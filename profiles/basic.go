package profiles

import (
	"fmt"

	"github.com/ugent-library/sip-creator/metadata"
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
	pkg := metadata.NewPackage()

	// Create an entity & a representation
	entity := p.createIntellectualEntity(fmt.Sprintf("%s/dc+schema.json", p.InDir), fmt.Sprintf("%s/metadata/descriptive/dc+schema.xml", p.BaseDir))
	representation := p.createRepresentation(fmt.Sprintf("%s/representation_1", p.InDir), "representation_1")
	entity.AddRepresentation(representation)
	pkg.AddRootEntity(entity)

	// Step 2: Generate metadata files

	// Generate package description file
	p.createDescriptionFile(fmt.Sprintf("%s/metadata/descriptive/dc+schema.xml", p.BaseDir), pkg, entity)

	// Generate package premis file
	p.createPremisPackage(fmt.Sprintf("%s/metadata/preservation/premis.xml", p.BaseDir), pkg, entity)

	// Iterate over representations
	entity.EachRepresentation(func(r *metadata.Representation) error {
		p.createPremisRepresentation(fmt.Sprintf("%s/representations/%s/metadata/preservation/premis.xml", p.BaseDir, r.Label), r, entity)

		p.createMetsRepresentation(fmt.Sprintf("%s/representations/%s/mets.xml", p.BaseDir, r.Label), r)
		return nil
	})

	// Generate package mets file
	p.createMetsPackage(fmt.Sprintf("%s/mets.xml", p.BaseDir), pkg)
}
