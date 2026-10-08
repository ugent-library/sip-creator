// Package input reads a folder prepared according to the input
// specification (docs/input-spec.md), checks it, and returns the source
// package the library builds a package from.
//
// A program that uses the library can build the same build.SourcePackage
// from its own data, without a folder.
package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/build"
)

// Read checks the folder at root against the input specification and
// returns the source package it holds. mapper is the profile's: it maps
// the rows of a description.csv onto the profile's description.
// documentSpec describes the profile's descriptive document, which a
// folder may supply in place of the rows. Where the library has a rule for
// what Read reads, such as a description's Validate or the rule for a
// representation name, Read runs that same rule and reports its findings
// with the file, and the line where there is one. Read returns every
// broken MUST rule together as a Violations error. When the error is not
// nil, the source package is incomplete and must not be built. Read
// returns another error, and no source package, if mapper is nil or root
// is not a folder.
func Read(root string, mapper Mapper, documentSpec DocumentSpec) (*build.SourcePackage, error) {
	if mapper == nil {
		return nil, errors.New("no mapper: pass the profile's mapper, which maps the rows of description.csv onto its description")
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
		return nil, fmt.Errorf("input folder %s is a file, not a folder", root)
	}

	r := &folderReader{root: abs}
	format, takesDocument := documentSpec.Model.(build.DocumentFormat)
	if takesDocument {
		r.documentName = documentSpec.Name
	} else {
		r.refusedDocumentName = documentSpec.Name
	}

	source, inv := r.walk()
	r.decode(source, inv, mapper, format)
	if len(r.violations) > 0 {
		return source, r.violations
	}
	return source, nil
}
