package build

import (
	"fmt"
	"path"
	"regexp"
	"unicode/utf8"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/sip"
)

// SourceFile is one input file for the build: where its bytes live now and
// where they land inside their container.
type SourceFile struct {
	// Source is the absolute path of the file on disk.
	Source string
	// Key is the characterization report key: the input-root-relative
	// slash path the report records for this file. Leave empty when no
	// characterization report is supplied.
	Key string
	// Path is the logical path relative to the file's container
	// (representation data/, documentation/, premis/), slash-separated.
	// Must satisfy ValidateXMLText: the METS and PREMIS documents carry it.
	Path string
}

// SourceRepresentation is one version of the content, such as a
// preservation master or an access copy, as it is handed to Builder.Build:
// its name and label, its files on disk and, optionally, a description of
// this version only.
type SourceRepresentation struct {
	// Name is the package-side name: the directory under representations/
	// and the representation METS OBJID. Required; must satisfy
	// ValidateRepresentationName, and names must be unique within a package.
	Name string
	// Label is the display name, emitted as the representation METS
	// mets/@LABEL. Optional: empty means the Name. Must satisfy
	// ValidateXMLText.
	Label string
	// Type is the representation's type, declared in the representation
	// METS content typing by profiles with EmitRepresentationType set.
	// Optional: empty means the resolved Label, or the Name under a profile
	// with Definition.RepresentationTypes, where it must equal the Name.
	// Must satisfy ValidateXMLText.
	Type string
	// Files are the content files, in packaging order.
	Files []SourceFile
	// Description optionally describes this version only: identity
	// (identifier, title) is not required here; the package-level
	// description carries the work's identity. Its concrete type must be
	// the profile's descriptive standard (meemoo.Terms for Meemoo
	// profiles, simpledc.Terms for ugent/basic, mods.Record for ugent/bibliographic) or,
	// for the UGent profiles, an EncodedDescription of that standard.
	Description sip.Description
	// Premis optionally supplies received preservation documents about
	// this representation: copied, never parsed. Each must be a
	// well-formed premis:premis document.
	Premis []SourceFile
	// Documentation optionally documents this representation only.
	Documentation []SourceFile
}

// label resolves the display label: Label, or Name when empty. The cascade
// lives here, in the library, so a representation gets the same label
// whether it comes from the CLI's representations.csv or from a
// SourcePackage built directly in Go.
func (sr SourceRepresentation) label() string {
	if sr.Label != "" {
		return sr.Label
	}
	return sr.Name
}

// resolvedType resolves the representation's type: Type, or the resolved
// label when empty.
func (sr SourceRepresentation) resolvedType() string {
	if sr.Type != "" {
		return sr.Type
	}
	return sr.label()
}

// SourcePackage is one package as it is handed to Builder.Build, given as
// data, not files to parse: descriptive metadata as a Description,
// characterization as a decoded report, essence and documentation as
// source paths. The CLI's input folder (cli/input) is one way to produce
// these values; a program that keeps its content and metadata elsewhere,
// such as in a database, constructs them directly.
//
// Build takes ownership of the data: under a profile that swaps
// identifiers, such as Meemoo's, assembly writes the entity identifier
// into the description.
type SourcePackage struct {
	// PackageIdentifier optionally supplies the package identifier instead
	// of minting one; this is how an update reuses the original package's
	// mets/@OBJID. Must take the uuid-<uuid> form when set.
	PackageIdentifier string
	// RecordStatus optionally declares this package's metsHdr/@RECORDSTATUS,
	// one of the sip.RecordStatus constants. Empty means the profile's
	// value; profiles declare none, which the E-ARK SIP spec reads as NEW.
	// An update-class status says this package supplements or replaces an
	// earlier one, whose identifier must then travel in PackageIdentifier.
	RecordStatus sip.RecordStatus
	// ContentCategory optionally declares this package's mets/@TYPE, its
	// content category in the CSIP vocabulary; empty means the profile's
	// value. Must satisfy ValidateXMLText.
	ContentCategory string
	// Description is the package-level descriptive metadata. Its concrete
	// type must be the profile's descriptive standard (meemoo.Terms for
	// Meemoo profiles, simpledc.Terms for ugent/basic, mods.Record for ugent/bibliographic)
	// or, for the UGent profiles, an EncodedDescription of that standard.
	Description sip.Description
	// Representations is the content. How many a package needs is the
	// profile's rule (Definition.MinRepresentations); a package without
	// any carries metadata only.
	Representations []SourceRepresentation
	// Documentation optionally documents the whole package.
	Documentation []SourceFile
	// Premis optionally supplies received preservation documents about the
	// whole package: copied, never parsed. Each must be a well-formed
	// premis:premis document.
	Premis []SourceFile
	// Characterization optionally supplies a pre-decoded characterization
	// report; nil means the build proceeds without format info (ADR-0009).
	// When present, every essence file must have an entry without a
	// characterizer error and with an MD5. That MD5 is the checksum the
	// package declares for the file, taken as given: the caller vouches
	// that the report describes the files (ADR-0032).
	Characterization characterization.Report
}

