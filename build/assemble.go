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
// input without writing anything to disk: every File node is created here
// with its Path declared, and the writer later back-fills fixity as it
// emits.
func (b *Builder) assemble(def Definition, in *Input) (*sip.Package, error) {
	pkg := sip.NewPackage(b.Destination, in.PackageIdentifier)
	b.Logger.Info("created a new package", slog.String("id", pkg.Identifier))

	pkg.Declaration = &def.Declaration

	e := sip.NewEntity()
	b.Logger.Info("created an intellectual entity", slog.String("id", e.Identifier))

	b.assembleDescriptive(e, def, in)
	// The package ships the XSDs its documents point at and nothing else:
	// what the METS documents reference, and what the descriptive document
	// references. Each encoder knows its own list.
	schemaFiles, err := schemaFileNodes(slices.Concat(mets.Schemas, def.Encoder.Schemas()))
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", def.Name, err)
	}
	pkg.SetSchemaFiles(schemaFiles)

	docs, err := b.assembleDocumentationNodes(in.Documentation, in.Characterization)
	if err != nil {
		return nil, err
	}
	pkg.SetDocumentationFiles(docs)
	received, err := b.assembleReceivedPremis("package", in.Premis)
	if err != nil {
		return nil, err
	}
	pkg.SetReceivedPremisFiles(received)
	if err := b.assembleRepresentations(e, def, in); err != nil {
		return nil, err
	}

	if def.EmitPackagePremis {
		pf := sip.NewFile()
		pf.Name = "premis.xml"
		pf.Path = "metadata/preservation/premis.xml"
		pf.Mime = "text/xml" // generated XML
		pkg.SetPremisFile(pf)
		b.Logger.Info("created a package PREMIS file", slog.String("id", pf.Identifier))
	}

	mf := sip.NewFile()
	mf.Name = "METS.xml"
	mf.Path = "METS.xml"
	// Set for the no-empty-Mime invariant even though no template reads it:
	// nothing references the package METS from inside the package.
	mf.Mime = "text/xml"
	pkg.SetMetsFile(mf)
	b.Logger.Info("created a package METS file", slog.String("id", mf.Identifier))

	pkg.SetRoot(e)
	return pkg, nil
}

func (b *Builder) assembleDescriptive(e *sip.Entity, def Definition, in *Input) {
	d := in.Descriptive
	// A standard that links descriptive and preservation metadata by a
	// shared identifier (meemoo's) swaps the entity identifier into the
	// description, and the producer's identifier it replaces travels as
	// MEEMOO-LOCAL-ID. Without a swap the document keeps the producer's
	// identifier as-is (ADR-0012).
	if s, ok := def.Encoder.(IdentifierSwapper); ok {
		e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", s.Swap(d, e.Identifier))
	}
	e.Description = d

	df := sip.NewFile()
	df.Name = def.DescriptiveName
	df.Path = "metadata/descriptive/" + df.Name
	df.Mime = "text/xml" // generated XML
	e.SetDescriptionFile(df)
	b.Logger.Info("created a descriptive file", slog.String("id", df.Identifier))
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
// (package and representation level alike), Path relative to the container's
// documentation/ dir. Unlike essence, documentation needs no characterization
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
// node. The package-side name (the directory under representations/ and the
// rep METS OBJID) is the producer's name, used verbatim: no spec dictates a
// naming scheme (CSIP requires only uniqueness, and meemoo 2.x only that the
// dir name equal the rep METS OBJID, which setting both from Name satisfies
// for free), and Input.Validate has already checked every name for
// uniqueness and the portable character set. Label and type resolve along
// the defaulting cascade (name → label → type).
func (b *Builder) assembleRepresentations(e *sip.Entity, def Definition, in *Input) error {
	for _, sr := range in.Representations {
		r := sip.NewRepresentation(sr.Name)
		r.Label = sr.label()
		r.Declaration = def.representationDeclaration(sr.resolvedType())
		b.Logger.Info("created a representation", slog.String("id", r.Identifier), slog.String("name", sr.Name))

		if sr.Descriptive != nil {
			// Mirror the package-level swap: the emitted document carries
			// the representation identifier instead of the producer's (a
			// no-op when the terms carry none; rep-level identity is
			// optional). The replaced value is not lifted: MEEMOO-LOCAL-ID
			// is an identifier of the entity.
			if s, ok := def.Encoder.(IdentifierSwapper); ok {
				s.Swap(sr.Descriptive, r.Identifier)
			}
			r.Description = sr.Descriptive

			df := sip.NewFile()
			df.Name = def.DescriptiveName
			df.Path = "metadata/descriptive/" + df.Name // rep-relative, per File.Path
			df.Mime = "text/xml"                        // generated XML
			r.SetDescriptionFile(df)
			b.Logger.Info("created a representation descriptive file", slog.String("id", df.Identifier))
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
			if in.Characterization != nil {
				rec, err := b.essenceRecord(in.Characterization, src)
				if err != nil {
					return err
				}
				f.Format = rec.Format
				if rec.Mime != "" {
					f.Mime = rec.Mime
				}
			}
			f.SetRepresentation(r)
			r.AddFile(f)
			b.Logger.Info("placed an essence file", slog.String("id", f.Identifier))
		}

		received, err := b.assembleReceivedPremis(fmt.Sprintf("representation %q", sr.Name), sr.Premis)
		if err != nil {
			return err
		}
		r.SetReceivedPremisFiles(received)

		docs, err := b.assembleDocumentationNodes(sr.Documentation, in.Characterization)
		if err != nil {
			return err
		}
		r.SetDocumentationFiles(docs)

		if def.EmitRepresentationPremis {
			pf := sip.NewFile()
			pf.Name = "premis.xml"
			pf.Path = "metadata/preservation/premis.xml" // rep-relative, per File.Path
			pf.Mime = "text/xml"                         // generated XML
			r.SetPremisFile(pf)
			b.Logger.Info("created a representation PREMIS file", slog.String("id", pf.Identifier))
		}

		mf := sip.NewFile()
		mf.Name = "METS.xml"
		mf.Path = "representations/" + r.Name + "/METS.xml" // package-relative: referenced from package METS
		mf.Mime = "text/xml"                                // generated XML
		r.SetMetsFile(mf)
		b.Logger.Info("created a representation METS file", slog.String("id", mf.Identifier))

		r.SetEntity(e)
		e.AddRepresentation(r)
	}
	return nil
}

// assembleReceivedPremis declares graph nodes for received preservation
// documents: copied as received, never parsed or merged,
// but each must actually be a premis:premis document (well-formed, PREMIS 3
// namespace), because packaging a non-PREMIS file under
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
		b.Logger.Info("placed a received preservation file", slog.String("id", node.Identifier))
	}
	return files, nil
}

// essenceRecord looks up the file's record and enforces ADR-0009's
// strictness: every essence file must be present in the report, error-free,
// and its checksum must match the bytes on disk; a stale format claim in
// preservation metadata is worse than none.
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

// verifyReportMD5 checks the report's checksum for src: the MD5 is
// what ties a record to the bytes it describes (the staleness defense).
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
