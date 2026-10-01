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

// folderWalker walks one input folder into a build.SourcePackage,
// collecting every violation on the way; Read makes one per call.
type folderWalker struct {
	root       string     // all messages and report keys are relative to it
	violations Violations // the findings so far
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
func (w *folderWalker) read() *build.SourcePackage {
	source := &build.SourcePackage{}

	var content []os.DirEntry
	var descriptionPath, documentPath, representationsPath, representationsCSVPath string

	for _, e := range w.readDir(w.root) {
		name := e.Name()
		src := filepath.Join(w.root, e.Name())
		if w.isDocumentName(name) {
			if w.expectFile(e, src, "the supplied descriptive document") {
				documentPath = src
			}
			continue
		}
		switch name {
		case descriptionName:
			if w.expectFile(e, src, "the descriptive rows file") {
				descriptionPath = src
			}
		case representationsName:
			if w.expectFolder(e, src, "the folder of representations") {
				representationsPath = src
			}
		case representationsCSVName:
			if w.expectFile(e, src, "the representations file") {
				representationsCSVPath = src
			}
		case documentationName:
			if w.expectFolder(e, src, "a folder") {
				source.Documentation = w.collectFiles(src)
			}
		case premisName:
			if w.expectFolder(e, src, "a folder") {
				source.Premis = w.collectPremisFiles(src)
			}
		case sidecarName:
			if w.expectFile(e, src, "the characterization report") {
				source.Characterization = w.decodeSidecar(src)
			}
		default:
			content = append(content, e)
		}
	}

	source.Description = w.description(descriptionPath, documentPath, true)

	if representationsPath != "" {
		// With a representations/ folder, all content lives inside it;
		// only the reserved names may sit beside it.
		for _, e := range content {
			w.violate("%s: content must live inside representations/ when that folder exists (only the reserved names of the input specification may sit beside it)", e.Name())
		}
		source.Representations = w.readRepresentations(representationsPath)
		if representationsCSVPath != "" {
			source.Representations = w.applyRepresentations(representationsCSVPath, source.Representations)
		}
	} else {
		if representationsCSVPath != "" {
			w.violate("representations.csv requires a representations/ folder; a flat folder is one representation named after the folder itself")
		}
		source.Representations = []build.SourceRepresentation{w.readFlatRepresentation(content)}
	}

	return source
}

func (w *folderWalker) readRepresentations(dir string) []build.SourceRepresentation {
	var reps []build.SourceRepresentation
	for _, e := range w.readDir(dir) {
		if !e.IsDir() {
			w.violate("representations/%s: only representation folders may sit directly inside representations/", e.Name())
			continue
		}
		name := e.Name()
		// The folder-name rule is the library's name rule:
		// one source of truth for what a representation may be called.
		if err := build.ValidateRepresentationName(name); err != nil {
			// Still read the folder: the naming fix shouldn't hide any
			// findings inside it (collect-all).
			w.violate("representations/%s: %v", e.Name(), err)
		}
		reps = append(reps, w.readRepresentation(filepath.Join(dir, e.Name()), name))
	}
	if len(reps) == 0 {
		w.violate("representations/ contains no representation folders: a package needs at least one version of the content")
	}
	return reps
}

func (w *folderWalker) readRepresentation(dir, repName string) build.SourceRepresentation {
	rep := build.SourceRepresentation{Name: repName}
	var descriptionPath, documentPath string
	for _, e := range w.readDir(dir) {
		name := e.Name()
		src := filepath.Join(dir, e.Name())
		if w.isDocumentName(name) {
			if w.expectFile(e, src, "the supplied descriptive document") {
				documentPath = src
			}
			continue
		}
		switch name {
		case descriptionName:
			if w.expectFile(e, src, "the descriptive rows file") {
				descriptionPath = src
			}
		case documentationName:
			if w.expectFolder(e, src, "a folder") {
				rep.Documentation = w.collectFiles(src)
			}
		case premisName:
			if w.expectFolder(e, src, "a folder") {
				rep.Premis = w.collectPremisFiles(src)
			}
		default:
			if e.IsDir() {
				w.walkContent(dir, src, &rep.Files)
				continue
			}
			rep.Files = append(rep.Files, w.newFile(dir, src))
		}
	}
	rep.Description = w.description(descriptionPath, documentPath, false)
	if len(rep.Files) == 0 {
		w.violate("%s: the representation contains no content files", w.rel(dir))
	}
	return rep
}

// isDocumentName reports whether name is the file name reserved for the
// profile's supplied descriptive document. Under a profile that takes
// none, no name is: a dc.xml under basic is content like any other file.
func (w *folderWalker) isDocumentName(name string) bool {
	return w.document != nil && name == w.document.DocumentName()
}

// expectFile reports whether the entry at src, which has a reserved name,
// is a file, and records a violation naming what the name holds when it is
// a folder.
func (w *folderWalker) expectFile(e os.DirEntry, src, holds string) bool {
	if !e.IsDir() {
		return true
	}
	w.violate("%s is a folder; the reserved name is for %s", w.rel(src), holds)
	return false
}

// expectFolder reports whether the entry at src, which has a reserved
// name, is a folder, and records a violation naming what the name holds
// when it is a file.
func (w *folderWalker) expectFolder(e os.DirEntry, src, holds string) bool {
	if e.IsDir() {
		return true
	}
	w.violate("%s is a file; the reserved name is for %s", w.rel(src), holds)
	return false
}

// readFlatRepresentation handles the simple case: no
// representations/ folder, so every non-reserved entry is the content of a
// single representation, named after the input folder itself.
func (w *folderWalker) readFlatRepresentation(entries []os.DirEntry) build.SourceRepresentation {
	name := filepath.Base(w.root)
	// The input folder's name becomes the representation's package-side
	// name, so it must satisfy the same rule as a folder under
	// representations/.
	if err := build.ValidateRepresentationName(name); err != nil {
		w.violate("the folder name names the single representation: %v", err)
	}
	rep := build.SourceRepresentation{Name: name}
	for _, e := range entries {
		src := filepath.Join(w.root, e.Name())
		if e.IsDir() {
			w.walkContent(w.root, src, &rep.Files)
			continue
		}
		rep.Files = append(rep.Files, w.newFile(w.root, src))
	}
	if len(rep.Files) == 0 {
		w.violate("the folder contains no content files")
	}
	return rep
}

// collectFiles gathers every file under dir recursively with Path relative
// to dir; documentation/, premis/, and representation content all collect
// the same way.
func (w *folderWalker) collectFiles(dir string) []build.SourceFile {
	var files []build.SourceFile
	w.walkContent(dir, dir, &files)
	return files
}

// collectPremisFiles collects a premis/ folder (package- or
// representation-level, same rule both places) and flags the one
// transport-level premis rule: premis.xml belongs to the generated
// document. Content conformance (a premis:premis document) is
// deliberately left to assembly.
func (w *folderWalker) collectPremisFiles(dir string) []build.SourceFile {
	files := w.collectFiles(dir)
	for _, f := range files {
		if path.Base(f.Path) == "premis.xml" {
			w.violate("%s: premis.xml is reserved for the generated preservation document; rename the received file", f.Key)
		}
	}
	return files
}

func (w *folderWalker) walkContent(base, dir string, files *[]build.SourceFile) {
	for _, e := range w.readDir(dir) {
		src := filepath.Join(dir, e.Name())
		if e.IsDir() {
			w.walkContent(base, src, files)
			continue
		}
		*files = append(*files, w.newFile(base, src))
	}
}

func (w *folderWalker) newFile(base, src string) build.SourceFile {
	relRoot, err := filepath.Rel(w.root, src)
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
func (w *folderWalker) decodeSidecar(src string) characterization.Report {
	f, err := os.Open(src)
	if err != nil {
		w.violate("siegfried.json: %v", err)
		return nil
	}
	defer f.Close()

	report, err := characterization.DecodeSiegfried(f)
	if err != nil {
		w.violate("siegfried.json: %v; regenerate it from the input root with: sf -hash md5 -json .", err)
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
func (w *folderWalker) readDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.violate("%s: %v", w.rel(dir), err)
		return nil
	}

	seen := make(map[string]string, len(entries))
	var kept []os.DirEntry
	for _, e := range entries {
		if isOSArtifact(e.Name()) {
			continue
		}
		if e.Type()&fs.ModeSymlink != 0 {
			w.violate("%s is a symbolic link; symbolic links are not allowed anywhere in an input folder", w.rel(filepath.Join(dir, e.Name())))
			continue
		}
		name := norm.NFC.String(e.Name())
		if prev, ok := seen[name]; ok {
			w.violate("%s: %q and %q are the same name after Unicode normalization; rename one", w.rel(dir), prev, e.Name())
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
func (w *folderWalker) rel(p string) string {
	rel, err := filepath.Rel(w.root, p)
	if err != nil {
		return p
	}
	if rel == "." {
		return "the input folder"
	}
	return filepath.ToSlash(rel)
}
