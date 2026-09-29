// Package input reads and validates a folder prepared per the input
// specification (docs/input-spec.md) into a neutral in-memory model.
//
// It is the CLI's frontend to the library: the library (profiles/, sip/)
// never imports it, and systems embedding the library construct the same
// data directly from their own stores instead of preparing a folder. The
// folder is one transport, not the API.
package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/sip"
)

// DescriptionBuilder builds the profile's description from the flat
// statements a description.csv decodes to. It is the one thing this package
// needs from a profile, declared here so the reader takes no profile
// definition: a profile's descriptive encoder satisfies it, and a test may
// pass its own. The description it returns carries the vocabulary's rules
// (Validate, ValidateRequired), which the reader runs and reports as
// violations.
type DescriptionBuilder interface {
	// NewDescription builds the description the terms state. The terms are
	// decoded for syntax only: which keys exist and what a term may say
	// are the description's own rules.
	NewDescription(terms []sip.Term) sip.Description
}

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
	// Name is the representation folder's name (the input folder's name
	// in the flat case): the package-side directory name.
	Name string
	// Label is the display name from representations.csv; empty when the
	// file is absent or leaves the cell empty (the library defaults it to
	// the name).
	Label string
	// Type is the representation type from representations.csv; empty when
	// the file is absent or leaves the cell empty (the library defaults it
	// to the label).
	Type string
	// Description is nil unless the representation has its own
	// description.csv; then it is what the reader's builder made of the
	// rows.
	Description sip.Description
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
	// Description is the package-level description from the top-level
	// description.csv, as the reader's builder made it.
	Description sip.Description
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
func (p *Package) BuilderInput() *build.Input {
	in := &build.Input{
		Characterization: p.Characterization,
		Documentation:    sourceFiles(p.Documentation),
		Premis:           sourceFiles(p.Premis),
	}
	// Assign a description only when the folder had one: a nil eark.Terms
	// stored in the interface field would read as a present, empty
	// description.
	if p.Description != nil {
		in.Description = p.Description
	}
	for _, rep := range p.Representations {
		sr := build.SourceRepresentation{
			Name:          rep.Name,
			Label:         rep.Label,
			Type:          rep.Type,
			Files:         sourceFiles(rep.Files),
			Premis:        sourceFiles(rep.Premis),
			Documentation: sourceFiles(rep.Documentation),
		}
		if rep.Description != nil {
			sr.Description = rep.Description
		}
		in.Representations = append(in.Representations, sr)
	}
	return in
}

func sourceFiles(files []File) []build.SourceFile {
	out := make([]build.SourceFile, len(files))
	for i, f := range files {
		out[i] = build.SourceFile{Source: f.Source, Key: f.Rel, Path: f.Path}
	}
	return out
}

// Reader reads input folders as one profile: the builder it holds says
// which vocabulary the rows of a description.csv are in, and each Read
// walks one folder with it. check and create both construct it with the
// profile's descriptive encoder, so they read a folder the same way.
type Reader struct {
	builder DescriptionBuilder
}

// New returns a reader whose folders are read with builder. Like
// build.New, it validates nothing: Read refuses a nil builder.
func New(builder DescriptionBuilder) *Reader {
	return &Reader{builder: builder}
}

// Read walks and validates the folder at root against the input
// specification. Every MUST violation is collected and returned together
// as a Violations error; when the error is non-nil the returned Package is
// incomplete and must not be built.
func (r *Reader) Read(root string) (*Package, error) {
	if r.builder == nil {
		return nil, errors.New("no description builder: construct the reader with the profile's descriptive encoder, which says what the rows of description.csv mean")
	}
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

	d := &directory{root: abs, builder: r.builder}
	pkg := d.read()
	pkg.Warnings = d.warnings
	if len(d.violations) > 0 {
		return pkg, d.violations
	}
	return pkg, nil
}
