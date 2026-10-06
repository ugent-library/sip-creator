package archive

import (
	"archive/zip"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

func testArchive(baseDir string) *Archive {
	return New(&Config{
		Destination: baseDir,
		Logger:      slog.New(slog.DiscardHandler),
	})
}

// writePackage lays out a minimal package tree under baseDir and returns
// the package pointing at it.
func writePackage(t *testing.T, baseDir string) *sip.Package {
	t.Helper()

	pkg := &sip.Package{
		Identifier: "uuid-test",
		Location:   filepath.Join(baseDir, "uuid-test"),
	}
	if err := os.MkdirAll(filepath.Join(pkg.Location, "representations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg.Location, "METS.xml"), []byte("<mets/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// A config without a logger zips the package; the messages are discarded.
func TestZipWithoutLogger(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)

	if err := New(&Config{Destination: baseDir}).Zip(pkg); err != nil {
		t.Fatalf("Zip() = %v, want nil", err)
	}
}

func TestZip(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)

	if err := testArchive(baseDir).Zip(pkg); err != nil {
		t.Fatalf("Zip() = %v, want nil", err)
	}

	r, err := zip.OpenReader(filepath.Join(baseDir, "uuid-test.zip"))
	if err != nil {
		t.Fatalf("opening produced zip: %v", err)
	}
	defer r.Close()

	names := make(map[string]bool, len(r.File))
	for _, f := range r.File {
		names[f.Name] = true
	}
	for _, want := range []string{"uuid-test/METS.xml", "uuid-test/representations/"} {
		if !names[want] {
			t.Errorf("zip is missing entry %q (got %v)", want, names)
		}
	}
	// The temporary file is gone once the zip has its final name.
	requireEntries(t, baseDir, "uuid-test", "uuid-test.zip")
}

// The zip gets the permissions a file made with os.Create gets, so the
// temporary file changes nothing for whoever reads the zip next.
func TestZipKeepsTheUsualPermissions(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)
	if err := testArchive(baseDir).Zip(pkg); err != nil {
		t.Fatalf("Zip() = %v, want nil", err)
	}

	reference, err := os.Create(filepath.Join(t.TempDir(), "reference"))
	if err != nil {
		t.Fatal(err)
	}
	reference.Close()
	want, err := os.Stat(reference.Name())
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Stat(filepath.Join(baseDir, "uuid-test.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Mode().Perm() != want.Mode().Perm() {
		t.Errorf("zip mode = %v, want %v as os.Create makes it", got.Mode().Perm(), want.Mode().Perm())
	}
}

// A zip that already exists is never replaced: it may be one a transfer
// has picked up, or another build's.
func TestZipRefusesAnExistingZip(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)
	existing := filepath.Join(baseDir, "uuid-test.zip")
	if err := os.WriteFile(existing, []byte("an earlier zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := testArchive(baseDir).Zip(pkg)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Zip() = %v, want the existing zip refused", err)
	}
	if got, _ := os.ReadFile(existing); string(got) != "an earlier zip" {
		t.Errorf("the existing zip was changed: %q", got)
	}
	requireEntries(t, baseDir, "uuid-test", "uuid-test.zip")
}

// A temporary file left by a run that was killed partway does not stop
// the next run, which writes over it and renames it.
func TestZipWritesOverAStaleTemporaryFile(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)
	if err := os.WriteFile(filepath.Join(baseDir, ".uuid-test.zip.tmp"), []byte("half a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := testArchive(baseDir).Zip(pkg); err != nil {
		t.Fatalf("Zip() = %v, want nil", err)
	}
	r, err := zip.OpenReader(filepath.Join(baseDir, "uuid-test.zip"))
	if err != nil {
		t.Fatalf("the zip does not open: %v", err)
	}
	r.Close()
	requireEntries(t, baseDir, "uuid-test", "uuid-test.zip")
}

// dataDescriptorFlag is general-purpose flag bit 3: the sizes and CRC
// follow the entry data in a trailing descriptor. Java's ZipInputStream
// rejects stored entries carrying it, so no file entry may set it.
const dataDescriptorFlag = 0x8

func TestZipFillsLocalFileHeaders(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)

	if err := testArchive(baseDir).Zip(pkg); err != nil {
		t.Fatalf("Zip() = %v, want nil", err)
	}

	r, err := zip.OpenReader(filepath.Join(baseDir, "uuid-test.zip"))
	if err != nil {
		t.Fatalf("opening produced zip: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Method != zip.Store {
			t.Errorf("%s: method = %d, want Store (%d)", f.Name, f.Method, zip.Store)
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Flags&dataDescriptorFlag != 0 {
			t.Errorf("%s: data descriptor flag set; ZipInputStream cannot read a stored entry with it", f.Name)
		}
		if f.CRC32 == 0 {
			t.Errorf("%s: CRC32 is zero, header was not filled", f.Name)
		}
		if want := uint64(len("<mets/>")); f.UncompressedSize64 != want {
			t.Errorf("%s: size = %d, want %d", f.Name, f.UncompressedSize64, want)
		}
	}
}

func TestZipUnreadableFileReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not make a file unreadable through its mode bits")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root: file permissions are not enforced")
	}

	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)

	unreadable := filepath.Join(pkg.Location, "secret.xml")
	if err := os.WriteFile(unreadable, []byte("<x/>"), 0o000); err != nil {
		t.Fatal(err)
	}

	if err := testArchive(baseDir).Zip(pkg); err == nil {
		t.Fatal("Zip() = nil, want error for unreadable file")
	}
	// Neither a zip under the final name nor the temporary file is left.
	requireEntries(t, baseDir, "uuid-test")
}

func TestZipUnwritableDestinationReturnsError(t *testing.T) {
	baseDir := t.TempDir()
	pkg := writePackage(t, baseDir)

	// Point the archive's output at a directory that doesn't exist so
	// os.Create fails.
	a := testArchive(filepath.Join(baseDir, "missing"))
	if err := a.Zip(pkg); err == nil {
		t.Fatal("Zip() = nil, want error for unwritable destination")
	}
}

// requireEntries fails the test unless dir holds exactly the named entries.
func requireEntries(t *testing.T, dir string, want ...string) {
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
