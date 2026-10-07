package input

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"golang.org/x/text/unicode/norm"
)

// folderReader holds what one read of an input folder shares across the
// walk and the decoders; Read makes one per call.
type folderReader struct {
	root       string     // all messages and report keys are relative to it
	violations Violations // the findings so far
	// documentName is the file name reserved for the profile's supplied
	// descriptive document at both levels; empty under a profile that
	// takes rows only.
	documentName string
	// refusedDocumentName is the profile's document name when its model
	// takes no supplied document, such as dc+schema.xml under meemoo/basic: a
	// file with that name at either level is a violation, never content.
	// Empty under a profile that takes a document.
	refusedDocumentName string
}

// inventory lists the files of an input folder that a decoder reads after
// the walk. An empty path means the folder has no such file. reps has an
// entry per folder under representations/; a flat folder has none, because
// its description.csv describes the package.
type inventory struct {
	pkg                descriptionFiles            // the package level
	reps               map[string]descriptionFiles // by representation name
	representationsCSV string
	sidecar            string
}

// descriptionFiles are the files one level may describe itself with.
type descriptionFiles struct {
	rows     string // description.csv
	document string // the profile's supplied document
}

// Reserved top-level names. Reserved names inside a representation are
// a subset. The profile's document name, when it has one, is reserved at
// both levels too (isDocumentName). Every reserved name is ASCII, which
// NFC normalization never alters, so comparing an unnormalized directory
// entry name to one is exact.
const (
	descriptionName        = "description.csv"
	representationsName    = "representations"
	representationsCSVName = "representations.csv"
	documentationName      = "documentation"
	premisName             = "premis"
	sidecarName            = "siegfried.json"
)

// walk reads the structure of the folder: the source package as far as
// names, kinds and places fill it, and the inventory of the files the
// decoders read. The reserved names each go to their collector or into
// the inventory; everything else is content, whose place depends on
// whether a representations/ folder exists.
func (r *folderReader) walk() (*build.SourcePackage, inventory) {
	source := &build.SourcePackage{}
	var inv inventory

	var content []os.DirEntry
	var found descriptionFiles
	var representationsPath, representationsCSVPath string

	for _, e := range r.readDir(r.root) {
		name := e.Name()
		src := filepath.Join(r.root, e.Name())
		if r.isDocumentName(name) {
			if r.expectFile(e, src, "the supplied descriptive document") {
				found.document = src
			}
			continue
		}
		if r.refuseDocument(name, src) {
			continue
		}
		switch name {
		case descriptionName:
			if r.expectFile(e, src, "the descriptive rows file") {
				found.rows = src
			}
		case representationsName:
			if r.expectFolder(e, src, "the folder of representations") {
				representationsPath = src
			}
		case representationsCSVName:
			if r.expectFile(e, src, "the representations file") {
				representationsCSVPath = src
			}
		case documentationName:
			if r.expectFolder(e, src, "a folder") {
				source.Documentation = r.collectFiles(src)
			}
		case premisName:
			if r.expectFolder(e, src, "a folder") {
				source.Premis = r.collectPremisFiles(src)
			}
		case sidecarName:
			if r.expectFile(e, src, "the characterization report") {
				inv.sidecar = src
			}
		default:
			content = append(content, e)
		}
	}

	inv.pkg = r.levelDescription(found, true)

	if representationsPath != "" {
		// With a representations/ folder, all content lives inside it;
		// only the reserved names may sit beside it.
		for _, e := range content {
			r.violate("%s: content must live inside representations/ when that folder exists (only the reserved names of the input specification may sit beside it)", e.Name())
		}
		source.Representations, inv.reps = r.readRepresentations(representationsPath)
		inv.representationsCSV = representationsCSVPath
	} else {
		if representationsCSVPath != "" {
			r.violate("representations.csv requires a representations/ folder; a flat folder is one representation named after the folder itself")
		}
		source.Representations = []build.SourceRepresentation{r.readFlatRepresentation(content)}
	}

	return source, inv
}

func (r *folderReader) readRepresentations(dir string) ([]build.SourceRepresentation, map[string]descriptionFiles) {
	var reps []build.SourceRepresentation
	descriptions := map[string]descriptionFiles{}
	for _, e := range r.readDir(dir) {
		if !e.IsDir() {
			r.violate("representations/%s: only representation folders may sit directly inside representations/", e.Name())
			continue
		}
		name := e.Name()
		// The folder-name rule is the library's name rule:
		// one source of truth for what a representation may be called.
		if err := build.ValidateRepresentationName(name); err != nil {
			// Still read the folder, so the problems inside it are
			// reported in the same run.
			r.violate("representations/%s: %v", e.Name(), err)
		}
		rep, files := r.readRepresentation(filepath.Join(dir, e.Name()), name)
		reps = append(reps, rep)
		descriptions[name] = files
	}
	if len(reps) == 0 {
		r.violate("representations/ contains no representation folders: a package needs at least one version of the content")
	}
	return reps, descriptions
}

