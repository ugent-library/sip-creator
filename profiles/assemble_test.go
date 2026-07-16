package profiles

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
)

// fakeIdentificator stands in for siegfried: it mints a File node the way
// formats.Identificator implementations do, without shelling out.
type fakeIdentificator struct{}

func (fakeIdentificator) Process(src string) *sip.File {
	f := sip.NewFile()
	f.Name = filepath.Base(src)
	f.Checksum = "cafebabe00000000000000000000cafe"
	return f
}

const testDescriptive = `{
	"dcterms:identifier": "local-id-001",
	"dcterms:title": {"@value": "Catus Testus", "@language": "nl"}
}`

// newTestProfile lays out a minimal input tree (one descriptive file, one
// representation with one essence file) and returns a profile plus its
// in/out dirs.
func newTestProfile(t *testing.T) (p *Profile, inDir, outDir string) {
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

	p = New(&Config{
		Source:      inDir,
		Destination: outDir,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Formats:     fakeIdentificator{},
	})
	return p, inDir, outDir
}

func TestAssemble(t *testing.T) {
	p, inDir, outDir := newTestProfile(t)

	pkg, err := p.assemble()
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
	p, inDir, _ := newTestProfile(t) // already has representation_1/cat.jpg

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

	pkg, err := p.assemble()
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

func TestAssembleMissingDescriptive(t *testing.T) {
	p, inDir, outDir := newTestProfile(t)
	if err := os.Remove(filepath.Join(inDir, "dc+schema.json")); err != nil {
		t.Fatal(err)
	}

	if _, err := p.assemble(); err == nil {
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
	p, inDir, _ := newTestProfile(t)
	if err := os.Remove(filepath.Join(inDir, "dc+schema.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(inDir, "dc+schema.json"), 0775); err != nil {
		t.Fatal(err)
	}

	if _, err := p.assemble(); err == nil {
		t.Fatal("assemble succeeded with a directory as descriptive source")
	}
}
