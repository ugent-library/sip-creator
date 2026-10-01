package build

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path"
	"slices"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// assemble builds the complete package graph from the caller-supplied
// source package without writing anything to disk: every File node is created
// here with its Path declared, and the writer later back-fills fixity as
// it emits.
func (b *Builder) assemble(source *SourcePackage) (*sip.Package, error) {
	pkg := sip.NewPackage(b.destination, source.PackageIdentifier)
	b.logger.Info("created a new package", slog.String("id", pkg.Identifier))

	// The package's declaration is a copy of the profile's, with the record
	// status and content category the source package supplies. A copy, so
	// the graph never points into the builder's profile.
	decl := b.profile.Declaration
	if source.RecordStatus != "" {
		decl.RecordStatus = source.RecordStatus
	}
	if source.ContentCategory != "" {
		decl.Type = source.ContentCategory
	}
	pkg.Declaration = &decl

	e := sip.NewEntity()
	b.logger.Info("created an intellectual entity", slog.String("id", e.Identifier))

	b.assembleDescriptive(e, source)
	// The package ships the XSDs the METS documents point at and those the
	// descriptive encoder lists. Each encoder knows its own list. The
	// descriptive list ships for a supplied document too, whatever that
	// document's own schema-location hint names.
	schemaFiles, err := schemaFileNodes(slices.Concat(mets.Schemas, b.profile.Encoder.Schemas()))
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", b.profile.Name, err)
	}
	pkg.SchemaFiles = schemaFiles

	docs, err := b.assembleDocumentationNodes(source.Documentation, source.Characterization)
	if err != nil {
		return nil, err
	}
	pkg.DocumentationFiles = docs
	received, err := b.assembleReceivedPremis("package", source.Premis)
	if err != nil {
		return nil, err
	}
	pkg.ReceivedPremisFiles = received
	if err := b.assembleRepresentations(e, *pkg.Declaration, source); err != nil {
		return nil, err
	}

	if b.profile.EmitPackagePremis {
		pf := sip.NewFile()
		pf.Name = "premis.xml"
		pf.Path = "metadata/preservation/premis.xml"
		pf.Mime = "text/xml" // generated XML
		pkg.PremisFile = pf
		b.logger.Info("created a package PREMIS file", slog.String("id", pf.Identifier))
	}

	mf := sip.NewFile()
	mf.Name = "METS.xml"
	mf.Path = "METS.xml"
	// Set for the no-empty-Mime invariant even though no template reads it:
	// nothing references the package METS from inside the package.
	mf.Mime = "text/xml"
	pkg.MetsFile = mf
	b.logger.Info("created a package METS file", slog.String("id", mf.Identifier))

	pkg.Root = e
	return pkg, nil
}

func (b *Builder) assembleDescriptive(e *sip.Entity, source *SourcePackage) {
	d := source.Description
	// A standard that links descriptive and preservation metadata by a
	// shared identifier (meemoo's) swaps the entity identifier into the
	// description, and the producer's identifier it replaces travels as
	// MEEMOO-LOCAL-ID. Without a swap the document keeps the producer's
	// identifier as-is (ADR-0012).
	if s, ok := b.profile.Encoder.(IdentifierSwapper); ok {
		e.AdditionalIdentifiers["MEEMOO-LOCAL-ID"] = s.Swap(d, e.Identifier)
	}
	e.Description = d

	df := sip.NewFile()
	df.Name = b.profile.DescriptiveName
	df.Path = "metadata/descriptive/" + df.Name
	df.Mime = "text/xml" // rendered, or supplied and read as XML
	e.DescriptionFile = df
	b.logger.Info("created a descriptive file", slog.String("id", df.Identifier))
}

// schemaFileNodes declares one graph node per XSD the package ships, sorted
// so METS emission is deterministic whatever order the encoders list them
// in, and each name once: the METS list and the descriptive encoder's list
// overlap where two documents point at the same schema. A name the bundle
// does not hold is a mistake in an encoder's list and is refused here,
// before any write, rather than landing in the package as an empty file.
func schemaFileNodes(names []string) ([]*sip.File, error) {
	xsds := schemas.Get()
	files := make([]*sip.File, 0, len(names))
	for _, name := range slices.Compact(slices.Sorted(slices.Values(names))) {
		if _, ok := xsds[name]; !ok {
			return nil, fmt.Errorf("the schema %q is not bundled", name)
		}
		f := sip.NewFile()
		f.Name = name
		f.Path = "schemas/" + name
		f.Mime = "application/xml"
		files = append(files, f)
	}
	return files, nil
}

