package profiles

import (
	"io"
	"log/slog"
	"path"

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
func (b *Builder) write(st *store.Store, pkg *sip.Package, def Definition, encodeDescriptive descriptiveEncoder) error {
	if err := b.writeSkeleton(st); err != nil {
		return err
	}
	if err := b.writeSchemas(st, pkg); err != nil {
		return err
	}
	if err := b.writeDocumentation(st, pkg); err != nil {
		return err
	}
	if err := b.writeEssence(st, pkg); err != nil {
		return err
	}
	if err := b.writeDescriptive(st, pkg, encodeDescriptive); err != nil {
		return err
	}
	if err := b.writeRepresentationMetadata(st, pkg, def); err != nil {
		return err
	}
	if def.EmitPackagePremis {
		if err := b.writePackagePremis(st, pkg); err != nil {
			return err
		}
	}
	return b.writePackageMets(st, pkg)
}

func (b *Builder) writeSkeleton(st *store.Store) error {
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

func (b *Builder) writeSchemas(st *store.Store, pkg *sip.Package) error {
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

// writeDocumentation copies the optional documentation files; directories
// are created per file, so a package without documentation gains no empty
// documentation/ dir.
func (b *Builder) writeDocumentation(st *store.Store, pkg *sip.Package) error {
	for _, f := range pkg.DocumentationFiles {
		if err := st.MkdirAll(path.Dir(f.Path)); err != nil {
			return err
		}
		info, err := st.CopyFile(f.Source, f.Path)
		if err != nil {
			return err
		}
		backfill(f, info)
	}
	return nil
}

func (b *Builder) writeEssence(st *store.Store, pkg *sip.Package) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		base := "representations/" + r.Label
		if err := st.MkdirAll(base + "/data"); err != nil {
			return err
		}
		if err := st.MkdirAll(base + "/metadata/preservation"); err != nil {
			return err
		}

		for _, f := range r.Files {
			// f.Path preserves the producer's nesting under data/; create
			// the intermediate dirs the flat MkdirAll above doesn't cover.
			if err := st.MkdirAll(base + "/" + path.Dir(f.Path)); err != nil {
				return err
			}
			// Fixity comes from the streamed copy: it describes the bytes
			// actually in the package, not the source they came from.
			info, err := st.CopyFile(f.Source, base+"/"+f.Path)
			if err != nil {
				return err
			}
			backfill(f, info)
		}
		return nil
	})
}

func (b *Builder) writeDescriptive(st *store.Store, pkg *sip.Package, encode descriptiveEncoder) error {
	df := pkg.Root.DescriptionFile
	info, err := st.WriteMetadata(df.Path, func(w io.Writer) error {
		return encode(w, pkg.Root.Description)
	})
	if err != nil {
		return err
	}
	backfill(df, info)
	return nil
}

func (b *Builder) writeRepresentationMetadata(st *store.Store, pkg *sip.Package, def Definition) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		if def.EmitRepresentationPremis {
			// PREMIS first: the representation METS embeds its fixity.
			pf := sip.NewFile()
			pf.Name = "premis.xml"
			pf.Path = "metadata/preservation/premis.xml" // rep-relative, per File.Path
			pf.Mime = "text/xml"                         // generated XML by construction
			info, err := st.WriteMetadata("representations/"+r.Label+"/"+pf.Path, func(w io.Writer) error {
				return premis.EncodeRepresentation(w, r)
			})
			if err != nil {
				return err
			}
			backfill(pf, info)
			r.AddPremisFile(pf)
			b.Logger.Info("created a representation PREMIS file", slog.Any("id", pf.Identifier))
		}

		mf := sip.NewFile()
		mf.Name = "METS.xml"
		mf.Path = "representations/" + r.Label + "/METS.xml" // package-relative: referenced from package METS
		mf.Mime = "text/xml"                                 // generated XML by construction
		info, err := st.WriteMetadata(mf.Path, func(w io.Writer) error {
			return mets.EncodeRepresentation(w, r, pkg.Spec)
		})
		if err != nil {
			return err
		}
		backfill(mf, info)
		r.AddMetsFile(mf)
		b.Logger.Info("created a representation METS file", slog.Any("id", mf.Identifier))
		return nil
	})
}

func (b *Builder) writePackagePremis(st *store.Store, pkg *sip.Package) error {
	// TODO also account for sub-IE(s) tied to the root entity
	pf := sip.NewFile()
	pf.Name = "premis.xml"
	pf.Path = "metadata/preservation/premis.xml"
	pf.Mime = "text/xml" // generated XML by construction
	info, err := st.WriteMetadata(pf.Path, func(w io.Writer) error {
		return premis.EncodeEntity(w, pkg.Root)
	})
	if err != nil {
		return err
	}
	backfill(pf, info)
	pkg.AddPremisFile(pf)
	b.Logger.Info("created a package PREMIS file", slog.Any("id", pf.Identifier))
	return nil
}

func (b *Builder) writePackageMets(st *store.Store, pkg *sip.Package) error {
	mf := sip.NewFile()
	mf.Name = "METS.xml"
	mf.Path = "METS.xml"
	// Set for the no-empty-Mime invariant even though no template reads it:
	// nothing references the package METS from inside the package.
	mf.Mime = "text/xml"
	info, err := st.WriteMetadata(mf.Path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})
	if err != nil {
		return err
	}
	backfill(mf, info)
	pkg.AddMetsFile(mf)
	b.Logger.Info("created a package METS file", slog.Any("id", mf.Identifier))
	return nil
}

// backfill records the on-disk facts the store measured onto a graph node.
func backfill(f *sip.File, info store.Info) {
	f.Size = info.Size
	f.Checksum = info.Checksum
	f.Created = info.Created
}
