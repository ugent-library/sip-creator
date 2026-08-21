package profiles

import (
	"fmt"
	"path"
	"regexp"

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
// supplies it. The CLI's input convention is one producer; an embedding
// system constructing it from its own stores is another.
type SourceRepresentation struct {
	Label string       // producer's label; the profile decides package-side naming
	Files []SourceFile // content files, in packaging order
	// Descriptive optionally describes this version only (input spec §3):
	// identity (identifier, title) is not required here; the package-level
	// descriptive carries the work's identity.
	Descriptive metadata.Terms
	// Premis optionally supplies received preservation documents about
	// this representation (input spec §5): copied, never parsed. Each
	// must be a well-formed premis:premis document (checked at assembly).
	Premis []SourceFile
	// Documentation optionally documents this representation only
	// (input spec §4).
	Documentation []SourceFile
}

// Input is one package's source material, given as data, not files to parse:
// descriptive metadata as terms, characterization as a decoded report,
// essence and documentation as source paths. The CLI's folder convention
// (cli/input) is one transport producing these values; embedding systems
// construct them directly.
//
// Build takes ownership of the data: the descriptive terms are mutated
// (the entity identifier is swapped in) during assembly.
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
	// whole package (input spec §5): copied, never parsed. Each must be a
	// well-formed premis:premis document (checked at assembly).
	Premis []SourceFile
	// Characterization optionally supplies a pre-decoded characterization
	// report; nil means the build proceeds without format info (ADR-0009).
	// Fully strict when present: every essence file must have a
	// checksum-bound entry.
	Characterization characterization.Report
}

// labelRx is the POSIX portable filename character set: a label satisfying
// it is usable verbatim as a directory name, zip entry, METS href, and
// OBJID on any filesystem, with no percent-encoding machinery.
var labelRx = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateRepresentationLabel returns why a representation label cannot be
// used: empty, or characters outside the portable set.
func ValidateRepresentationLabel(label string) error {
	if !labelRx.MatchString(label) {
		return fmt.Errorf("representation label %q may only contain letters, digits, and . _ -", label)
	}
	return nil
}

// Validate reports the first invariant the input breaks. These are the
// graph rules every producer must satisfy: the folder convention enforces
// them with operator-phrased Violations before building; embedding callers
// hit them here (the input-convention plan's "validation splits in two").
// Fail-fast: one error, developer-phrased.
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
	// The identifier is checked here mechanically: assembly swaps it for
	// the object identifier, so the graph cannot build without one. All
	// other requiredness is profile policy: Definition.RequiredElements,
	// checked by Build.
	if in.Descriptive.LocalIdentifier() == "" {
		return fmt.Errorf("descriptive metadata carries no dcterms:identifier; the local identifier is required")
	}

	if len(in.Representations) == 0 {
		return fmt.Errorf("no representations supplied: a package needs at least one version of the content")
	}
	labels := make(map[string]bool, len(in.Representations))
	for _, r := range in.Representations {
		if err := ValidateRepresentationLabel(r.Label); err != nil {
			return err
		}
		if labels[r.Label] {
			return fmt.Errorf("representation label %q supplied twice", r.Label)
		}
		labels[r.Label] = true
		if len(r.Files) == 0 {
			return fmt.Errorf("representation %q has no content files", r.Label)
		}
		if err := validateFiles(fmt.Sprintf("representation %q", r.Label), r.Files); err != nil {
			return err
		}
		if r.Descriptive != nil {
			if err := r.Descriptive.Validate(); err != nil {
				return fmt.Errorf("representation %q descriptive: %w", r.Label, err)
			}
		}
		if err := validatePremisNames(fmt.Sprintf("representation %q", r.Label), r.Premis); err != nil {
			return err
		}
		if err := validateFiles(fmt.Sprintf("representation %q documentation", r.Label), r.Documentation); err != nil {
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