// nameRx is the POSIX portable filename character set: a name in it,
// other than . and .., is usable verbatim as a directory name, zip entry,
// METS href, and OBJID on any filesystem, with no percent-encoding
// machinery.
var nameRx = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateRepresentationName returns why a representation name cannot be
// used: empty, characters outside the portable set, or . or .., which name
// the representations/ directory itself or the package root rather than a
// directory inside it.
func ValidateRepresentationName(name string) error {
	if !nameRx.MatchString(name) {
		return fmt.Errorf("representation name %q may only contain letters, digits, and . _ -", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("representation name %q names a directory outside representations/; choose another name", name)
	}
	return nil
}

// ValidateXMLText returns why a value cannot be written into the package's
// XML documents: it is not valid UTF-8, or it holds a character XML 1.0
// excludes, such as a control character other than tab, line feed and
// carriage return. Escaping cannot carry such a character; it would be
// replaced, and a file name would no longer name its file.
func ValidateXMLText(value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%q is not valid UTF-8", value)
	}
	for _, c := range value {
		if !isXMLChar(c) {
			return fmt.Errorf("%q holds the character %U, which XML cannot carry", value, c)
		}
	}
	return nil
}

// isXMLChar reports whether c is a character XML 1.0 allows in a document
// (its Char production).
func isXMLChar(c rune) bool {
	return c == '\t' || c == '\n' || c == '\r' ||
		0x20 <= c && c <= 0xD7FF || 0xE000 <= c && c <= 0xFFFD || 0x10000 <= c && c <= 0x10FFFF
}

// Validate reports the first rule the source package breaks. These are the
// rules every source package must satisfy, however it was made. The CLI's
// input reader checks the same rules on the input folder first and reports
// every violation with its file and line; a SourcePackage built directly
// in Go meets them here. Fail-fast: one error, phrased for the developer.
func (sp *SourcePackage) Validate() error {
	if sp.PackageIdentifier != "" {
		if err := sip.ValidateIdentifier(sp.PackageIdentifier); err != nil {
			return err
		}
	}
	if sp.RecordStatus != "" && !sp.RecordStatus.IsValid() {
		return fmt.Errorf("record status %q is not in the SIP3 vocabulary; use the sip.RecordStatus constants", sp.RecordStatus)
	}
	// An update-class status names an earlier package by reusing its
	// identifier; without one the package claims to update something it
	// does not name. The reverse is allowed: a NEW package may carry an
	// identifier minted before the build, for instance by a system that
	// registers a package before building it.
	if sp.RecordStatus.IsUpdate() && sp.PackageIdentifier == "" {
		return fmt.Errorf("record status %s updates an earlier package, so PackageIdentifier must carry that package's identifier", sp.RecordStatus)
	}
	if err := ValidateXMLText(sp.ContentCategory); err != nil {
		return fmt.Errorf("content category: %w", err)
	}
	if sp.Description == nil {
		return fmt.Errorf("no descriptive metadata supplied")
	}
	// The one place the description's rules run before a write; the
	// metadata models trust it. A package-level description must also state what
	// its standard requires of one (an identifier and a title at least); a
	// representation's need not.
	if err := sp.Description.Validate(); err != nil {
		return fmt.Errorf("descriptive metadata: %w", err)
	}
	if err := sp.Description.ValidateRequired(); err != nil {
		return fmt.Errorf("descriptive metadata: %w", err)
	}

	names := make(map[string]bool, len(sp.Representations))
	for _, r := range sp.Representations {
		if err := ValidateRepresentationName(r.Name); err != nil {
			return err
		}
		if names[r.Name] {
			return fmt.Errorf("representation name %q supplied twice", r.Name)
		}
		names[r.Name] = true
		if err := ValidateXMLText(r.Label); err != nil {
			return fmt.Errorf("representation %q label: %w", r.Name, err)
		}
		if err := ValidateXMLText(r.Type); err != nil {
			return fmt.Errorf("representation %q type: %w", r.Name, err)
		}
		if len(r.Files) == 0 {
			return fmt.Errorf("representation %q has no content files", r.Name)
		}
		if err := validateFiles(fmt.Sprintf("representation %q", r.Name), r.Files); err != nil {
			return err
		}
		if r.Description != nil {
			if err := r.Description.Validate(); err != nil {
				return fmt.Errorf("representation %q descriptive: %w", r.Name, err)
			}
		}
		if err := validatePremisNames(fmt.Sprintf("representation %q", r.Name), r.Premis); err != nil {
			return err
		}
		if err := validateFiles(fmt.Sprintf("representation %q documentation", r.Name), r.Documentation); err != nil {
			return err
		}
	}

	if err := validateFiles("documentation", sp.Documentation); err != nil {
		return err
	}
	return validatePremisNames("package", sp.Premis)
}

// validatePremisNames guards the received-premis file list with the usual
// file rules plus one naming rule: premis.xml is the generated document's
// name, and a received file must never shadow or collide with it.
func validatePremisNames(container string, files []SourceFile) error {
	if err := validateFiles(container+" premis", files); err != nil {
		return err
	}
	for _, f := range files {
		if path.Base(f.Path) == "premis.xml" {
			return fmt.Errorf("%s premis: premis.xml is reserved for the generated preservation document; rename the received file %q", container, f.Path)
		}
	}
	return nil
}

func validateFiles(container string, files []SourceFile) error {
	paths := make(map[string]bool, len(files))
	for _, f := range files {
		if f.Source == "" || f.Path == "" {
			return fmt.Errorf("%s: a file needs both a Source and a Path (got Source %q, Path %q)", container, f.Source, f.Path)
		}
		if err := ValidateXMLText(f.Path); err != nil {
			return fmt.Errorf("%s: %w", container, err)
		}
		if paths[f.Path] {
			return fmt.Errorf("%s: two files share the logical path %q", container, f.Path)
		}
		paths[f.Path] = true
	}
	return nil
}