func (r *folderReader) readRepresentation(dir, repName string) (build.SourceRepresentation, descriptionFiles) {
	rep := build.SourceRepresentation{Name: repName}
	var found descriptionFiles
	for _, e := range r.readDir(dir) {
		name := e.Name()
		src := filepath.Join(dir, e.Name())
		if r.isDocumentName(name) {
			if r.expectFile(e, src, "the supplied descriptive document") {
				found.document = src
			}
			continue
		}
		if r.refuseDocument(name, src) {
			continue
		}
		switch name {
		case descriptionName:
			if r.expectFile(e, src, "the descriptive rows file") {
				found.rows = src
			}
		case documentationName:
			if r.expectFolder(e, src, "a folder") {
				rep.Documentation = r.collectFiles(src)
			}
		case premisName:
			if r.expectFolder(e, src, "a folder") {
				rep.Premis = r.collectPremisFiles(src)
			}
		default:
			if e.IsDir() {
				r.walkContent(dir, src, &rep.Files)
				continue
			}
			rep.Files = append(rep.Files, r.newFile(dir, src))
		}
	}
	files := r.levelDescription(found, false)
	if len(rep.Files) == 0 {
		r.violate("%s: the representation contains no content files", r.rel(dir))
	}
	return rep, files
}

// levelDescription applies the rules on which description files one level
// has, and returns the one the decoder reads, if any. Both at one level is
// a violation: an entity has one description, and the tool does not pick.
// The package level needs one; a representation may have neither.
func (r *folderReader) levelDescription(found descriptionFiles, packageLevel bool) descriptionFiles {
	switch {
	case found.rows != "" && found.document != "":
		described := "representation"
		if packageLevel {
			described = "package"
		}
		r.violate("%s and %s are both present; describe the %s with one of the two, not both (input specification §3)", r.rel(found.rows), r.rel(found.document), described)
		return descriptionFiles{}
	case found.rows == "" && found.document == "" && packageLevel:
		r.violateMissingDescription()
	}
	return found
}

// violateMissingDescription records that the package level describes
// nothing, naming the file or files the profile accepts. It repeats the
// library's rule (SourcePackage.Validate, ADR-0025), so that check, which
// never builds, reports it too.
func (r *folderReader) violateMissingDescription() {
	if r.documentName == "" {
		r.violate("descriptive rows are missing: every package folder needs a description.csv describing the content (input specification §3)")
		return
	}
	r.violate("descriptive metadata is missing: every package folder needs a description.csv or a %s describing the content (input specification §3)", r.documentName)
}

// isDocumentName reports whether name is the file name reserved for the
// profile's supplied descriptive document. Under a profile that takes
// none, no name is: a dc.xml under meemoo/basic is content like any other file.
func (r *folderReader) isDocumentName(name string) bool {
	return r.documentName != "" && name == r.documentName
}

// refuseDocument reports whether name is the document name of a profile
// that takes no supplied document, and records a violation at src when it
// is. Under basic a dc+schema.xml would otherwise pass as content: the
// producer meant it as the description, and the package would carry it as
// essence.
func (r *folderReader) refuseDocument(name, src string) bool {
	if r.refusedDocumentName == "" || name != r.refusedDocumentName {
		return false
	}
	r.violate("%s: the profile takes no supplied descriptive document; describe the package in description.csv", r.rel(src))
	return true
}

// expectFile reports whether the entry at src, which has a reserved name,
// is a file, and records a violation naming what the name holds when it is
// a folder.
func (r *folderReader) expectFile(e os.DirEntry, src, holds string) bool {
	if !e.IsDir() {
		return true
	}
	r.violate("%s is a folder; the reserved name is for %s", r.rel(src), holds)
	return false
}

// expectFolder reports whether the entry at src, which has a reserved
// name, is a folder, and records a violation naming what the name holds
// when it is a file.
func (r *folderReader) expectFolder(e os.DirEntry, src, holds string) bool {
	if e.IsDir() {
		return true
	}
	r.violate("%s is a file; the reserved name is for %s", r.rel(src), holds)
	return false
}