// assembleDocumentationNodes declares graph nodes for documentation files
// (package and representation level alike), each Path relative to its
// container and under documentation/. Unlike essence, documentation needs no characterization
// entry (ADR-0009), but a present entry's checksum must match: a mismatch
// proves the report stale.
func (b *Builder) assembleDocumentationNodes(sources []SourceFile, chars characterization.Report) ([]*sip.File, error) {
	var files []*sip.File
	for _, src := range sources {
		f := sip.NewFile()
		f.Name = path.Base(src.Path)
		f.Source = src.Source
		f.Path = "documentation/" + src.Path
		f.Mime = "application/octet-stream" // unknown; a report entry may refine it below

		if chars != nil {
			if rec, ok := chars[src.Key]; ok && rec.MD5 != "" {
				if err := verifyReportMD5(src.Source, rec); err != nil {
					return nil, err
				}
				if rec.Mime != "" {
					f.Mime = rec.Mime
				}
			}
		}

		files = append(files, f)
	}
	return files, nil
}

// assembleRepresentations turns each supplied representation into a graph
// node. The producer's name is used verbatim as the directory under
// representations/ and as the representation METS OBJID: no spec dictates
// a naming scheme (CSIP requires only that names be unique; meemoo 2.x
// requires the directory name to equal the OBJID, which holds because
// both come from Name). SourcePackage.Validate has already checked the
// names. decl is the package's declaration, which each representation's
// declaration starts from.
func (b *Builder) assembleRepresentations(e *sip.Entity, decl sip.MetsDeclaration, source *SourcePackage) error {
	for _, sr := range source.Representations {
		r := sip.NewRepresentation(sr.Name)
		r.Label = sr.label()
		r.Declaration = b.profile.representationDeclaration(decl, sr.resolvedType())
		b.logger.Info("created a representation", slog.String("id", r.Identifier), slog.String("name", sr.Name))

		if sr.Description != nil {
			// Mirror the package-level swap: the emitted document carries
			// the representation identifier instead of the producer's (a
			// no-op when the terms carry none; rep-level identity is
			// optional). The replaced value is not lifted: MEEMOO-LOCAL-ID
			// is an identifier of the entity.
			if s, ok := b.profile.Encoder.(IdentifierSwapper); ok {
				s.Swap(sr.Description, r.Identifier)
			}
			r.Description = sr.Description

			df := sip.NewFile()
			df.Name = b.profile.DescriptiveName
			df.Path = "metadata/descriptive/" + df.Name // rep-relative, per File.Path
			df.Mime = "text/xml"                        // rendered, or supplied and read as XML
			r.DescriptionFile = df
			b.logger.Info("created a representation descriptive file", slog.String("id", df.Identifier))
		}

		for _, src := range sr.Files {
			f := sip.NewFile()
			f.Name = path.Base(src.Path)
			f.Source = src.Source
			f.Path = "data/" + src.Path // rep-relative, per File.Path semantics
			// Characterization is an optional enricher (ADR-0009): the report
			// asserts formats for SOURCE files, and the MD5 check proves each
			// record still describes the bytes on disk. Fixity is not its job;
			// the writer computes that during the streamed copy.
			f.Mime = "application/octet-stream" // unknown; the report may refine it below
			if source.Characterization != nil {
				rec, err := b.essenceRecord(source.Characterization, src)
				if err != nil {
					return err
				}
				f.Format = rec.Format
				if rec.Mime != "" {
					f.Mime = rec.Mime
				}
			}
			f.Representation = r
			r.Files = append(r.Files, f)
			b.logger.Info("placed an essence file", slog.String("id", f.Identifier))
		}

		received, err := b.assembleReceivedPremis(fmt.Sprintf("representation %q", sr.Name), sr.Premis)
		if err != nil {
			return err
		}
		r.ReceivedPremisFiles = received

		docs, err := b.assembleDocumentationNodes(sr.Documentation, source.Characterization)
		if err != nil {
			return err
		}
		r.DocumentationFiles = docs

		if b.profile.EmitRepresentationPremis {
			pf := sip.NewFile()
			pf.Name = "premis.xml"
			pf.Path = "metadata/preservation/premis.xml" // rep-relative, per File.Path
			pf.Mime = "text/xml"                         // generated XML
			r.PremisFile = pf
			b.logger.Info("created a representation PREMIS file", slog.String("id", pf.Identifier))
		}

		mf := sip.NewFile()
		mf.Name = "METS.xml"
		mf.Path = "representations/" + r.Name + "/METS.xml" // package-relative: referenced from package METS
		mf.Mime = "text/xml"                                // generated XML
		r.MetsFile = mf
		b.logger.Info("created a representation METS file", slog.String("id", mf.Identifier))

		r.Entity = e
		e.Representations = append(e.Representations, r)
	}
	return nil
}

