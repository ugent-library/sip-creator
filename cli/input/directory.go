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
// report keys are relative to, the findings collected so far, the
// vocabulary that gives the rows of a description.csv their meaning, and,
// when the profile takes a supplied document, the DocumentVocabulary that
// names and judges it.
type directory struct {
	root       string
	violations Violations
	vocabulary Vocabulary
	// document names the file reserved for a supplied descriptive document
	// and judges its root; nil under a profile that takes rows only.
	document DocumentVocabulary
}

// Reserved top-level names. Reserved names inside a representation are
// a subset. The profile's document name, when it has one, is reserved at
// both levels too; the vocabulary supplies it (isDocumentName). Every
// reserved name is ASCII, which NFC normalization never alters, so
// comparing an unnormalized directory entry name to one is exact.
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
func (d *directory) read() *build.SourcePackage {
	source := &build.SourcePackage{}

	var content []os.DirEntry
	var description, document, repsDir, repsCSV string

	for _, e := range d.readDir(d.root) {
		name := e.Name()
		src := filepath.Join(d.root, e.Name())
		if d.isDocumentName(name) {
			if d.expectFile(e, src, "the supplied descriptive document") {
				document = src
			}
			continue
		}
		switch name {
		case descriptionName:
			if d.expectFile(e, src, "the descriptive rows file") {
				description = src
			}
		case representationsName:
			if d.expectFolder(e, src, "the folder of representations") {
				repsDir = src
			}
		case representationsCSVName:
			if d.expectFile(e, src, "the representations file") {
				repsCSV = src
			}
		case documentationName:
			if d.expectFolder(e, src, "a folder") {
				source.Documentation = d.collectFiles(src)
			}
		case premisName:
			if d.expectFolder(e, src, "a folder") {
				source.Premis = d.collectPremisFiles(src)
			}
		case sidecarName:
			if d.expectFile(e, src, "the characterization report") {
				source.Characterization = d.decodeSidecar(src)
			}
		default:
			content = append(content, e)
		}
	}

	source.Description = d.description(description, document, true)

	if repsDir != "" {
		// With a representations/ folder, all content lives inside it;
		// only the reserved names may sit beside it.
		for _, e := range content {
			d.violate("%s: content must live inside representations/ when that folder exists (only the reserved names of the input specification may sit beside it)", e.Name())
		}
		source.Representations = d.readRepresentations(repsDir)
		if repsCSV != "" {
			source.Representations = d.applyRepresentations(repsCSV, source.Representations)
		}
	} else {
		if repsCSV != "" {
			d.violate("representations.csv requires a representations/ folder; a flat folder is one representation named after the folder itself")
		}
		source.Representations = []build.SourceRepresentation{d.readFlatRepresentation(content)}
	}

	return source
}

func (d *directory) readRepresentations(dir string) []build.SourceRepresentation {
	var reps []build.SourceRepresentation
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

func (d *directory) readRepresentation(dir, name string) build.SourceRepresentation {
	rep := build.SourceRepresentation{Name: name}
	var description, document string
	for _, e := range d.readDir(dir) {
		name := e.Name()
		src := filepath.Join(dir, e.Name())
		if d.isDocumentName(name) {
			if d.expectFile(e, src, "the supplied descriptive document") {
				document = src
			}
			continue
		}
		switch name {
		case descriptionName:
			if d.expectFile(e, src, "the descriptive rows file") {
				description = src
			}
		case documentationName:
			if d.expectFolder(e, src, "a folder") {
				rep.Documentation = d.collectFiles(src)
			}
		case premisName:
			if d.expectFolder(e, src, "a folder") {
				rep.Premis = d.collectPremisFiles(src)
			}
		default:
			if e.IsDir() {
				d.walkContent(dir, src, &rep.Files)
				continue
			}
			rep.Files = append(rep.Files, d.newFile(dir, src))
		}
	}
	rep.Description = d.description(description, document, false)
	if len(rep.Files) == 0 {
		d.violate("%s: the representation contains no content files", d.rel(dir))
	}
	return rep
}

// isDocumentName reports whether name is the file name reserved for the
// profile's supplied descriptive document. Under a profile that takes
// none, no name is: a dc.xml under basic is content like any other file.
func (d *directory) isDocumentName(name string) bool {
	return d.document != nil && name == d.document.DocumentName()
}

// expectFile reports whether the entry at src, which has a reserved name,
// is a file, and records a violation naming what the name holds when it is
// a folder.
func (d *directory) expectFile(e os.DirEntry, src, holds string) bool {
	if !e.IsDir() {
		return true
	}
	d.violate("%s is a folder; the reserved name is for %s", d.rel(src), holds)
	return false
}

// expectFolder reports whether the entry at src, which has a reserved
// name, is a folder, and records a violation naming what the name holds
// when it is a file.
func (d *directory) expectFolder(e os.DirEntry, src, holds string) bool {
	if e.IsDir() {
		return true
	}
	d.violate("%s is a file; the reserved name is for %s", d.rel(src), holds)
	return false
}

// readFlatRepresentation handles the simple case: no
// representations/ folder, so every non-reserved entry is the content of a
// single representation, named after the input folder itself.
func (d *directory) readFlatRepresentation(entries []os.DirEntry) build.SourceRepresentation {
	name := filepath.Base(d.root)
	// The input folder's name becomes the representation's package-side
	// name, so it must satisfy the same rule as a folder under
	// representations/.
	if err := build.ValidateRepresentationName(name); err != nil {
		d.violate("the folder name names the single representation: %v", err)
	}
	rep := build.SourceRepresentation{Name: name}
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
func (d *directory) collectFiles(dir string) []build.SourceFile {
	var files []build.SourceFile
	d.walkContent(dir, dir, &files)
	return files
}

// collectPremisFiles collects a premis/ folder (package- or
// representation-level, same rule both places) and flags the one
// transport-level premis rule: premis.xml belongs to the generated
// document. Content conformance (a premis:premis document) is
// deliberately left to assembly.
func (d *directory) collectPremisFiles(dir string) []build.SourceFile {
	files := d.collectFiles(dir)
	for _, f := range files {
		if path.Base(f.Path) == "premis.xml" {
			d.violate("%s: premis.xml is reserved for the generated preservation document; rename the received file", f.Key)
		}
	}
	return files
}

func (d *directory) walkContent(base, dir string, files *[]build.SourceFile) {
	for _, e := range d.readDir(dir) {
		src := filepath.Join(dir, e.Name())
		if e.IsDir() {
			d.walkContent(base, src, files)
			continue
		}
		*files = append(*files, d.newFile(base, src))
	}
}

func (d *directory) newFile(base, src string) build.SourceFile {
	relRoot, err := filepath.Rel(d.root, src)
	if err != nil {
		relRoot = src
	}
	relBase, err := filepath.Rel(base, src)
	if err != nil {
		relBase = filepath.Base(src)
	}
	return build.SourceFile{
		Source: src,
		// Key is not NFC-normalized: it must match the filename exactly
		// as the characterization report recorded it.
		Key:  path.Clean(filepath.ToSlash(relRoot)),
		Path: norm.NFC.String(filepath.ToSlash(relBase)),
	}
}

// decodeSidecar decodes the optional pre-computed characterization report.
// A present report must parse (ADR-0009); the assembler verifies each
// entry's MD5, because only it knows which entries it needs.
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
// CSIP nor Meemoo assigns semantics to file order; explicit sequencing is
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