// readFlatRepresentation handles the simple case: no
// representations/ folder, so every non-reserved entry is the content of a
// single representation, named after the input folder itself.
func (r *folderReader) readFlatRepresentation(entries []os.DirEntry) build.SourceRepresentation {
	name := filepath.Base(r.root)
	// The input folder's name becomes the representation's package-side
	// name, so it must satisfy the same rule as a folder under
	// representations/.
	if err := build.ValidateRepresentationName(name); err != nil {
		r.violate("the folder name names the single representation: %v", err)
	}
	rep := build.SourceRepresentation{Name: name}
	for _, e := range entries {
		src := filepath.Join(r.root, e.Name())
		if e.IsDir() {
			r.walkContent(r.root, src, &rep.Files)
			continue
		}
		rep.Files = append(rep.Files, r.newFile(r.root, src))
	}
	if len(rep.Files) == 0 {
		r.violate("the folder contains no content files")
	}
	return rep
}

// collectFiles gathers every file under dir recursively with Path relative
// to dir; documentation/, premis/, and representation content all collect
// the same way.
func (r *folderReader) collectFiles(dir string) []build.SourceFile {
	var files []build.SourceFile
	r.walkContent(dir, dir, &files)
	return files
}

// collectPremisFiles collects a premis/ folder (package- or
// representation-level, same rule both places) and applies the input
// rules for received preservation files: premis.xml belongs to the
// generated document, and every file must be well-formed XML. Whether the
// root is a premis:premis element is left to assembly, which checks it for
// every source package (build.assembleReceivedPremis).
func (r *folderReader) collectPremisFiles(dir string) []build.SourceFile {
	files := r.collectFiles(dir)
	for _, f := range files {
		if path.Base(f.Path) == "premis.xml" {
			r.violate("%s: premis.xml is reserved for the generated preservation document; rename the received file", f.Key)
		}
		r.checkWellFormed(f)
	}
	return files
}

// checkWellFormed reports f when it is not well-formed XML.
func (r *folderReader) checkWellFormed(f build.SourceFile) {
	file, err := os.Open(f.Source)
	if err != nil {
		r.violate("%s: %v", f.Key, err)
		return
	}
	defer file.Close()

	if _, err := xmldoc.Root(file); err != nil {
		r.violate("%s: %v", f.Key, err)
	}
}

func (r *folderReader) walkContent(base, dir string, files *[]build.SourceFile) {
	for _, e := range r.readDir(dir) {
		src := filepath.Join(dir, e.Name())
		if e.IsDir() {
			r.walkContent(base, src, files)
			continue
		}
		*files = append(*files, r.newFile(base, src))
	}
}

func (r *folderReader) newFile(base, src string) build.SourceFile {
	// Rel cannot fail here: root, base and src are absolute, and src lies
	// under both.
	relRoot, _ := filepath.Rel(r.root, src)
	relBase, _ := filepath.Rel(base, src)
	f := build.SourceFile{
		Source: src,
		// Key is not NFC-normalized: it must match the filename exactly
		// as the characterization report recorded it.
		Key:  path.Clean(filepath.ToSlash(relRoot)),
		Path: norm.NFC.String(filepath.ToSlash(relBase)),
	}
	// The path is written into the METS and PREMIS documents, so it is held
	// to the library's rule for their text (SourcePackage.Validate). Key
	// holds the same names from the input root, so the finding names the
	// file the producer must rename.
	if err := build.ValidateXMLText(f.Key); err != nil {
		r.violate("%v; rename the file or folder", err)
	}
	return f
}

// readDir lists dir under the rules that hold everywhere in the input
// folder: a symbolic link is a violation and is never followed, OS
// artifacts are skipped without a word, and two names that are the same
// after NFC normalization are a collision, because they can coexist on a
// filesystem that does not normalize but would collide in the package.
// os.ReadDir sorts by name, so the order of every file list is the same
// from run to run; neither CSIP nor Meemoo gives that order a meaning.
func (r *folderReader) readDir(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		r.violate("%s: %v", r.rel(dir), err)
		return nil
	}

	seen := make(map[string]string, len(entries))
	var kept []os.DirEntry
	for _, e := range entries {
		if isOSArtifact(e.Name()) {
			continue
		}
		if e.Type()&fs.ModeSymlink != 0 {
			r.violate("%s is a symbolic link; symbolic links are not allowed anywhere in an input folder", r.rel(filepath.Join(dir, e.Name())))
			continue
		}
		name := norm.NFC.String(e.Name())
		if prev, ok := seen[name]; ok {
			r.violate("%s: %q and %q are the same name after Unicode normalization; rename one", r.rel(dir), prev, e.Name())
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
func (r *folderReader) rel(p string) string {
	rel, err := filepath.Rel(r.root, p)
	if err != nil {
		return p
	}
	if rel == "." {
		return "the input folder"
	}
	return filepath.ToSlash(rel)
}
