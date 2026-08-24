package profiles

import (
	"io"
	"path"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// write emits pkg to disk in dependency order, back-filling fixity on File
// nodes as each file lands. Every node was created at assembly; the writer
// creates no graph nodes. The ordering is encoded only here and must not be
// changed casually: representation METS embeds the fixity of its PREMIS
// file, and package METS embeds the fixity of everything before it, so it
// goes strictly last.
func (b *Builder) write(st *store.Store, pkg *sip.Package, encodeDescriptive descriptiveEncoder) error {
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
	if err := b.writeRepresentationMetadata(st, pkg, encodeDescriptive); err != nil {
		return err
	}
	if pkg.PremisFile != nil {
		if err := b.writePackagePremis(st, pkg); err != nil {
			return err
		}
	}
	if err := b.writeReceivedPremis(st, pkg); err != nil {
		return err
	}
	return b.writePackageMets(st, pkg)
}

func (b *Builder) writeReceivedPremis(st *store.Store, pkg *sip.Package) error {
	return copyFiles(st, "", pkg.ReceivedPremisFiles)
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

func (b *Builder) writeDocumentation(st *store.Store, pkg *sip.Package) error {
	return copyFiles(st, "", pkg.DocumentationFiles)
}

func (b *Builder) writeEssence(st *store.Store, pkg *sip.Package) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		base := "representations/" + r.Name
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
		return encode(w, pkg.Root.Description, metadata.PackageSchemas)
	})
	if err != nil {
		return err
	}
	backfill(df, info)
	return nil
}

func (b *Builder) writeRepresentationMetadata(st *store.Store, pkg *sip.Package, encodeDescriptive descriptiveEncoder) error {
	return pkg.Root.EachRepresentation(func(r *sip.Representation) error {
		base := "representations/" + r.Name + "/"

		if r.Description != nil {
			df := r.DescriptionFile
			if err := st.MkdirAll(base + "metadata/descriptive"); err != nil {
				return err
			}
			info, err := st.WriteMetadata(base+df.Path, func(w io.Writer) error {
				return encodeDescriptive(w, r.Description, metadata.RepresentationSchemas)
			})
			if err != nil {
				return err
			}
			backfill(df, info)
		}

		if pf := r.PremisFile; pf != nil {
			info, err := st.WriteMetadata(base+pf.Path, func(w io.Writer) error {
				return premis.EncodeRepresentation(w, r)
			})
			if err != nil {
				return err
			}
			backfill(pf, info)
		}

		if err := copyFiles(st, base, r.DocumentationFiles); err != nil {
			return err
		}
		if err := copyFiles(st, base, r.ReceivedPremisFiles); err != nil {
			return err
		}

		mf := r.MetsFile
		info, err := st.WriteMetadata(mf.Path, func(w io.Writer) error {
			return mets.EncodeRepresentation(w, r, pkg.Spec)
		})
		if err != nil {
			return err
		}
		backfill(mf, info)
		return nil
	})
}

func (b *Builder) writePackagePremis(st *store.Store, pkg *sip.Package) error {
	// TODO also account for sub-IE(s) tied to the root entity
	pf := pkg.PremisFile
	info, err := st.WriteMetadata(pf.Path, func(w io.Writer) error {
		return premis.EncodeEntity(w, pkg.Root)
	})
	if err != nil {
		return err
	}
	backfill(pf, info)
	return nil
}

func (b *Builder) writePackageMets(st *store.Store, pkg *sip.Package) error {
	mf := pkg.MetsFile
	info, err := st.WriteMetadata(mf.Path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})
	if err != nil {
		return err
	}
	backfill(mf, info)
	return nil
}

// copyFiles copies pre-declared file nodes into the package under prefix
// (empty for package level, "representations/<name>/" for a representation),
// back-filling fixity from each streamed copy. Directories are created per
// file, so a container without such files gains no empty dir.
func copyFiles(st *store.Store, prefix string, files []*sip.File) error {
	for _, f := range files {
		if err := st.MkdirAll(path.Dir(prefix + f.Path)); err != nil {
			return err
		}
		info, err := st.CopyFile(f.Source, prefix+f.Path)
		if err != nil {
			return err
		}
		backfill(f, info)
	}
	return nil
}

// backfill records the on-disk facts the store measured onto a graph node.
func backfill(f *sip.File, info store.Info) {
	f.Size = info.Size
	f.Checksum = info.Checksum
	f.Created = info.Created
}
