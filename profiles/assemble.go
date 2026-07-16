package profiles

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// assemble builds the complete package graph from the input tree without
// writing anything to disk: every File node is born here with its Path
// declared, and the writer later back-fills fixity as it emits.
func (b *Builder) assemble(def Definition) (*sip.Package, error) {
	pkg := sip.NewPackage(b.OutDir)
	b.Logger.Info("created a new package", slog.Any("id", pkg.Identifier))

	pkg.Spec = &def.Mets

	e := sip.NewEntity()
	b.Logger.Info("created an intellectual entity", slog.Any("id", e.Identifier))

	if err := b.assembleDescriptive(e, def); err != nil {
		return nil, err
	}

	pkg.AddSchemaFiles(schemaFileNodes())

	if err := b.assembleRepresentations(e); err != nil {
		return nil, err
	}

	pkg.AddRootEntity(e)
	return pkg, nil
}

func (b *Builder) assembleDescriptive(e *sip.Entity, def Definition) error {
	src := filepath.Join(b.InDir, def.DescriptiveSource)
	d, err := decodeDescriptive(src)
	if err != nil {
		return err
	}

	// Per the spec, we want to swap in the premis identifier for
	// dcterms:identifier, and keep the source's own identifier with the
	// entity for the premis file.
	d.SetObjectIdentifier(e.Identifier)
	// TODO This could be auto-detected based off the salience of the source metadata
	//   e.g. metadata properties which describe a group of additional identifiers.
	if def.LocalIdentifierScheme != "" {
		e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", d.GetLocalIdentifier(def.LocalIdentifierScheme))
	}
	e.Description = d

	df := sip.NewFile()
	df.Name = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + ".xml"
	df.Path = "metadata/descriptive/" + df.Name // declared, not derived from disk
	e.AddDescriptionFile(df)
	b.Logger.Info("created a descriptive file", slog.Any("id", df.Identifier))
	return nil
}

// decodeDescriptive is the single seam between an input path and a
// sip.Descriptive; a future decoder registry (CSV, ...) replaces only this.
func decodeDescriptive(src string) (*metadata.Description, error) {
	fi, err := os.Lstat(src)
	if err != nil {
		return nil, fmt.Errorf("descriptive source: %w", err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("descriptive source %s is a directory, not a metadata file", src)
	}

	f, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("descriptive source: %w", err)
	}
	defer f.Close()

	return metadata.Decode(f)
}

// schemaFileNodes declares one graph node per bundled XSD, sorted so METS
// emission is deterministic (schemas.Get() is a map; iterating it directly
// reordered the fileSec on every run — the Phase-0 baseline finding).
func schemaFileNodes() []*sip.File {
	xsds := schemas.Get()
	files := make([]*sip.File, 0, len(xsds))
	for _, name := range slices.Sorted(maps.Keys(xsds)) {
		f := sip.NewFile()
		f.Name = name
		f.Path = "schemas/" + name
		files = append(files, f)
	}
	return files
}

// TODO fix case "representation_0"
var repDirRx = regexp.MustCompile("representation_([0-9]+)$")

func (b *Builder) assembleRepresentations(e *sip.Entity) error {
	return filepath.Walk(b.InDir, func(dir string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || !repDirRx.MatchString(filepath.Base(dir)) {
			return nil
		}

		r := sip.NewRepresentation(filepath.Base(dir))
		b.Logger.Info("created a representation", slog.Any("id", r.Identifier))

		if err := b.assembleEssenceFiles(dir, r); err != nil {
			return err
		}

		r.SetEntity(e)
		e.AddRepresentation(r)
		return nil
	})
}

func (b *Builder) assembleEssenceFiles(dir string, r *sip.Representation) error {
	return filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// TODO ignore all descriptive files per the spec (dc, dc+schema, mods)

		f := b.Formats.Process(src) // identify the SOURCE, before anything is on disk
		f.Source = src
		f.Path = "data/" + filepath.Base(src) // rep-relative, per File.Path semantics
		f.SetRepresentation(r)
		r.AddFile(f)
		b.Logger.Info("placed an essence file", slog.Any("id", f.Identifier))
		return nil
	})
}
