package profiles

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/ugent-library/sip-creator/characterization"
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

	chars, err := b.assembleCharacterization(def)
	if err != nil {
		return nil, err
	}

	e := sip.NewEntity()
	b.Logger.Info("created an intellectual entity", slog.Any("id", e.Identifier))

	if err := b.assembleDescriptive(e, def); err != nil {
		return nil, err
	}

	pkg.AddSchemaFiles(schemaFileNodes())

	if err := b.assembleDocumentation(pkg, chars); err != nil {
		return nil, err
	}

	if err := b.assembleRepresentations(e, chars); err != nil {
		return nil, err
	}

	pkg.AddRootEntity(e)
	return pkg, nil
}

// assembleCharacterization resolves the build's characterization report: a
// caller-supplied report wins; otherwise the profile's sidecar file is read
// from the input root. No report at all is fine — the build proceeds
// without format info — but a present one must decode (ADR-0009: optional
// in contract, fully strict when present).
func (b *Builder) assembleCharacterization(def Definition) (characterization.Report, error) {
	if b.Characterization != nil {
		return b.Characterization, nil
	}
	if def.CharacterizationSource == "" {
		return nil, nil
	}

	src := filepath.Join(b.InDir, def.CharacterizationSource)
	f, err := os.Open(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("characterization source: %w", err)
	}
	defer f.Close()

	report, err := characterization.DecodeSiegfried(f)
	if err != nil {
		return nil, fmt.Errorf("characterization source %s: %w", src, err)
	}
	b.Logger.Info("decoded a characterization report", slog.Int("entries", len(report)))
	return report, nil
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

// assembleDocumentation declares graph nodes for the optional package-level
// documentation/ input directory; an absent directory means none. Nested
// structure is preserved in the package-relative Path.
func (b *Builder) assembleDocumentation(pkg *sip.Package, chars characterization.Report) error {
	dir := filepath.Join(b.InDir, "documentation")
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("documentation source: %w", err)
	}

	var files []*sip.File
	err := filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, src)
		if err != nil {
			return err
		}

		// Documentation is lenient where essence is strict (ADR-0009):
		// these files carry no premis:format and may postdate the report,
		// so no entry is required — but a present entry must still be
		// checksum-true, because a mismatch proves the report stale.
		if chars != nil {
			key, err := reportKey(b.InDir, src)
			if err != nil {
				return err
			}
			if rec, ok := chars[key]; ok && rec.MD5 != "" {
				if err := verifyReportMD5(src, rec); err != nil {
					return err
				}
			}
		}

		f := sip.NewFile()
		f.Name = filepath.Base(src)
		f.Source = src
		f.Path = "documentation/" + filepath.ToSlash(rel)
		files = append(files, f)
		return nil
	})
	if err != nil {
		return err
	}

	pkg.AddDocumentationFiles(files)
	return nil
}

// TODO fix case "representation_0"
var repDirRx = regexp.MustCompile("representation_([0-9]+)$")

func (b *Builder) assembleRepresentations(e *sip.Entity, chars characterization.Report) error {
	return filepath.Walk(b.InDir, func(dir string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || !repDirRx.MatchString(filepath.Base(dir)) {
			return nil
		}

		r := sip.NewRepresentation(filepath.Base(dir))
		b.Logger.Info("created a representation", slog.Any("id", r.Identifier))

		if err := b.assembleEssenceFiles(dir, r, chars); err != nil {
			return err
		}

		r.SetEntity(e)
		e.AddRepresentation(r)
		return nil
	})
}

func (b *Builder) assembleEssenceFiles(dir string, r *sip.Representation, chars characterization.Report) error {
	return filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// TODO ignore all descriptive files per the spec (dc, dc+schema, mods)

		f := sip.NewFile()
		f.Name = filepath.Base(src)
		f.Source = src
		f.Path = "data/" + f.Name // rep-relative, per File.Path semantics
		// Characterization is an optional enricher (ADR-0009): the report
		// asserts formats for SOURCE files, and the MD5 binding proves each
		// record still describes the bytes on disk. Fixity is not its job —
		// the writer computes that during the streamed copy.
		if chars != nil {
			rec, err := b.essenceRecord(chars, src)
			if err != nil {
				return err
			}
			f.Format = rec.Format
		}
		f.SetRepresentation(r)
		r.AddFile(f)
		b.Logger.Info("placed an essence file", slog.Any("id", f.Identifier))
		return nil
	})
}

// essenceRecord looks up src's record and enforces ADR-0009's strictness:
// every essence file must be present in the report, error-free, and
// checksum-bound to the bytes on disk — a stale format claim in
// preservation metadata is worse than none.
func (b *Builder) essenceRecord(chars characterization.Report, src string) (characterization.Record, error) {
	key, err := reportKey(b.InDir, src)
	if err != nil {
		return characterization.Record{}, err
	}
	rec, ok := chars[key]
	if !ok {
		return characterization.Record{}, fmt.Errorf(
			"characterization report has no entry for %q (report keys look like %s) — generate the report from the input root: cd %s && sf -hash md5 -json .",
			key, sampleKey(chars), b.InDir)
	}
	if rec.Errors != "" {
		return characterization.Record{}, fmt.Errorf("characterization report records an error for %q: %s", key, rec.Errors)
	}
	if rec.MD5 == "" {
		return characterization.Record{}, fmt.Errorf("characterization report carries no checksum for %q — generate it with sf -hash md5 -json", key)
	}
	if err := verifyReportMD5(src, rec); err != nil {
		return characterization.Record{}, err
	}
	return rec, nil
}

// reportKey is the input-relative slash path a report keys its records by —
// the same normalization DecodeSiegfried applies to sf's filenames.
func reportKey(inDir, src string) (string, error) {
	rel, err := filepath.Rel(inDir, src)
	if err != nil {
		return "", err
	}
	return path.Clean(filepath.ToSlash(rel)), nil
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
