// Package input reads and validates a folder prepared per the input
// specification (docs/input-spec.md) into a neutral in-memory model.
//
// It is the CLI's frontend to the library: the library (profiles/, sip/)
// never imports it, and systems embedding the library construct the same
// data directly from their own stores instead of preparing a folder. The
// folder is one transport, not the API.
package input

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/profiles"
)

// File is one content, documentation, or received-PREMIS file found in the
// input folder.
type File struct {
	// Source is the absolute path of the file on disk.
	Source string
	// Rel is the path relative to the input root, slash-separated.
	// It is the characterization report key and is deliberately not
	// Unicode-normalized: a report lookup must match the filename
	// exactly as siegfried recorded it.
	Rel string
	// Path is the path the file will have inside the package, relative
	// to its container (the representation, documentation/ or premis/
	// root), slash-separated and NFC-normalized.
	Path string
}

// Representation is one version of the content.
type Representation struct {
	// Label is the representation folder's name (the input folder's name
	// in the flat case).
	Label string
	// Descriptive is nil unless the representation has its own
	// metadata.csv.
	Descriptive metadata.Terms
	// Files are the content files, in deterministic traversal order
	// (lexical per directory).
	Files []File
	// Documentation are the files from the representation's
	// documentation/ folder.
	Documentation []File
	// Premis is received preservation XML, passed through unparsed.
	Premis []File
}

// Package is the validated result of reading one input folder.
type Package struct {
	// Root is the absolute path of the input folder.
	Root string
	// Descriptive is the package-level terms from the top-level
	// metadata.csv.
	Descriptive metadata.Terms
	// Representations holds at least one representation; a flat folder
	// reads as a single one.
	Representations []Representation
	// Documentation are the files from the top-level documentation/
	// folder.
	Documentation []File
	// Premis is received preservation XML, passed through unparsed.
	Premis []File
	// Characterization is nil when the folder has no siegfried.json.
	Characterization characterization.Report
	// Warnings are SHOULD-level findings; the build proceeds.
	Warnings []string
}

// BuilderInput maps the validated folder onto the library's build input.
func (p *Package) BuilderInput() *profiles.Input {
	in := &profiles.Input{
		Descriptive:      p.Descriptive,
		Characterization: p.Characterization,
		Documentation:    sourceFiles(p.Documentation),
		Premis:           sourceFiles(p.Premis),
	}
	for _, rep := range p.Representations {
		in.Representations = append(in.Representations, profiles.SourceRepresentation{
			Label:         rep.Label,
			Files:         sourceFiles(rep.Files),
			Descriptive:   rep.Descriptive,
			Premis:        sourceFiles(rep.Premis),
			Documentation: sourceFiles(rep.Documentation),
		})
	}
	return in
}

func sourceFiles(files []File) []profiles.SourceFile {
	out := make([]profiles.SourceFile, len(files))
	for i, f := range files {
		out[i] = profiles.SourceFile{Source: f.Source, Key: f.Rel, Path: f.Path}
	}
	return out
}

// Read walks and validates the folder at root against the input
// specification. Every MUST violation is collected and returned together
// as a Violations error; when the error is non-nil the returned Package is
// incomplete and must not be built.
func Read(root string) (*Package, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("input folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("input folder %s is not a directory", root)
	}

	r := &reader{root: abs}
	pkg := r.read()
	pkg.Warnings = r.warnings
	if len(r.violations) > 0 {
		return pkg, r.violations
	}
	return pkg, nil
}

// reader carries the walk's state: the input root all messages and report
// keys are relative to, and the findings collected so far.
type reader struct {
	root       string
	violations Violations
	warnings   []string
}

// Reserved top-level names. Reserved names inside a representation are
// a subset.
const (
	metadataName        = "metadata.csv"
	representationsName = "representations"
	documentationName   = "documentation"
	premisName          = "premis"
	sidecarName         = "siegfried.json"
)

func (r *reader) read() *Package {
	pkg := &Package{Root: r.root}

	var content []os.DirEntry
	var repsDir string
	sawMetadata := false

	for _, e := range r.readDir(r.root) {
		// Reserved names are ASCII, which NFC normalization never alters,
		// so comparing unnormalized names is exact.
		name := e.Name()
		src := filepath.Join(r.root, e.Name())
		switch name {
		case metadataName:
			if e.IsDir() {
				r.violate("metadata.csv is a folder; the reserved name is for the metadata file")
				continue
			}
			sawMetadata = true
			pkg.Descriptive = r.decodeMetadataCSV(src, true)
		case representationsName:
			if !e.IsDir() {
				r.violate("representations is a file; the reserved name is for the folder of representations")
				continue
			}
			repsDir = src
		case documentationName:
			if !e.IsDir() {
				r.violate("documentation is a file; the reserved name is for a folder")
				continue
			}
			pkg.Documentation = r.collectFiles(src)
		case premisName:
			if !e.IsDir() {
				r.violate("premis is a file; the reserved name is for a folder")
				continue
			}
			pkg.Premis = r.collectPremisFiles(src)
		case sidecarName:
			if e.IsDir() {
				r.violate("siegfried.json is a folder; the reserved name is for the characterization report")
				continue
			}
			pkg.Characterization = r.decodeSidecar(src)
		default:
			content = append(content, e)
		}
	}

	if !sawMetadata {
		r.violate("metadata.csv is missing: every package folder needs one describing the content (input specification §3)")
	}

	if repsDir != "" {
		// With a representations/ folder, all content lives inside it;
		// only the reserved names may sit beside it.
		for _, e := range content {
			r.violate("%s: content must live inside representations/ when that folder exists (only documentation/ and premis/ may sit beside it)", e.Name())
		}
		pkg.Representations = r.readRepresentations(repsDir)
	} else {
		pkg.Representations = []Representation{r.readFlatRepresentation(content)}
	}

	return pkg
}

// decodeSidecar decodes the optional pre-computed characterization report.
// Decode strictness is ADR-0009's: a present report must parse; per-entry
// verification (the MD5 check) stays with the assembler, which knows which
// entries it needs.
func (r *reader) decodeSidecar(src string) characterization.Report {
	f, err := os.Open(src)
	if err != nil {
		r.violate("siegfried.json: %v", err)
		return nil
	}
	defer f.Close()

	report, err := characterization.DecodeSiegfried(f)
	if err != nil {
		r.violate("siegfried.json: %v; regenerate it from the input root with: sf -hash md5 -json .", err)
		return nil
	}
	return report
}
