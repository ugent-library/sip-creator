package profiles

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// cannedMatch is the sidecar match writeSidecar stamps on every file.
var cannedMatch = []map[string]any{{
	"ns":     "pronom",
	"id":     "fmt/999",
	"format": "Test Format",
	"mime":   "image/test",
}}

// writeSidecar generates inDir's siegfried.json the way `sf -hash md5 -json .`
// run from the input root would: an entry per file on disk (real MD5s, the
// canned match) plus the self-entry a real run records for the sidecar file
// it is writing into — consumers must ignore entries they never look up.
// Tests that add input files afterwards must call it again.
func writeSidecar(t *testing.T, inDir string) {
	t.Helper()
	writeSidecarWith(t, inDir, cannedMatch)
}

func writeSidecarWith(t *testing.T, inDir string, matches []map[string]any) {
	t.Helper()
	var files []map[string]any
	err := filepath.Walk(inDir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Base(src) == "siegfried.json" {
			return nil
		}
		rel, err := filepath.Rel(inDir, src)
		if err != nil {
			return err
		}
		files = append(files, map[string]any{
			"filename": filepath.ToSlash(rel),
			"filesize": info.Size(),
			"errors":   "",
			"md5":      fileMD5(t, src),
			"matches":  matches,
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, map[string]any{
		"filename": "siegfried.json", "filesize": 0,
		"errors": "empty source", "md5": "d41d8cd98f00b204e9800998ecf8427e",
		"matches": []map[string]any{},
	})

	doc, err := json.Marshal(map[string]any{"siegfried": "1.11.0-test", "files": files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "siegfried.json"), doc, 0600); err != nil {
		t.Fatal(err)
	}
}

func fileMD5(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // test files are tiny
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

const testDescriptive = `{
	"dcterms:identifier": "local-id-001",
	"dcterms:title": {"@value": "Catus Testus", "@language": "nl"}
}`

// basicDef returns the registered "basic" definition the tests build with.
func basicDef(t *testing.T) Definition {
	t.Helper()
	def, ok := Get("basic")
	if !ok {
		t.Fatal(`no "basic" definition registered`)
	}
	return def
}

// newTestBuilder lays out a minimal input tree (one descriptive file, one
// representation with one essence file, a matching sidecar) and returns a
// builder plus its in/out dirs.
func newTestBuilder(t *testing.T) (b *Builder, inDir, outDir string) {
	t.Helper()
	inDir, outDir = t.TempDir(), t.TempDir()

	if err := os.WriteFile(filepath.Join(inDir, "dc+schema.json"), []byte(testDescriptive), 0600); err != nil {
		t.Fatal(err)
	}
	repDir := filepath.Join(inDir, "representation_1")
	if err := os.MkdirAll(repDir, 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repDir, "cat.jpg"), []byte("not really a jpeg"), 0600); err != nil {
		t.Fatal(err)
	}
	writeSidecar(t, inDir)

	b = New(&Config{
		Source:      inDir,
		Destination: outDir,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return b, inDir, outDir
}

func TestAssemble(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	// The package is rooted under the destination but nothing exists yet.
	if want := filepath.Join(outDir, pkg.Identifier); pkg.Location != want {
		t.Errorf("Location = %q, want %q", pkg.Location, want)
	}

	// Root entity wired, with the local identifier lifted onto it and the
	// entity identifier swapped into the description.
	e := pkg.Root
	if e == nil {
		t.Fatal("no root entity")
	}
	if got := e.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; got != "local-id-001" {
		t.Errorf("MEEMOO-LOCAL-ID = %q, want %q", got, "local-id-001")
	}
	d, ok := e.Description.(*metadata.Description)
	if !ok {
		t.Fatalf("Description is %T, want *metadata.Description", e.Description)
	}
	if d.Identifier != e.Identifier {
		t.Errorf("description identifier = %q, want entity identifier %q", d.Identifier, e.Identifier)
	}

	// Description file node: declared name and package-relative path.
	df := e.DescriptionFile
	if df == nil {
		t.Fatal("no description file node")
	}
	if df.Name != "dc+schema.xml" {
		t.Errorf("description file Name = %q, want %q", df.Name, "dc+schema.xml")
	}
	if df.Path != "metadata/descriptive/dc+schema.xml" {
		t.Errorf("description file Path = %q, want %q", df.Path, "metadata/descriptive/dc+schema.xml")
	}
	if df.Mime != "text/xml" {
		t.Errorf("description file Mime = %q, want %q", df.Mime, "text/xml")
	}

	// One schema node per bundled XSD, in sorted (deterministic) order.
	if len(pkg.SchemaFiles) != len(schemas.Get()) {
		t.Errorf("schema nodes = %d, want %d", len(pkg.SchemaFiles), len(schemas.Get()))
	}
	names := make([]string, 0, len(pkg.SchemaFiles))
	for _, sf := range pkg.SchemaFiles {
		names = append(names, sf.Name)
		if sf.Path != "schemas/"+sf.Name {
			t.Errorf("schema Path = %q, want %q", sf.Path, "schemas/"+sf.Name)
		}
		if sf.Mime != "application/xml" {
			t.Errorf("schema Mime = %q, want %q", sf.Mime, "application/xml")
		}
	}
	if !slices.IsSorted(names) {
		t.Errorf("schema nodes not sorted: %v", names)
	}

	// One representation, wired to the entity in both directions.
	if len(e.Representations) != 1 {
		t.Fatalf("representations = %d, want 1", len(e.Representations))
	}
	r := e.Representations[0]
	if r.Label != "representation_1" {
		t.Errorf("Label = %q, want %q", r.Label, "representation_1")
	}
	if r.Entity != e {
		t.Error("representation not wired back to the entity")
	}

	// The essence node records its source and a rep-relative path.
	if len(r.Files) != 1 {
		t.Fatalf("essence files = %d, want 1", len(r.Files))
	}
	f := r.Files[0]
	if want := filepath.Join(inDir, "representation_1", "cat.jpg"); f.Source != want {
		t.Errorf("Source = %q, want %q", f.Source, want)
	}
	if f.Path != "data/cat.jpg" {
		t.Errorf("Path = %q, want %q", f.Path, "data/cat.jpg")
	}
	if f.Representation != r {
		t.Error("essence file not wired back to the representation")
	}

	// The essence node is enriched with the sidecar's format and mime; the
	// report's extra entries (dc+schema.json, the sidecar itself) are ignored.
	if f.Format == nil || f.Format.FormatRegistry.Key != "fmt/999" {
		t.Errorf("essence Format = %+v, want fmt/999 from the sidecar", f.Format)
	}
	if f.Mime != "image/test" {
		t.Errorf("essence Mime = %q, want %q from the sidecar", f.Mime, "image/test")
	}

	// PREMIS and METS nodes are the writer's to create, not the assembler's.
	if pkg.PremisFile != nil || pkg.MetsFile != nil || r.PremisFile != nil || r.MetsFile != nil {
		t.Error("assemble created premis/mets nodes; those belong to write")
	}

	// The load-bearing guarantee: assembly writes nothing.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("assemble wrote to the destination: %v", entries)
	}
}

func TestAssembleRepresentations(t *testing.T) {
	b, inDir, _ := newTestBuilder(t) // already has representation_1/cat.jpg

	// More representations: multi-file, double-digit, and one with a nested
	// subdirectory; plus a non-matching directory that must be skipped.
	write := func(rel string) {
		t.Helper()
		path := filepath.Join(inDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0775); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("essence bytes"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("representation_2/a.jpg")
	write("representation_2/sub/deep.tif")
	write("representation_10/b.jpg")
	write("documentation/readme.txt")
	// The sidecar must describe the files as they are now — essence added
	// after the report was generated aborts the build by design.
	writeSidecar(t, inDir)

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	// filepath.Walk visits lexically, so this order is deterministic; the
	// "documentation" dir must not have produced a representation.
	var labels []string
	for _, r := range pkg.Root.Representations {
		labels = append(labels, r.Label)
	}
	want := []string{"representation_1", "representation_10", "representation_2"}
	if !slices.Equal(labels, want) {
		t.Fatalf("representation labels = %v, want %v", labels, want)
	}

	files := func(label string) map[string]*sip.File {
		t.Helper()
		for _, r := range pkg.Root.Representations {
			if r.Label == label {
				m := make(map[string]*sip.File, len(r.Files))
				for _, f := range r.Files {
					m[f.Name] = f
				}
				return m
			}
		}
		t.Fatalf("no representation %q", label)
		return nil
	}

	// Files in nested subdirectories are walked too, and flatten into
	// data/ — Path uses the base name, Source keeps the real location.
	rep2 := files("representation_2")
	if len(rep2) != 2 {
		t.Fatalf("representation_2 files = %d, want 2", len(rep2))
	}
	deep, ok := rep2["deep.tif"]
	if !ok {
		t.Fatal("nested file deep.tif not assembled")
	}
	if deep.Path != "data/deep.tif" {
		t.Errorf("nested file Path = %q, want %q", deep.Path, "data/deep.tif")
	}
	if want := filepath.Join(inDir, "representation_2", "sub", "deep.tif"); deep.Source != want {
		t.Errorf("nested file Source = %q, want %q", deep.Source, want)
	}

	if rep10 := files("representation_10"); len(rep10) != 1 {
		t.Errorf("representation_10 files = %d, want 1", len(rep10))
	}
}

// requireEmpty asserts the fail-fast property: a failed assemble leaves
// nothing in the destination.
func requireEmpty(t *testing.T, outDir string) {
	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("failed assemble wrote to the destination: %v", entries)
	}
}

// Characterization is optional in contract (ADR-0009): no sidecar means the
// build proceeds without format info.
func TestAssembleWithoutSidecar(t *testing.T) {
	b, inDir, _ := newTestBuilder(t)
	if err := os.Remove(filepath.Join(inDir, "siegfried.json")); err != nil {
		t.Fatal(err)
	}

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble without sidecar: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format != nil {
		t.Errorf("essence Format = %+v, want nil without a sidecar", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream without a sidecar", f.Mime)
	}
}

// An entry with empty matches[] is an honest no-match: Format stays nil for
// that file only, and assembly succeeds.
func TestAssembleSidecarNoMatch(t *testing.T) {
	b, inDir, _ := newTestBuilder(t)
	writeSidecarWith(t, inDir, []map[string]any{})

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble with no-match sidecar: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format != nil {
		t.Errorf("essence Format = %+v, want nil on no match", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream on no match", f.Mime)
	}
}

// A match that asserts no mime still yields the Format, and the mime falls
// back to the admitted unknown — the two facts are independent.
func TestAssembleSidecarMatchWithoutMime(t *testing.T) {
	b, inDir, _ := newTestBuilder(t)
	writeSidecarWith(t, inDir, []map[string]any{{"ns": "pronom", "id": "fmt/999"}})

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble with mimeless match: %v", err)
	}
	f := pkg.Root.Representations[0].Files[0]
	if f.Format == nil || f.Format.FormatRegistry.Key != "fmt/999" {
		t.Errorf("essence Format = %+v, want fmt/999", f.Format)
	}
	if f.Mime != "application/octet-stream" {
		t.Errorf("essence Mime = %q, want octet-stream when the match asserts none", f.Mime)
	}
}

// A present-but-broken sidecar aborts assembly: misconfiguration must be
// loud, and it fails before anything is written.
func TestAssembleSidecarMalformed(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	if err := os.WriteFile(filepath.Join(inDir, "siegfried.json"), []byte("{ not json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded despite a malformed sidecar")
	}
	requireEmpty(t, outDir)
}

// Essence the report doesn't know aborts: the file was added (or the report
// generated from the wrong directory) after the sf run.
func TestAssembleSidecarMissingEntry(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	// Added after writeSidecar — exactly the staleness the strictness catches.
	if err := os.WriteFile(filepath.Join(inDir, "representation_1", "extra.jpg"), []byte("late arrival"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded despite essence missing from the sidecar")
	}
	requireEmpty(t, outDir)
}

// Changed bytes fail the MD5 binding: a stale report must never lend its
// format claims to different content.
func TestAssembleSidecarChecksumMismatch(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	if err := os.WriteFile(filepath.Join(inDir, "representation_1", "cat.jpg"), []byte("different bytes now"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded despite essence changed since the report")
	}
	requireEmpty(t, outDir)
}

// A report without checksums (sf run without -hash md5) can't bind records
// to bytes, so it aborts rather than being trusted.
func TestAssembleSidecarChecksumless(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	raw := `{"siegfried":"1.11.0","files":[{"filename":"representation_1/cat.jpg","filesize":17,"errors":"","matches":[]}]}`
	if err := os.WriteFile(filepath.Join(inDir, "siegfried.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded despite a checksumless sidecar entry")
	}
	requireEmpty(t, outDir)
}

// A per-file error recorded by sf aborts: the tool is telling us it never
// characterized these bytes.
func TestAssembleSidecarEntryError(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	raw := `{"siegfried":"1.11.0","files":[{"filename":"representation_1/cat.jpg","filesize":17,"errors":"permission denied","md5":"ab","matches":[]}]}`
	if err := os.WriteFile(filepath.Join(inDir, "siegfried.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded despite an sf-reported file error")
	}
	requireEmpty(t, outDir)
}

// A caller-supplied report is the library transport (ADR-0009): it wins over
// file discovery — the sidecar on disk is never even read.
func TestAssembleCallerReport(t *testing.T) {
	b, inDir, _ := newTestBuilder(t)
	// Poison the on-disk sidecar: reading it would abort the build.
	if err := os.WriteFile(filepath.Join(inDir, "siegfried.json"), []byte("{ not json"), 0600); err != nil {
		t.Fatal(err)
	}

	fr := sip.NewFormatRegistry()
	fr.Name = "pronom"
	fr.Key = "fmt/999"
	b.Characterization = characterization.Report{
		"representation_1/cat.jpg": {
			Format: &sip.Format{FormatRegistry: fr},
			Mime:   "image/test",
			MD5:    fileMD5(t, filepath.Join(inDir, "representation_1", "cat.jpg")),
		},
	}

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble with caller report: %v", err)
	}
	if f := pkg.Root.Representations[0].Files[0]; f.Format == nil || f.Format.FormatRegistry.Key != "fmt/999" {
		t.Errorf("essence Format = %+v, want fmt/999 from the caller report", f.Format)
	}
}

// Documentation is optional: absent means no nodes; present means nodes
// with structure preserved in Path and — as ever — zero writes.
func TestAssembleDocumentation(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)

	pkg, err := b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if len(pkg.DocumentationFiles) != 0 {
		t.Errorf("documentation nodes = %d without a documentation dir, want 0", len(pkg.DocumentationFiles))
	}

	if err := os.MkdirAll(filepath.Join(inDir, "documentation", "sub"), 0775); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"documentation/manual.txt", "documentation/sub/notes.txt"} {
		if err := os.WriteFile(filepath.Join(inDir, filepath.FromSlash(rel)), []byte("doc"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	pkg, err = b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble with documentation: %v", err)
	}

	paths := make([]string, 0, len(pkg.DocumentationFiles))
	for _, f := range pkg.DocumentationFiles {
		paths = append(paths, f.Path)
		if f.Source == "" {
			t.Errorf("documentation node %s has no Source", f.Path)
		}
		// The sidecar predates these files: no entry, so the mime is the
		// admitted unknown (documentation is lenient, no abort).
		if f.Mime != "application/octet-stream" {
			t.Errorf("documentation Mime = %q, want octet-stream without an entry", f.Mime)
		}
	}
	slices.Sort(paths)
	want := []string{"documentation/manual.txt", "documentation/sub/notes.txt"}
	if !slices.Equal(paths, want) {
		t.Errorf("documentation paths = %v, want %v", paths, want)
	}

	// With a regenerated sidecar the documentation entries exist, and their
	// mime is taken from the report.
	writeSidecar(t, inDir)
	pkg, err = b.assemble(basicDef(t))
	if err != nil {
		t.Fatalf("assemble with regenerated sidecar: %v", err)
	}
	for _, f := range pkg.DocumentationFiles {
		if f.Mime != "image/test" {
			t.Errorf("documentation Mime = %q, want %q from the sidecar", f.Mime, "image/test")
		}
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("assemble wrote to the destination: %v", entries)
	}
}

func TestAssembleMissingDescriptive(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)
	if err := os.Remove(filepath.Join(inDir, "dc+schema.json")); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded without a descriptive source")
	}

	// Fail-fast: a bad input leaves no partial package behind.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("failed assemble wrote to the destination: %v", entries)
	}
}

func TestAssembleDescriptiveIsDirectory(t *testing.T) {
	b, inDir, _ := newTestBuilder(t)
	if err := os.Remove(filepath.Join(inDir, "dc+schema.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inDir, "dc+schema.json"), 0775); err != nil {
		t.Fatal(err)
	}

	if _, err := b.assemble(basicDef(t)); err == nil {
		t.Fatal("assemble succeeded with a directory as descriptive source")
	}
}
