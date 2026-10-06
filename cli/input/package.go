// Package input reads and validates a folder prepared per the input
// specification (docs/input-spec.md) into the source package the library
// builds a package from.
//
// It is the CLI's frontend to the library: the library (build/, profiles/,
// sip/) never imports it, and systems embedding the library construct the
// same build.SourcePackage directly from their own stores instead of preparing
// a folder. The folder is one transport, not the API.
package input

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/build"
)

// Read walks and validates the folder at root against the input
// specification and returns the source package it holds, the value the
// builder takes. mapper is the profile's: it maps the rows of a
// description.csv onto the profile's description. documentSpec describes the
// profile's descriptive document, which a folder may supply in place of the
// rows. Where the library has a rule for what Read reads
// (a description's Validate and ValidateRequired, the metadata model's
// check of a supplied document's root), Read runs that same rule and
// reports its findings with file and line. Every MUST violation is
// collected and returned together as a Violations error; when the error is
// non-nil the returned source package is incomplete and must not be built.
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
