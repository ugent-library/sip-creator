package build_test

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

func fileMD5(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) // test files are tiny
	if err != nil {
		t.Fatal(err)
	}
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

// testDescription satisfies the strictest registered profile: Meemoo's
// four required elements, Dutch entries on the lang-tagged ones. It is the
// meemoo/basic profile's input; ugent tests swap in identityTerms.
func testDescription() meemoo.Terms {
	return meemoo.Terms{
		{Key: "dcterms:identifier", Value: "local-id-001"},
		{Key: "dcterms:title", Lang: "nl", Value: "Catus Testus"},
		{Key: "dcterms:description", Lang: "nl", Value: "Een testkat"},
		{Key: "dcterms:created", Value: "2026"},
	}
}

// meemooIdentityTerms is the same identity in Meemoo's standard: short of
// the four keys the basic profile requires.
func meemooIdentityTerms() meemoo.Terms {
	return meemoo.Terms{
		{Key: "dcterms:identifier", Value: "local-id-001"},
		{Key: "dcterms:title", Lang: "nl", Value: "Catus Testus"},
	}
}

// identityTerms is the input convention's own MUSTs and nothing more, in
// the ugent/basic profile's standard, Simple Dublin Core.
func identityTerms() simpledc.Terms {
	return simpledc.Terms{
		{Key: "identifier", Value: "local-id-001"},
		{Key: "title", Value: "Catus Testus"},
	}
}

// identityRecord is the same identity in the ugent/bibliographic profile's standard,
// a MODS record without items.
func identityRecord() mods.Record {
	return mods.Record{
		Identifier: "local-id-001",
		Titles:     []mods.Title{{Value: "Catus Testus"}},
	}
}

// writeEssence puts one content file on disk and returns its build.SourceFile.
func writeEssence(t *testing.T, dir, name, content string) build.SourceFile {
	t.Helper()
	src := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(src), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return build.SourceFile{Source: src, Key: name, Path: name}
}

// testFormat returns the canned format assertion the report-based tests use.
func testFormat() *sip.Format {
	fr := sip.NewFormatRegistry()
	fr.Name = "pronom"
	fr.Key = "fmt/999"
	return &sip.Format{FormatRegistry: fr}
}

// report builds a characterization report with an entry (matching checksum,
// canned format) for every given source file.
func report(t *testing.T, files ...build.SourceFile) characterization.Report {
	t.Helper()
	rep := make(characterization.Report, len(files))
	for _, f := range files {
		rep[f.Key] = characterization.Record{
			Format: testFormat(),
			Mime:   "image/test",
			MD5:    fileMD5(t, f.Source),
		}
	}
	return rep
}

// newTestBuilder returns a builder for def over the minimal valid input
// data: one representation with one essence file, descriptive terms, no
// report.
func newTestBuilder(t *testing.T, def build.Definition) (b *build.Builder, in *build.SourcePackage, outDir string) {
	t.Helper()
	inDir, outDir := t.TempDir(), t.TempDir()
	cat := writeEssence(t, inDir, "cat.jpg", "not really a jpeg")

	in = &build.SourcePackage{
		Description: testDescription(),
		Representations: []build.SourceRepresentation{
			{Name: "archival", Files: []build.SourceFile{cat}},
		},
	}
	b, err := build.New(&build.Config{
		Profile:     def,
		Destination: outDir,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return b, in, outDir
}

// basicDef returns the registered "meemoo/basic" definition the tests build with.
func basicDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("meemoo/basic")
	if !ok {
		t.Fatal(`no "meemoo/basic" definition registered`)
	}
	return def
}

// ugentBasicDef returns the registered "ugent/basic" definition the tests build with.
func ugentBasicDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("ugent/basic")
	if !ok {
		t.Fatal(`no "ugent/basic" definition registered`)
	}
	return def
}

// ugentBasicWithoutVocabulary returns ugent/basic without its
// representation types: a profile whose representation names and types are
// free, for the tests of the label and type cascade (ADR-0014).
func ugentBasicWithoutVocabulary(t *testing.T) build.Definition {
	t.Helper()
	def := ugentBasicDef(t)
	def.RepresentationTypes = nil
	return def
}

// bibliographicDef returns the registered "ugent/bibliographic" definition the tests
// build with.
func bibliographicDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("ugent/bibliographic")
	if !ok {
		t.Fatal(`no "ugent/bibliographic" definition registered`)
	}
	return def
}

