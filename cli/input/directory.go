package input

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"golang.org/x/text/unicode/norm"
)

// directory is the input folder under inspection: the root all messages and
// report keys are relative to, the findings collected so far, and the
// profile the folder is read as.
type directory struct {
	root       string
	violations Violations
	warnings   []string
	profile    build.Definition
}

// Reserved top-level names. Reserved names inside a representation are
// a subset.
const (
	descriptionName        = "description.csv"
	representationsName    = "representations"
	representationsCSVName = "representations.csv"
	documentationName      = "documentation"
	premisName             = "premis"
	sidecarName            = "siegfried.json"
)

// read walks the top level: the reserved names each go to their decoder or
// collector, everything else is content, whose place depends on whether a
// representations/ folder exists.
func (d *directory) read() *Package {
	pkg := &Package{Root: d.root}

	var content []os.DirEntry
	var description, repsDir, repsCSV string

	for _, e := range d.readDir(d.root) {
		// Reserved names are ASCII, which NFC normalization never alters,
		// so comparing unnormalized names is exact.
		name := e.Name()
		src := filepath.Join(d.root, e.Name())
		switch name {
		case descriptionName:
			if e.IsDir() {
				d.violate("%s is a folder; the reserved name is for the descriptive rows file", name)
				continue
			}
			description = src
		case representationsName:
			if !e.IsDir() {
				d.violate("representations is a file; the reserved name is for the folder of representations")
				continue
			}
			repsDir = src
		case representationsCSVName:
			if e.IsDir() {
				d.violate("representations.csv is a folder; the reserved name is for the representations file")
				continue
			}
			repsCSV = src
		case documentationName:
			if !e.IsDir() {
				d.violate("documentation is a file; the reserved name is for a folder")
				continue
			}
			pkg.Documentation = d.collectFiles(src)
		case premisName:
			if !e.IsDir() {
				d.violate("premis is a file; the reserved name is for a folder")
				continue
			}
			pkg.Premis = d.collectPremisFiles(src)
		case sidecarName:
			if e.IsDir() {
				d.violate("siegfried.json is a folder; the reserved name is for the characterization report")
				continue
			}
			pkg.Characterization = d.decodeSidecar(src)
		default:
			content = append(content, e)
		}
	}

	pkg.Description = d.decodeDescription(description, true)

	if repsDir != "" {
		// With a representations/ folder, all content lives inside it;
		// only the reserved names may sit beside it.
		for _, e := range content {
			d.violate("%s: content must live inside representations/ when that folder exists (only documentation/ and premis/ may sit beside it)", e.Name())
		}
		pkg.Representations = d.readRepresentations(repsDir)
		if repsCSV != "" {
			pkg.Representations = d.applyRepresentations(repsCSV, pkg.Representations)
		}
	} else {
		if repsCSV != "" {
			d.violate("representations.csv requires a representations/ folder; a flat folder is one representation named after the folder itself")
		}
		pkg.Representations = []Representation{d.readFlatRepresentation(content)}
	}

	return pkg
}

func (d *directory) readRepresentations(dir string) []Representation {
	var reps []Representation
	for _, e := range d.readDir(dir) {
		if !e.IsDir() {
			d.violate("representations/%s: only representation folders may sit directly inside representations/", e.Name())
			continue
		}
		name := e.Name()
		// The folder-name rule is the library's name rule:
		// one source of truth for what a representation may be called.
		if err := build.ValidateRepresentationName(name); err != nil {
			// Still read the folder: the naming fix shouldn't hide any
			// findings inside it (collect-all).
			d.violate("representations/%s: %v", e.Name(), err)
		}
		reps = append(reps, d.readRepresentation(filepath.Join(dir, e.Name()), name))
	}
	if len(reps) == 0 {
		d.violate("representations/ contains no representation folders: a package needs at least one version of the content")
	}
	return reps
}

func (d *directory) readRepresentation(dir, name string) Representation {
	rep := Representation{Name: name}
	var description string
	for _, e := range d.readDir(dir) {
		// Reserved names are ASCII, which NFC normalization never alters,
		// so comparing unnormalized names is exact.
		name := e.Name()
		src := filepath.Join(dir, e.Name())
		switch name {
		case descriptionName:
			if e.IsDir() {
				d.violate("%s is a folder; the reserved name is for the descriptive rows file", d.rel(src))
				continue
			}
			description = src
		case documentationName:
			if !e.IsDir() {
				d.violate("%s is a file; the reserved name is for a folder", d.rel(src))
				continue
			}
			rep.Documentation = d.collectFiles(src)
		case premisName:
			if !e.IsDir() {
				d.violate("%s is a file; the reserved name is for a folder", d.rel(src))
				continue
			}
			rep.Premis = d.collectPremisFiles(src)
		default:
			if e.IsDir() {
				d.walkContent(dir, src, &rep.Files)
				continue
			}
			rep.Files = append(rep.Files, d.newFile(dir, src))
		}
	}
	rep.Description = d.decodeDescription(description, false)
	if len(rep.Files) == 0 {
		d.violate("%s: the representation contains no content files", d.rel(dir))
	}
	return rep
}

