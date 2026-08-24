package profiles

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
	pkg.SetSchemaFiles(schemaFileNodes())

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
	// Read the local identifier before swapping in the entity identifier:
	// the terms hold one identifier slot, and the swap overwrites it. Per
	// the meemoo spec the emitted document carries the entity identifier;
	// the producer's own identifier travels as MEEMOO-LOCAL-ID.
	if def.EmitLocalIdentifier {
		e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", d.LocalIdentifier())
	}
	d.SetObjectIdentifier(e.Identifier)
	e.Description = d

	df := sip.NewFile()
	df.Name = def.DescriptiveName
	df.Path = "metadata/descriptive/" + df.Name
	df.Mime = "text/xml" // generated XML
	e.SetDescriptionFile(df)
	b.Logger.Info("created a descriptive file", slog.String("id", df.Identifier))
}

// schemaFileNodes declares one graph node per bundled XSD, sorted so METS
// emission is deterministic (schemas.Get() is a map; iterating it directly
// reorders the fileSec on every run).
func schemaFileNodes() []*sip.File {
	xsds := schemas.Get()
	files := make([]*sip.File, 0, len(xsds))
	for _, name := range slices.Sorted(maps.Keys(xsds)) {
		f := sip.NewFile()
		f.Name = name
		f.Path = "schemas/" + name
		f.Mime = "application/xml"
		files = append(files, f)
	}
	return files
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
// node. The package-side name (representation_N, in supplied order) is
// this project's stable convention, taken from the meemoo spec's
// illustrated layout. No spec dictates the name: CSIP requires only
// uniqueness, and meemoo 2.x only that the dir name equal the rep METS
// OBJID (which setting both from Name satisfies for free). The producer's
// label rides along as the human-readable name (rep METS mets/@LABEL).
func (b *Builder) assembleRepresentations(e *sip.Entity, def Definition, in *Input) error {
	for i, sr := range in.Representations {
		r := sip.NewRepresentation(fmt.Sprintf("representation_%d", i+1))
		r.Label = sr.Label
		b.Logger.Info("created a representation", slog.String("id", r.Identifier), slog.String("label", sr.Label))

		if sr.Descriptive != nil {
			// Mirror the package-level swap: when the terms carry an
			// identifier, the emitted document carries the representation
			// identifier instead (a no-op when they carry none; rep-level
			// identity is optional).
			sr.Descriptive.SetObjectIdentifier(r.Identifier)
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
			// asserts formats for SOURCE files, and the MD5 binding proves each
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

		received, err := b.assembleReceivedPremis(fmt.Sprintf("representation %q", sr.Label), sr.Premis)
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
// and checksum-bound to the bytes on disk; a stale format claim in
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

// verifyReportMD5 checks the report's checksum binding for src: the MD5 is
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
