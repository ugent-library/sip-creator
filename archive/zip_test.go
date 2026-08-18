package archive

import (
	"archive/zip"
	"log/slog"
	"os"
	"path/filepath"
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
}

func TestZipUnreadableFileReturnsError(t *testing.T) {
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