// readFlatRepresentation handles the simple case: no
// representations/ folder, so every non-reserved entry is the content of a
// single representation, named after the input folder itself.
func (d *directory) readFlatRepresentation(entries []os.DirEntry) Representation {
	name := filepath.Base(d.root)
	// The input folder's name becomes the representation's package-side
	// name, so it must satisfy the same rule as a folder under
	// representations/.
	if err := build.ValidateRepresentationName(name); err != nil {
		d.violate("the folder name names the single representation: %v", err)
	}
	rep := Representation{Name: name}
	for _, e := range entries {
		src := filepath.Join(d.root, e.Name())
		if e.IsDir() {
			d.walkContent(d.root, src, &rep.Files)
			continue
		}
		rep.Files = append(rep.Files, d.newFile(d.root, src))
	}
	if len(rep.Files) == 0 {
		d.violate("the folder contains no content files")
	}
	return rep
}

// collectFiles gathers every file under dir recursively with Path relative
// to dir; documentation/, premis/, and representation content all collect
// the same way.
func (d *directory) collectFiles(dir string) []File {
	var files []File
	d.walkContent(dir, dir, &files)
	return files
}

// collectPremisFiles collects a premis/ folder (package- or
// representation-level, same rule both places) and flags the one
// transport-level premis rule: premis.xml belongs to the generated
// document. Content conformance (well-formed premis:premis) is
// deliberately left to assembly.
func (d *directory) collectPremisFiles(dir string) []File {
	files := d.collectFiles(dir)
	for _, f := range files {
		if path.Base(f.Path) == "premis.xml" {
			d.violate("%s: premis.xml is reserved for the generated preservation document; rename the received file", f.Rel)
		}
	}
	return files
}

func (d *directory) walkContent(base, dir string, files *[]File) {
	for _, e := range d.readDir(dir) {
		src := filepath.Join(dir, e.Name())
		if e.IsDir() {
			d.walkContent(base, src, files)
			continue
		}
		*files = append(*files, d.newFile(base, src))
	}
}

func (d *directory) newFile(base, src string) File {
	relRoot, err := filepath.Rel(d.root, src)
	if err != nil {
		relRoot = src
	}
	relBase, err := filepath.Rel(base, src)
	if err != nil {
		relBase = filepath.Base(src)
	}
	return File{
		Source: src,
		// Rel is not NFC-normalized: it must match the filename exactly
		// as the characterization report recorded it.
		Rel:  path.Clean(filepath.ToSlash(relRoot)),
		Path: norm.NFC.String(filepath.ToSlash(relBase)),
	}
}

// decodeSidecar decodes the optional pre-computed characterization report.
// Decode strictness is ADR-0009's: a present report must parse; per-entry
// verification (the MD5 check) stays with the assembler, which knows which
// entries it needs.
func (d *directory) decodeSidecar(src string) characterization.Report {
	f, err := os.Open(src)
	if err != nil {
		d.violate("siegfried.json: %v", err)
		return nil
	}
	defer f.Close()

	report, err := characterization.DecodeSiegfried(f)
	if err != nil {
		d.violate("siegfried.json: %v; regenerate it from the input root with: sf -hash md5 -json .", err)
		return nil
	}
	return report
}

// readDir lists dir applying the rules that hold everywhere in the input
// tree: symbolic links are a violation and are never
// followed; OS artifacts are silently ignored (never packaged, never
// warned about); and two names identical after NFC normalization are a
// collision, because such pairs can coexist on non-normalizing filesystems
// and would collide in the package.
//
// os.ReadDir lists lexically, so every file list built over it is in
// deterministic traversal order. That order carries no meaning (neither
// CSIP nor meemoo assigns semantics to file order; explicit sequencing is
// a deferred manifest feature), but it must be stable:
// METS emission and scripts/reference-diff.sh depend on run-to-run identical order.
func (d *directory) readDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		d.violate("%s: %v", d.rel(dir), err)
		return nil
	}

	seen := make(map[string]string, len(entries))
	var kept []os.DirEntry
	for _, e := range entries {
		if isOSArtifact(e.Name()) {
			continue
		}
		if e.Type()&fs.ModeSymlink != 0 {
			d.violate("%s is a symbolic link; symbolic links are not allowed anywhere in an input folder", d.rel(filepath.Join(dir, e.Name())))
			continue
		}
		name := norm.NFC.String(e.Name())
		if prev, ok := seen[name]; ok {
			d.violate("%s: %q and %q are the same name after Unicode normalization; rename one", d.rel(dir), prev, e.Name())
			continue
		}
		seen[name] = e.Name()
		kept = append(kept, e)
	}
	return kept
}

// isOSArtifact reports whether name is an OS artifact to ignore silently.
func isOSArtifact(name string) bool {
	if strings.HasPrefix(name, "._") {
		return true
	}
	switch strings.ToLower(name) {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	}
	return false
}

// rel makes a path presentable in a violation message: relative to the
// input root, slash-separated.
func (d *directory) rel(p string) string {
	rel, err := filepath.Rel(d.root, p)
	if err != nil {
		return p
	}
	if rel == "." {
		return "the input folder"
	}
	return filepath.ToSlash(rel)
}
