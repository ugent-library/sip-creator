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
// builder takes. vocabulary is the profile's: it says what the rows of a
// description.csv mean. Where the library has a rule for what Read reads
// (a description's Validate and ValidateRequired, the encoder's check of a
// supplied document's root), Read runs that same rule and reports its
// findings with file and line. Every MUST violation is collected and
// returned together as a Violations error; when the error is non-nil the
// returned source package is incomplete and must not be built.
func Read(root string, vocabulary Vocabulary) (*build.SourcePackage, error) {
	if vocabulary == nil {
		return nil, errors.New("no vocabulary: pass the profile's vocabulary, which says what the rows of description.csv mean")
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

	w := &folderWalker{root: abs, vocabulary: vocabulary}
	// A vocabulary whose profile takes a supplied document says so by
	// implementing DocumentFormat; under any other, no file name is the
	// document's.
	w.document, _ = vocabulary.(DocumentFormat)
	source := w.read()
	if len(w.violations) > 0 {
		return source, w.violations
	}
	return source, nil
}
