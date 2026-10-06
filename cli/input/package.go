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
	"github.com/ugent-library/sip-creator/sip"
)

// ReadDetails holds what a read of an input folder found beyond the source
// package: the descriptive rows as the producer wrote them, whether the
// folder supplies representations.csv, and what the reader left out. It
// describes the folder for an operator; the build does not use it.
type ReadDetails struct {
	// PackageRows are the rows of the package-level description.csv in file
	// order, with Key as the row spells it, before the profile's mapping.
	// Empty when the package level has no description.csv, or one that
	// could not be read.
	PackageRows []sip.Term
	// RepresentationRows are the rows of each representation's
	// description.csv, by representation name, in the same form as
	// PackageRows. A representation without one has no entry.
	RepresentationRows map[string][]sip.Term
	// RepresentationsCSV reports whether the folder supplies a
	// representations.csv next to its representations/ folder.
	RepresentationsCSV bool
	// SkippedOSArtifacts lists the operating system files, such as
	// .DS_Store, that the reader left out without a violation, by slash
	// path relative to the input folder, in the order the reader met them.
	SkippedOSArtifacts []string
}

// Read walks and validates the folder at root against the input
// specification and returns the source package it holds, the value the
// builder takes, and the details of the read that the source package does
// not carry. mapper is the profile's: it maps the rows of a
// description.csv onto the profile's description. documentSpec describes the
// profile's descriptive document, which a folder may supply in place of the
// rows. Where the library has a rule for what Read reads
// (a description's Validate and ValidateRequired, the metadata model's
// check of a supplied document's root), Read runs that same rule and
// reports its findings with file and line. Every MUST violation is
// collected and returned together as a Violations error; when the error is
// non-nil the returned source package is incomplete and must not be built,
// and the details are as far as the reader got.
func Read(root string, mapper Mapper, documentSpec DocumentSpec) (*build.SourcePackage, ReadDetails, error) {
	if mapper == nil {
		return nil, ReadDetails{}, errors.New("no mapper: pass the profile's mapper, which maps the rows of description.csv onto its description")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, ReadDetails{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, ReadDetails{}, fmt.Errorf("input folder: %w", err)
	}
	if !info.IsDir() {
		return nil, ReadDetails{}, fmt.Errorf("input folder %s is a file, not a folder", root)
	}

	r := &folderReader{root: abs}
	format, takesDocument := documentSpec.Model.(build.DocumentFormat)
	if takesDocument {
		r.documentName = documentSpec.Name
	} else {
		r.refusedDocumentName = documentSpec.Name
	}

	source, inv := r.walk()
	details := r.decode(source, inv, mapper, format)
	details.SkippedOSArtifacts = r.skippedOSArtifacts
	if len(r.violations) > 0 {
		return source, details, r.violations
	}
	return source, details, nil
}