// assembleReceivedPremis declares graph nodes for received preservation
// documents: copied as received, never parsed or merged,
// but each must actually be a premis:premis document (parses as XML,
// PREMIS 3 namespace), because packaging a non-PREMIS file under
// metadata/preservation/ would be a false preservation claim. The check
// applies to every producer, so it lives here, not in the CLI walker alone.
func (b *Builder) assembleReceivedPremis(container string, sources []SourceFile) ([]*sip.File, error) {
	var files []*sip.File
	for _, src := range sources {
		f, err := os.Open(src.Source)
		if err != nil {
			return nil, fmt.Errorf("%s premis: %w", container, err)
		}
		err = premis.ValidateReceived(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s premis %s: %w", container, src.Path, err)
		}

		node := sip.NewFile()
		node.Name = path.Base(src.Path)
		node.Source = src.Source
		node.Path = "metadata/preservation/" + src.Path // container-relative, per File.Path
		node.Mime = "text/xml"                          // verified XML above
		files = append(files, node)
		b.logger.Info("placed a received preservation file", slog.String("id", node.Identifier))
	}
	return files, nil
}

// essenceRecord looks up the file's record and refuses it unless it is
// present, error-free, and its checksum matches the bytes on disk
// (ADR-0009): a stale format claim in preservation metadata is worse than
// none.
func (b *Builder) essenceRecord(chars characterization.Report, src SourceFile) (characterization.Record, error) {
	rec, ok := chars[src.Key]
	if !ok {
		return characterization.Record{}, fmt.Errorf(
			"characterization report has no entry for %q (report keys look like %s); generate the report from the input root: sf -hash md5 -json .",
			src.Key, sampleKey(chars))
	}
	if rec.Errors != "" {
		return characterization.Record{}, fmt.Errorf("characterization report records an error for %q: %s", src.Key, rec.Errors)
	}
	if rec.MD5 == "" {
		return characterization.Record{}, fmt.Errorf("characterization report carries no checksum for %q; generate it with sf -hash md5 -json", src.Key)
	}
	if err := verifyReportMD5(src.Source, rec); err != nil {
		return characterization.Record{}, err
	}
	return rec, nil
}

// sampleKey picks a deterministic example key for error messages, so a
// report generated from the wrong directory is self-explaining.
func sampleKey(chars characterization.Report) string {
	keys := slices.Sorted(maps.Keys(chars))
	if len(keys) == 0 {
		return "(the report is empty)"
	}
	return fmt.Sprintf("%q", keys[0])
}

// verifyReportMD5 checks that the report's checksum for src matches the
// file: the MD5 proves the record still describes these bytes.
func verifyReportMD5(src string, rec characterization.Record) error {
	sum, err := md5File(src)
	if err != nil {
		return err
	}
	if sum != rec.MD5 {
		return fmt.Errorf("%s changed since the characterization report was generated (file md5 %s, report has %s); regenerate the report", src, sum, rec.MD5)
	}
	return nil
}

// md5File streams the file's MD5; essence can be large.
func md5File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
