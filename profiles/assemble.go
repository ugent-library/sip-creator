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
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// assemble builds the complete package graph from the caller-supplied
// input without writing anything to disk: every File node is born here
// with its Path declared, and the writer later back-fills fixity as it
// emits.
func (b *Builder) assemble(def Definition, in *Input) (*sip.Package, error) {
	pkg := sip.NewPackage(b.OutDir)
	b.Logger.Info("created a new package", slog.Any("id", pkg.Identifier))

	pkg.Spec = &def.Mets

	e := sip.NewEntity()
	b.Logger.Info("created an intellectual entity", slog.Any("id", e.Identifier))

	b.assembleDescriptive(e, def, in)
	pkg.AddSchemaFiles(schemaFileNodes())

	if err := b.assembleDocumentation(pkg, in); err != nil {
		return nil, err
	}
	if err := b.assembleRepresentations(e, in); err != nil {
		return nil, err
	}

	pkg.AddRootEntity(e)
	return pkg, nil
}

func (b *Builder) assembleDescriptive(e *sip.Entity, def Definition, in *Input) {
	d := in.Descriptive
	// Read the local identifier before swapping in the entity identifier:
	// the terms hold one identifier slot, and the swap overwrites it. Per
	// the meemoo spec the emitted document carries the entity identifier;
	// the producer's own identifier travels as MEEMOO-LOCAL-ID.
	if def.LocalIdentifierScheme != "" {
		e.AddAdditionalIdentifier("MEEMOO-LOCAL-ID", d.GetLocalIdentifier(def.LocalIdentifierScheme))
	}
	d.SetObjectIdentifier(e.Identifier)
	e.Description = d

	df := sip.NewFile()
	df.Name = def.DescriptiveName
	df.Path = "metadata/descriptive/" + df.Name // declared, not derived from disk
	df.Mime = "text/xml"                        // the writer renders it as XML by construction
	e.AddDescriptionFile(df)
	b.Logger.Info("created a descriptive file", slog.Any("id", df.Identifier))
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
		f.Mime = "application/xml" // XSDs by construction
		files = append(files, f)
	}
	return files
}

// assembleDocumentation declares graph nodes for the optional package-level
// documentation files.
func (b *Builder) assembleDocumentation(pkg *sip.Package, in *Input) error {
	var files []*sip.File
	for _, src := range in.Documentation {
		f := sip.NewFile()
		f.Name = path.Base(src.Path)
		f.Source = src.Source
		f.Path = "documentation/" + src.Path
		f.Mime = "application/octet-stream" // the admitted unknown, never a guess

		// Documentation is lenient where essence is strict (ADR-0009):
		// these files carry no premis:format and may postdate the report,
		// so no entry is required — but a present entry must still be
		// checksum-true, because a mismatch proves the report stale.
		if in.Characterization != nil {
			if rec, ok := in.Characterization[src.Key]; ok && rec.MD5 != "" {
				if err := verifyReportMD5(src.Source, rec); err != nil {
					return err
				}
				if rec.Mime != "" {
					f.Mime = rec.Mime
				}
			}
		}

		files = append(files, f)
	}
	pkg.AddDocumentationFiles(files)
	return nil
}

// assembleRepresentations turns each supplied representation into a graph
// node. The package-side directory name is the profile family's convention
// (meemoo's representation_N, in supplied order); the producer's label is
// input-side naming and does not reach the package yet.
func (b *Builder) assembleRepresentations(e *sip.Entity, in *Input) error {
	for i, sr := range in.Representations {
		r := sip.NewRepresentation(fmt.Sprintf("representation_%d", i+1))
		b.Logger.Info("created a representation", slog.Any("id", r.Identifier), slog.String("label", sr.Label))

		for _, src := range sr.Files {
			f := sip.NewFile()
			f.Name = path.Base(src.Path)
			f.Source = src.Source
			f.Path = "data/" + src.Path // rep-relative, per File.Path semantics
			// Characterization is an optional enricher (ADR-0009): the report
			// asserts formats for SOURCE files, and the MD5 binding proves each
			// record still describes the bytes on disk. Fixity is not its job —
			// the writer computes that during the streamed copy.
			f.Mime = "application/octet-stream" // the admitted unknown, never a guess
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
			b.Logger.Info("placed an essence file", slog.Any("id", f.Identifier))
		}

		r.SetEntity(e)
		e.AddRepresentation(r)
	}
	return nil
}

// essenceRecord looks up the file's record and enforces ADR-0009's
// strictness: every essence file must be present in the report, error-free,
// and checksum-bound to the bytes on disk — a stale format claim in
// preservation metadata is worse than none.
func (b *Builder) essenceRecord(chars characterization.Report, src SourceFile) (characterization.Record, error) {
	rec, ok := chars[src.Key]
	if !ok {
		return characterization.Record{}, fmt.Errorf(
			"characterization report has no entry for %q (report keys look like %s) — generate the report from the input root: sf -hash md5 -json .",
			src.Key, sampleKey(chars))
	}
	if rec.Errors != "" {
		return characterization.Record{}, fmt.Errorf("characterization report records an error for %q: %s", src.Key, rec.Errors)
	}
	if rec.MD5 == "" {
		return characterization.Record{}, fmt.Errorf("characterization report carries no checksum for %q — generate it with sf -hash md5 -json", src.Key)
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
		return fmt.Errorf("%s changed since the characterization report was generated (file md5 %s, report has %s) — regenerate the report", src, sum, rec.MD5)
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
