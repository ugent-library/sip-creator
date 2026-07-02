package profiles

import (
	"fmt"
	"log/slog"

	"github.com/ugent-library/sip-creator/sip"
)

func (p *Profile) Roda() *sip.Package {
	p.Logger.Info("starting...")
	// Step 1: Compose and parse input

	// Create skeleton
	pkg := p.createPackage()

	p.Logger.Info("created a new package", slog.Any("id", pkg.Identifier))

	// Create an entity & associate with a description file
	e := p.createIntellectualEntity()

	p.Logger.Info("created an intellectual entity", slog.Any("id", e.Identifier))

	src := fmt.Sprintf("%s/dc.json", p.InDir)
	dest := fmt.Sprintf("%s/metadata/descriptive", p.BaseDir)
	f := p.createDescriptiveFile(src, dest, func(d Description) {
		d.SetObjectIdentifier(e.Identifier)
	})

	p.Logger.Info("created a descriptive file", slog.Any("id", f.Identifier))

	e.AddDescriptionFile(f)

	// Loop over all "representation_*" directories and parse them
	//   in other profiles, we may require custom logic to hook a representation to a specific
	//   sub-entity.
	p.eachDirectory(func(dir string, r *sip.Representation) {
		p.Logger.Info("created a representation", slog.Any("id", r.Identifier))
		p.eachEssenceFile(dir, r.Label, func(f *sip.File) {
			p.Logger.Info("placed an essence file", slog.Any("id", f.Identifier))
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

	pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		mts := p.generateRepresentationMets(r)
		r.AddMetsFile(mts)
		p.Logger.Info("created a representation METS file", slog.Any("id", mts.Identifier))

		return nil
	})

	mts := p.generatePackageMets(pkg)
	pkg.AddMetsFile(mts)
	p.Logger.Info("created a package METS file", slog.Any("id", mts.Identifier))

	p.Logger.Info("finished.")

	return pkg
}
