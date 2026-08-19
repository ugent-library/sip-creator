package input

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// repNameRx is the representation folder-name rule (input spec §2).
var repNameRx = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (r *reader) readRepresentations(dir string) []Representation {
	var reps []Representation
	for _, e := range r.readDir(dir) {
		if !e.IsDir() {
			r.violate("representations/%s: only representation folders may sit directly inside representations/", e.Name())
			continue
		}
		label := e.Name()
		if !repNameRx.MatchString(label) {
			// Still read the folder: the naming fix shouldn't hide any
			// findings inside it (collect-all).
			r.violate("representations/%s: a representation name may only contain letters, digits, and . _ -", e.Name())
		}
		reps = append(reps, r.readRepresentation(filepath.Join(dir, e.Name()), label))
	}
	if len(reps) == 0 {
		r.violate("representations/ contains no representation folders — a package needs at least one version of the content")
	}
	return reps
}

func (r *reader) readRepresentation(dir, label string) Representation {
	rep := Representation{Label: label}
	for _, e := range r.readDir(dir) {
		// The reserved names are ASCII, so byte comparison is exact — NFC
		// normalization is needed only where non-ASCII can occur (see
		// readDir and newFile).
		name := e.Name()
		src := filepath.Join(dir, e.Name())
		switch {
		case name == metadataName:
			if e.IsDir() {
				r.violate("%s is a folder — the reserved name is for the metadata file", r.rel(src))
				continue
			}
			rep.Descriptive = r.decodeMetadataCSV(src, false)
		case name == documentationName:
			if !e.IsDir() {
				r.violate("%s is a file — the reserved name is for a folder", r.rel(src))
				continue
			}
			rep.Documentation = r.collectFiles(src)
		case name == premisName:
			if !e.IsDir() {
				r.violate("%s is a file — the reserved name is for a folder", r.rel(src))
				continue
			}
			rep.Premis = r.collectFiles(src)
		case e.IsDir():
			r.walkContent(dir, src, &rep.Files)
		default:
			rep.Files = append(rep.Files, r.newFile(dir, src))
		}
	}
	if len(rep.Files) == 0 {
		r.violate("%s: the representation contains no content files", r.rel(dir))
	}
	return rep
}

// readFlatRepresentation handles the simple case (input spec §2): no
// representations/ folder, so every non-reserved entry is the content of a
// single representation, labeled after the input folder itself.
func (r *reader) readFlatRepresentation(entries []os.DirEntry) Representation {
	rep := Representation{Label: filepath.Base(r.root)}
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
// to dir — documentation/, premis/, and representation content all collect
// the same way.
func (r *reader) collectFiles(dir string) []File {
	var files []File
	r.walkContent(dir, dir, &files)
	return files
}

func (r *reader) walkContent(base, dir string, files *[]File) {
	for _, e := range r.readDir(dir) {
		src := filepath.Join(dir, e.Name())
		if e.IsDir() {
			r.walkContent(base, src, files)
			continue
		}
		*files = append(*files, r.newFile(base, src))
	}
}

func (r *reader) newFile(base, src string) File {
	relRoot, err := filepath.Rel(r.root, src)
	if err != nil {
		relRoot = src
	}
	relBase, err := filepath.Rel(base, src)
	if err != nil {
		relBase = filepath.Base(src)
	}
	return File{
		Source: src,
		// Rel stays byte-exact (no NFC): it must match the filename the
		// characterization report recorded from this same filesystem.
		Rel:  path.Clean(filepath.ToSlash(relRoot)),
		Path: norm.NFC.String(filepath.ToSlash(relBase)),
	}
}

// readDir lists dir applying the rules that hold everywhere in the input
// tree (input spec §1): symbolic links are a violation and are never
// followed; OS artifacts are silently ignored — never packaged, never
// warned about; and two names identical after NFC normalization are a
// collision — such pairs can coexist on non-normalizing filesystems and
// would collide in the package.
//
// os.ReadDir lists lexically, so every file list built over it is in
// deterministic traversal order. That order carries no meaning — neither
// CSIP nor meemoo assigns semantics to file order (explicit sequencing is
// the deferred manifest feature, input spec §8) — but it must be stable:
// METS emission and the baseline gate depend on run-to-run identical order.
func (r *reader) readDir(dir string) []os.DirEntry {
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
			r.violate("%s is a symbolic link — symbolic links are not allowed anywhere in an input folder", r.rel(filepath.Join(dir, e.Name())))
			continue
		}
		name := norm.NFC.String(e.Name())
		if prev, ok := seen[name]; ok {
			r.violate("%s: %q and %q are the same name after Unicode normalization — rename one", r.rel(dir), prev, e.Name())
			continue
		}
		seen[name] = e.Name()
		kept = append(kept, e)
	}
	return kept
}

// isOSArtifact reports whether name is operating-system droppings the spec
// says to ignore silently (§1).
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
func (r *reader) rel(p string) string {
	rel, err := filepath.Rel(r.root, p)
	if err != nil {
		return p
	}
	if rel == "." {
		return "the input folder"
	}
	return filepath.ToSlash(rel)
}
