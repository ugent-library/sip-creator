package profiles

import (
	"io"
	"log/slog"

	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// write emits pkg to disk in dependency order, back-filling fixity on File
// nodes as each file lands. The ordering is load-bearing and encoded only
// here: representation METS embeds the fixity of its PREMIS file, and
// package METS embeds the fixity of everything before it — strictly last.
func (p *Profile) write(st *store.Store, pkg *sip.Package) error {
	if err := p.writeSkeleton(st); err != nil {
		return err
	}
	if err := p.writeSchemas(st, pkg); err != nil {
		return err
	}
	if err := p.writeEssence(st, pkg); err != nil {
		return err
	}
	if err := p.writeDescriptive(st, pkg); err != nil {
		return err
	}
	if err := p.writeRepresentationMetadata(st, pkg); err != nil {
		return err
	}
	if err := p.writePackagePremis(st, pkg); err != nil {
		return err
	}
	return p.writePackageMets(st, pkg)
}

func (p *Profile) writeSkeleton(st *store.Store) error {
	for _, dir := range []string{
		"metadata/descriptive",
		"metadata/preservation",
		"representations",
		"schemas",
	} {
		if err := st.MkdirAll(dir); err != nil {
			return err
		}
	}
	return nil
}

func (p *Profile) writeSchemas(st *store.Store, pkg *sip.Package) error {
	xsds := schemas.Get()
	for _, sf := range pkg.SchemaFiles {
		info, err := st.WriteMetadata(sf.Path, func(w io.Writer) error {
			_, err := w.Write(xsds[sf.Name])
			return err
		})
		if err != nil {
			return err
		}
		backfill(sf, info)
	}
	return nil
}

func (p *Profile) writeEssence(st *store.Store, pkg *sip.Package) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		base := "representations/" + r.Label
		if err := st.MkdirAll(base + "/data"); err != nil {
			return err
		}
		if err := st.MkdirAll(base + "/metadata/preservation"); err != nil {
			return err
		}

		for _, f := range r.Files {
			// Fixity comes from the streamed copy: it describes the bytes
			// actually in the package, not the source they came from.
			info, err := st.CopyFile(f.Source, base+"/data/"+f.Name)
			if err != nil {
				return err
			}
			backfill(f, info)
		}
		return nil
	})
}

func (p *Profile) writeDescriptive(st *store.Store, pkg *sip.Package) error {
	df := pkg.Root.DescriptionFile
	info, err := st.WriteMetadata(df.Path, pkg.Root.Description.Encode)
	if err != nil {
		return err
	}
	backfill(df, info)
	return nil
}

func (p *Profile) writeRepresentationMetadata(st *store.Store, pkg *sip.Package) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		// PREMIS first: the representation METS embeds its fixity.
		pf := sip.NewFile()
		pf.Name = "premis.xml"
		pf.Path = "metadata/preservation/premis.xml" // rep-relative, per File.Path
		info, err := st.WriteMetadata("representations/"+r.Label+"/"+pf.Path, func(w io.Writer) error {
			return premis.EncodeRepresentation(w, r)
		})
		if err != nil {
			return err
		}
		backfill(pf, info)
		r.AddPremisFile(pf)
		p.Logger.Info("created a representation PREMIS file", slog.Any("id", pf.Identifier))

		mf := sip.NewFile()
		mf.Name = "METS.xml"
		mf.Path = "representations/" + r.Label + "/METS.xml" // package-relative: referenced from package METS
		info, err = st.WriteMetadata(mf.Path, func(w io.Writer) error {
			return mets.EncodeRepresentation(w, r, pkg.Spec)
		})
		if err != nil {
			return err
		}
		backfill(mf, info)
		r.AddMetsFile(mf)
		p.Logger.Info("created a representation METS file", slog.Any("id", mf.Identifier))
		return nil
	})
}

func (p *Profile) writePackagePremis(st *store.Store, pkg *sip.Package) error {
	// TODO also account for sub-IE(s) tied to the root entity
	pf := sip.NewFile()
	pf.Name = "premis.xml"
	pf.Path = "metadata/preservation/premis.xml"
	info, err := st.WriteMetadata(pf.Path, func(w io.Writer) error {
		return premis.EncodeEntity(w, pkg.Root)
	})
	if err != nil {
		return err
	}
	backfill(pf, info)
	pkg.AddPremisFile(pf)
	p.Logger.Info("created a package PREMIS file", slog.Any("id", pf.Identifier))
	return nil
}

func (p *Profile) writePackageMets(st *store.Store, pkg *sip.Package) error {
	mf := sip.NewFile()
	mf.Name = "METS.xml"
	mf.Path = "METS.xml"
	info, err := st.WriteMetadata(mf.Path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})
	if err != nil {
		return err
	}
	backfill(mf, info)
	pkg.AddMetsFile(mf)
	p.Logger.Info("created a package METS file", slog.Any("id", mf.Identifier))
	return nil
}

// backfill records the on-disk facts the store measured onto a graph node.
func backfill(f *sip.File, info store.Info) {
	f.Size = info.Size
	f.Checksum = info.Checksum
	f.Created = info.Created
}
