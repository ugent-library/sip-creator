package profiles

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/metadata"
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
	// (representation data/, documentation/), slash-separated.
	Path string
}

// SourceRepresentation is one version of the content, as the caller
// supplies it.
type SourceRepresentation struct {
	// Name is the package-side name: the directory under representations/
	// and the representation METS OBJID. Required; must satisfy
	// ValidateRepresentationName, and names must be unique within a package.
	Name string
	// Label is the display name, emitted as the representation METS
	// mets/@LABEL. Optional: empty means the Name. Must satisfy
	// ValidateAttributeText.
	Label string
	// Type is the representation's type, declared in the representation
	// METS content typing by profiles with EmitRepresentationType set.
	// Optional: empty means the resolved Label. Must satisfy
	// ValidateAttributeText.
	Type string
	// Files are the content files, in packaging order.
	Files []SourceFile
	// Descriptive optionally describes this version only: identity
	// (identifier, title) is not required here; the package-level
	// descriptive carries the work's identity.
	Descriptive metadata.Terms
	// Premis optionally supplies received preservation documents about
	// this representation: copied, never parsed. Each must be a
	// well-formed premis:premis document.
	Premis []SourceFile
	// Documentation optionally documents this representation only.
	Documentation []SourceFile
}

// label resolves the display label: Label, or Name when empty. The cascade
// lives here, in the library, so the CLI's representations.csv and an
// embedding caller's direct Input get identical defaulting.
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

// Input is one package's source material, given as data, not files to parse:
// descriptive metadata as terms, characterization as a decoded report,
// essence and documentation as source paths. The CLI's folder convention
// (cli/input) is one transport producing these values; embedding systems
// construct them directly.
//
// Build takes ownership of the data: the descriptive terms may be mutated
// (profiles with SwapObjectIdentifier swap the entity identifier in)
// during assembly.
type Input struct {
	// PackageIdentifier optionally supplies the package identifier instead
	// of minting one; this is how an update reuses the original package's
	// mets/@OBJID. Must take the uuid-<uuid> form when set.
	PackageIdentifier string
	// Descriptive is the package-level descriptive metadata.
	Descriptive metadata.Terms
	// Representations is the content, at least one.
	Representations []SourceRepresentation
	// Documentation optionally documents the whole package.
	Documentation []SourceFile
	// Premis optionally supplies received preservation documents about the
	// whole package: copied, never parsed. Each must be a well-formed
	// premis:premis document.
	Premis []SourceFile
	// Characterization optionally supplies a pre-decoded characterization
	// report; nil means the build proceeds without format info (ADR-0009).
	// Fully strict when present: every essence file must have an entry
	// whose checksum matches the source bytes.
	Characterization characterization.Report
}

// nameRx is the POSIX portable filename character set: a name satisfying
// it is usable verbatim as a directory name, zip entry, METS href, and
// OBJID on any filesystem, with no percent-encoding machinery.
var nameRx = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateRepresentationName returns why a representation name cannot be
// used: empty, or characters outside the portable set.
func ValidateRepresentationName(name string) error {
	if !nameRx.MatchString(name) {
		return fmt.Errorf("representation name %q may only contain letters, digits, and . _ -", name)
	}
	return nil
}

// ValidateAttributeText returns why a value cannot be emitted as a METS
// attribute: the METS templates do no XML escaping, so the XML-active
// characters are rejected rather than escaped. The empty string is fine
// (an empty Label or Type falls back along the defaulting cascade).
func ValidateAttributeText(value string) error {
	if i := strings.IndexAny(value, `<>&"`); i >= 0 {
		return fmt.Errorf("%q contains %q, which cannot be emitted into METS; the characters < > & \" are not allowed", value, value[i])
	}
	return nil
}

// Validate reports the first invariant the input breaks. These are the
// graph rules every producer must satisfy: the folder convention enforces
// them with Violations phrased for the operator before building; embedding
// callers hit them here. Fail-fast: one error, phrased for the developer.
func (in *Input) Validate() error {
	if in.PackageIdentifier != "" {
		if err := sip.ValidateIdentifier(in.PackageIdentifier); err != nil {
			return err
		}
	}
	if len(in.Descriptive) == 0 {
		return fmt.Errorf("no descriptive metadata supplied")
	}
	if err := in.Descriptive.Validate(); err != nil {
		return err
	}
	// The identifier is the package's one required identity in the
	// descriptive document: profiles that swap (SwapObjectIdentifier) need
	// a slot to overwrite, the others emit it as the identifier consumers
	// find the package by (ADR-0012). All other requiredness is profile
	// policy: Definition.RequiredElements, checked by Build.
	if in.Descriptive.LocalIdentifier() == "" {
		return fmt.Errorf("descriptive metadata carries no dcterms:identifier; the local identifier is required")
	}

	if len(in.Representations) == 0 {
		return fmt.Errorf("no representations supplied: a package needs at least one version of the content")
	}
	names := make(map[string]bool, len(in.Representations))
	for _, r := range in.Representations {
		if err := ValidateRepresentationName(r.Name); err != nil {
			return err
		}
		if names[r.Name] {
			return fmt.Errorf("representation name %q supplied twice", r.Name)
		}
		names[r.Name] = true
		if err := ValidateAttributeText(r.Label); err != nil {
			return fmt.Errorf("representation %q label: %w", r.Name, err)
		}
		if err := ValidateAttributeText(r.Type); err != nil {
			return fmt.Errorf("representation %q type: %w", r.Name, err)
		}
		if len(r.Files) == 0 {
			return fmt.Errorf("representation %q has no content files", r.Name)
		}
		if err := validateFiles(fmt.Sprintf("representation %q", r.Name), r.Files); err != nil {
			return err
		}
		if r.Descriptive != nil {
			if err := r.Descriptive.Validate(); err != nil {
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

	if err := validateFiles("documentation", in.Documentation); err != nil {
		return err
	}
	return validatePremisNames("package", in.Premis)
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
		if paths[f.Path] {
			return fmt.Errorf("%s: two files share the logical path %q", container, f.Path)
		}
		paths[f.Path] = true
	}
	return nil
}