func requireEmpty(t *testing.T, outDir string) {
	t.Helper()
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the destination is not empty: %v", entries)
	}
}

// A definition built outside the registry names no metadata model and is
// refused when the builder is constructed, as an error rather than
// a panic, so no build can start from it.
func TestNewRefusesDefinitionWithoutModel(t *testing.T) {
	def := basicDef(t)
	def.Model = nil
	_, err := build.New(&build.Config{
		Profile:     def,
		Destination: t.TempDir(),
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if want := "names no metadata model"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("New error = %v, want %q", err, want)
	}
}

// namedFormat is a profile's metadata model with its format name replaced.
type namedFormat struct {
	build.MetadataModel
	name string
}

func (m namedFormat) ModelType() string { return m.name }

// A model whose format the METS MDTYPE vocabulary does not list gets
// MDTYPE OTHER, with its format's name in OTHERMDTYPE, on the METS dmdSec
// that points at its document.
func TestBuildWritesOtherMDType(t *testing.T) {
	def := basicDef(t)
	def.Model = namedFormat{def.Model, "catalogue-record"}
	b, in, _ := newTestBuilder(t, def)

	pkg, err := b.Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	mets, err := os.ReadFile(filepath.Join(pkg.Location, "METS.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := `MDTYPE="OTHER" OTHERMDTYPE="catalogue-record"`; !strings.Contains(string(mets), want) {
		t.Errorf("the package METS lacks %s", want)
	}
}

// A format name METS cannot record is refused when the builder is
// constructed: none at all, since METS requires MDTYPE, or text XML cannot
// carry.
func TestNewRefusesAFormatNameMETSCannotRecord(t *testing.T) {
	for name, tc := range map[string]struct {
		format, want string
	}{
		"no name":          {"", "names no format"},
		"not text for XML": {"catalogue\x01record", "XML cannot carry"},
	} {
		t.Run(name, func(t *testing.T) {
			def := basicDef(t)
			def.Model = namedFormat{def.Model, tc.format}
			_, err := build.New(&build.Config{Profile: def, Destination: t.TempDir()})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("New error = %v, want %q", err, tc.want)
			}
		})
	}
}

// A config without a logger builds a package; the progress messages are
// discarded.
func TestBuildWithoutLogger(t *testing.T) {
	_, in, _ := newTestBuilder(t, basicDef(t))
	b, err := build.New(&build.Config{Profile: basicDef(t), Destination: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Build(in); err != nil {
		t.Fatalf("Build error = %v, want nil", err)
	}
}

// Build refuses invalid input data before any side effect: a SourcePackage
// built directly in Go that breaks a rule leaves nothing on disk.
func TestBuildInvalidSourceWritesNothing(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.Representations = nil

	_, err := b.Build(in)
	if want := "no representations supplied"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Build error = %v, want %q", err, want)
	}
	requireEmpty(t, outDir)
}

// A package directory that cannot be created ends the build with the
// file system's error, which a caller can test for with errors.Is.
func TestBuildReportsAnUnwritableDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not make a folder read-only through its mode bits")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}
	b, in, outDir := newTestBuilder(t, basicDef(t))
	if err := os.Chmod(outDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(outDir, 0o700) })

	_, err := b.Build(in)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Build error = %v, want a permission error", err)
	}
}

// An update reuses the earlier package's identifier, and so its directory
// name. Build refuses a package directory that already exists instead of
// writing into it, where the earlier package's files would stay beside the
// new ones, and leaves that directory as it was.
func TestBuildRefusesAnExistingPackageDirectory(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.PackageIdentifier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"
	in.RecordStatus = sip.RecordStatusReplacement
	earlier := filepath.Join(outDir, in.PackageIdentifier)
	if err := os.MkdirAll(earlier, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(earlier, "METS.xml"), []byte("the earlier package"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := b.Build(in)
	if want := "package directory " + earlier + " already exists"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Build error = %v, want %q", err, want)
	}
	if got, _ := os.ReadFile(filepath.Join(earlier, "METS.xml")); string(got) != "the earlier package" {
		t.Errorf("the earlier package's METS was changed: %q", got)
	}
	requireDirectoryHolds(t, outDir, in.PackageIdentifier)
}

// A build that fails while writing leaves nothing behind, whatever the
// cause: here an essence file that is gone by the time it is copied, which
// assembly does not notice without a characterization report.
func TestBuildLeavesNothingWhenWritingFails(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	if err := os.Remove(in.Representations[0].Files[0].Source); err != nil {
		t.Fatal(err)
	}

	_, err := b.Build(in)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Build error = %v, want the missing file reported", err)
	}
	requireEmpty(t, outDir)
}

