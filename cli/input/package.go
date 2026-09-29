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
// specification and returns the source package it holds, the value the builder
// takes. Every MUST violation is collected and returned together as a
// Violations error; when the error is non-nil the returned source package is
// incomplete and must not be built.
func (r *Reader) Read(root string) (*build.SourcePackage, error) {
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
	m := d.read()
	if len(d.violations) > 0 {
		return m, d.violations
	}
	return m, nil
}
