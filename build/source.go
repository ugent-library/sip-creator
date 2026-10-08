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
	// Key is the file's key in the characterization report: its path
	// relative to the input root, with slashes, as the report records it.
	// It is set only when a characterization report is supplied.
	Key string
	// Path is the file's path, with slashes, under the directory its kind
	// goes to in its container: data/ for essence, documentation/ for
	// documentation and metadata/preservation/ for received PREMIS. It must
	// satisfy ValidateXMLText, because the METS and PREMIS documents carry
	// it.
	Path string
}

// SourceRepresentation is one version of the content, such as a
// preservation master or an access copy, as it is handed to Builder.Build:
// its name and label, its files on disk and, optionally, a description of
// this version only.
type SourceRepresentation struct {
	// Name is the representation's name in the package: the directory
	// under representations/ and the representation METS OBJID. It is
	// required. It must satisfy ValidateRepresentationName and be unique
	// within the package.
	Name string
	// Label is the display name, written as the representation METS
	// mets/@LABEL. When it is empty, the label is Name. It must satisfy
	// ValidateXMLText.
	Label string
	// Type is the representation's type, which the representation METS
	// declares under a profile with Definition.EmitRepresentationType set.
	// When it is empty, the type is the label. Under a profile with
	// Definition.RepresentationTypes, the type is Name, and Type must be
	// empty or equal to Name. It must satisfy ValidateXMLText.
	Type string
	// Files are the content files, in packaging order.
	Files []SourceFile
	// Description optionally describes this representation only. It need
	// not state an identifier or a title, because the package-level
	// description carries the work's identity. Its concrete type must be
	// the profile's description type, or an EncodedDescription where the
	// profile accepts one.
	Description sip.Description
	// Premis optionally lists received preservation documents about this
	// representation, which the package carries as received. Each must be
	// a well-formed premis:premis document.
	Premis []SourceFile
	// Documentation optionally documents this representation only.
	Documentation []SourceFile
}

// label returns Label, or Name when Label is empty.
func (sr SourceRepresentation) label() string {
	if sr.Label != "" {
		return sr.Label
	}
	return sr.Name
}

// resolvedType returns Type, or the label when Type is empty.
func (sr SourceRepresentation) resolvedType() string {
	if sr.Type != "" {
		return sr.Type
	}
	return sr.label()
}

// SourcePackage is one package as it is handed to Builder.Build, given as
// data, not files to parse: descriptive metadata as a Description,
// characterization as a decoded report, essence and documentation as
// source paths.
//
// Build takes ownership of the data: under a profile that swaps
// identifiers, such as Meemoo's, Builder.Build writes the entity
// identifier into the description.
type SourcePackage struct {
	// PackageIdentifier optionally supplies the package identifier instead
	// of minting one. This is how an update reuses the original package's
	// mets/@OBJID. When set, it must take the uuid-<uuid> form.
	PackageIdentifier string
	// RecordStatus optionally declares this package's
	// metsHdr/@RECORDSTATUS, one of the sip.RecordStatus constants. When it
	// is empty, the profile's value applies. The profiles in this module
	// declare none, which the E-ARK SIP spec reads as NEW. An update status
	// says this package supplements or replaces an earlier one, whose
	// identifier must then travel in PackageIdentifier.
	RecordStatus sip.RecordStatus
	// ContentCategory optionally declares this package's mets/@TYPE, its
	// content category in the CSIP vocabulary. When it is empty, the
	// profile's value applies. It must satisfy ValidateXMLText.
	ContentCategory string
	// Description is the package-level descriptive metadata. It is
	// required. Its concrete type must be the profile's description type,
	// or an EncodedDescription where the profile accepts one.
	Description sip.Description
	// Representations is the content. How many a package needs is the
	// profile's rule (Definition.MinRepresentations). A package without
	// any carries metadata only.
	Representations []SourceRepresentation
	// Documentation optionally documents the whole package.
	Documentation []SourceFile
	// Premis optionally lists received preservation documents about the
	// whole package, which the package carries as received. Each must be a
	// well-formed premis:premis document.
	Premis []SourceFile
	// Characterization optionally supplies a decoded characterization
	// report. When it is nil, the package records no format information
	// (ADR-0009). When present, every essence file must have an entry
	// without a characterizer error and with an MD5. That MD5 is the
	// checksum the package declares for the file, taken as given: the
	// caller vouches that the report describes the files (ADR-0032).
	Characterization characterization.Report
}

// nameRx is the POSIX portable filename character set: a name in it,
// other than . and .., is usable verbatim as a directory name, zip entry,
// METS href, and OBJID on any filesystem, without percent-encoding.
var nameRx = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateRepresentationName checks that name is not empty, holds only
// characters of the POSIX portable filename character set, and is neither
// "." nor "..". Those two name the representations/ directory itself or
// the package root rather than a directory inside it. It returns an error
// if a check fails.
func ValidateRepresentationName(name string) error {
	if !nameRx.MatchString(name) {
		return fmt.Errorf("representation name %q may only contain letters, digits, and . _ -", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("representation name %q names a directory outside representations/; choose another name", name)
	}
	return nil
}

// ValidateXMLText checks that value can be written into the package's XML
// documents: it is valid UTF-8 and holds only characters XML 1.0 allows.
// It returns an error if value is not valid UTF-8 or holds a character
// XML 1.0 excludes, such as a control character other than tab, line feed
// and carriage return. Escaping cannot carry such a character. It would be
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

// Validate checks the rules every source package must satisfy, whatever
// its profile and however it was made. It returns an error naming the
// first rule the package breaks, phrased for a developer. The profile's
// own rules are Definition.ValidateSource's.
func (sp *SourcePackage) Validate() error {
	if sp.PackageIdentifier != "" {
		if err := sip.ValidateIdentifier(sp.PackageIdentifier); err != nil {
			return err
		}
	}
	if sp.RecordStatus != "" && !sp.RecordStatus.IsValid() {
		return fmt.Errorf("record status %q is not in the SIP3 vocabulary; use the sip.RecordStatus constants", sp.RecordStatus)
	}
	// An update status names the earlier package by reusing its
	// identifier. Without one, the package claims to update a package
	// it does not name. The reverse is allowed: a NEW package may carry an
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
	// The description's rules run here, before any write, and the metadata
	// models do not check them again. A package-level description must also
	// state the keys its standard requires, such as an identifier and a
	// title. A representation's description need not.
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

// validatePremisNames checks a list of received PREMIS files with
// validateFiles, and checks that no file is named premis.xml, the name of
// the generated preservation document. It returns an error naming the
// first rule a file breaks.
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