// A temporary directory left by a build that was killed partway is
// removed, not written into, so none of its files reach the package; a
// successful build leaves only the package directory.
func TestBuildRemovesAStaleTemporaryDirectory(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	in.PackageIdentifier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"
	stale := filepath.Join(outDir, "."+in.PackageIdentifier+".tmp", "representations", "archival", "data")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "left-over.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, err := b.Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkg.Location, "representations", "archival", "data", "left-over.jpg")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file from the stale temporary directory reached the package (%v)", err)
	}
	requireDirectoryHolds(t, outDir, in.PackageIdentifier)
}

// requireDirectoryHolds fails the test unless dir holds exactly the named
// entries.
func requireDirectoryHolds(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s holds %v, want %v", dir, got, want)
	}
}

// Build enforces what each standard requires of a package-level
// description. Identity-only terms build a complete ugent/basic package; under
// basic they are refused, and a missing identity is refused under every
// profile, all before any side effect.
func TestBuildRequiredPerStandard(t *testing.T) {
	b, in, _ := newTestBuilder(t, ugentBasicDef(t))
	in.Description = identityTerms()
	if _, err := b.Build(in); err != nil {
		t.Fatalf("ugent/basic Build refused identity-only terms: %v", err)
	}

	cases := []struct {
		name string
		def  build.Definition
		desc sip.Description
		want string
	}{
		{"basic without description and created", basicDef(t), meemooIdentityTerms(), "description is required"},
		{"basic without a title", basicDef(t), meemoo.Terms{{Key: "dcterms:identifier", Value: "x"}}, "title is required"},
		{"ugent/basic without an identifier", ugentBasicDef(t), simpledc.Terms{{Key: "title", Value: "x"}}, "identifier is required"},
		{"ugent/bibliographic without a title", bibliographicDef(t), mods.Record{Identifier: "x"}, "title is required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, c.def)
			in.Description = c.desc
			_, err := b.Build(in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Build error = %v, want %q", err, c.want)
			}
			requireEmpty(t, outDir)
		})
	}
}

// The ugent/bibliographic profile builds a complete package from a record: mods.xml
// under metadata/descriptive with the items rendered, the METS set plus the
// MODS schema and nothing else under schemas/, and a package METS whose
// dmdSec types the document as MODS 3.7. No swap: the record keeps the
// producer's identifier and no MEEMOO-LOCAL-ID is lifted (ADR-0012).
func TestBuildBibliographic(t *testing.T) {
	b, in, _ := newTestBuilder(t, bibliographicDef(t))
	rec := identityRecord()
	rec.Items = []mods.Item{{CallNumber: "EX.0001", Barcode: "000000123"}}
	in.Description = rec

	pkg, err := b.Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	doc, err := os.ReadFile(filepath.Join(pkg.Location, "metadata", "descriptive", "mods.xml"))
	if err != nil {
		t.Fatalf("mods.xml not written: %v", err)
	}
	for _, want := range []string{
		`<mods:mods `,
		`<mods:title>Catus Testus</mods:title>`,
		`<mods:itemIdentifier type="barcode">000000123</mods:itemIdentifier>`,
	} {
		if !strings.Contains(string(doc), want) {
			t.Errorf("mods.xml missing %s\n%s", want, doc)
		}
	}
	if got := identifierTerm(pkg.Root.Description); got != "local-id-001" {
		t.Errorf("description identifier = %q, want the producer's %q", got, "local-id-001")
	}
	if _, ok := pkg.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
		t.Error("MEEMOO-LOCAL-ID lifted onto the entity; ugent/bibliographic has no swap")
	}

	names := make([]string, 0, len(pkg.SchemaFiles))
	for _, sf := range pkg.SchemaFiles {
		names = append(names, sf.Name)
	}
	want := slices.Sorted(slices.Values(append(slices.Clone(mets.Schemas), "mods-3-7.xsd")))
	if !slices.Equal(names, want) {
		t.Errorf("schemas = %v, want the METS set plus the MODS schema %v", names, want)
	}

	metsDoc, err := os.ReadFile(filepath.Join(pkg.Location, "METS.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metsDoc), `MDTYPE="MODS" MDTYPEVERSION="3.7"`) {
		t.Errorf("package METS dmdSec does not type the document as MODS 3.7:\n%s", metsDoc)
	}
}
