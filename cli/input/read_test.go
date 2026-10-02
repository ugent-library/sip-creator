package input

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// meemooVocab and earkVocab are the tests' vocabularies for the two flat
// profiles: the statements as stated, wrapped as the profile's terms. The
// CLI's own vocabularies in cli/input/vocabulary import this package, so
// its tests cannot use them. earkVocab also takes a dc.xml, judged by the
// eark metadata model, so Read's document rules are tested through it.
type meemooVocab struct{}

func (meemooVocab) Description(statements []Statement) (sip.Description, []error) {
	return meemoo.Terms(terms(statements)), nil
}

type earkVocab struct{}

func (earkVocab) Description(statements []Statement) (sip.Description, []error) {
	return eark.Terms(terms(statements)), nil
}

func (earkVocab) DocumentName() string { return "dc.xml" }

func (earkVocab) ValidateDocumentRoot(root xml.StartElement) error {
	return eark.Definition.Model.(build.DocumentFormat).ValidateDocumentRoot(root)
}

func terms(statements []Statement) []sip.Term {
	out := make([]sip.Term, len(statements))
	for i, s := range statements {
		out[i] = sip.Term{Key: s.Key, Lang: s.Lang, Value: s.Value}
	}
	return out
}

// minimalCSV is the smallest description.csv that passes check: Meemoo's
// basic content profile requires these four keys. minimalDC is its eark
// counterpart: Simple DC requires identity only.
const (
	minimalCSV = "key,value\nidentifier,ID-1\ntitle,Test\ndescription,Testbeschrijving\ncreated,2026\n"
	minimalDC  = "key,value\nidentifier,ID-1\ntitle,Test\n"
)

// validPremis is the smallest document the received-preservation rules
// accept: well-formed, premis:premis root, PREMIS 3 namespace.
const validPremis = `<?xml version="1.0"?><premis:premis xmlns:premis="http://www.loc.gov/premis/v3" version="3.0"><premis:event/></premis:premis>`

// writeTree builds an input folder from slash paths: a key ending in "/"
// creates an empty folder, anything else a file with the given content.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(abs, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func assertViolation(t *testing.T, err error, substr string) {
	t.Helper()
	var v Violations
	if !errors.As(err, &v) {
		t.Fatalf("want Violations mentioning %q, got %v", substr, err)
	}
	for _, line := range v {
		if strings.Contains(line, substr) {
			return
		}
	}
	t.Errorf("no violation mentions %q; got:\n%s", substr, v.Error())
}

// assertFinding checks that one of a parser's findings mentions substr.
func assertFinding(t *testing.T, errs []error, substr string) {
	t.Helper()
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return
		}
	}
	t.Errorf("no finding mentions %q; got: %v", substr, errs)
}

func paths(files []build.SourceFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

func TestReadFlat(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
		"0002.tiff":       "b",
		"0010.tiff":       "c",
		"0001.tiff":       "a",
		"sub/0003.tiff":   "d",
	})

	pkg, err := Read(root, meemooVocab{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(pkg.Representations) != 1 {
		t.Fatalf("want 1 representation, got %d", len(pkg.Representations))
	}
	rep := pkg.Representations[0]
	if rep.Name != filepath.Base(root) {
		t.Errorf("flat name = %q, want the folder name %q", rep.Name, filepath.Base(root))
	}

	// Deterministic traversal order (lexical per folder); the order
	// carries no semantics, but it must be stable run to run.
	want := []string{"0001.tiff", "0002.tiff", "0010.tiff", "sub/0003.tiff"}
	got := paths(rep.Files)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("file order = %v, want %v", got, want)
	}

	f := rep.Files[3]
	if f.Key != "sub/0003.tiff" {
		t.Errorf("Key = %q, want input-root-relative %q", f.Key, "sub/0003.tiff")
	}
	if f.Source != filepath.Join(root, "sub", "0003.tiff") {
		t.Errorf("Source = %q, want the absolute disk path", f.Source)
	}
	if pkg.Description == nil {
		t.Error("package descriptive terms missing")
	}
	if pkg.Characterization != nil {
		t.Error("no sidecar in the tree, but a characterization report was returned")
	}
}

func TestReadRepresentations(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                           minimalCSV,
		"siegfried.json":                            `{"siegfried":"1.11.0","files":[]}`,
		"documentation/report.pdf":                  "r",
		"premis/vendor.xml":                         validPremis,
		"representations/master/scan_2.tiff":        "b",
		"representations/master/scan_10.tiff":       "c",
		"representations/access/book.pdf":           "p",
		"representations/access/description.csv":    "key,value\ntitle,PDF-versie\n",
		"representations/access/documentation/n.md": "n",
		"representations/access/premis/ocr.xml":     validPremis,
	})

	pkg, err := Read(root, meemooVocab{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(pkg.Representations) != 2 {
		t.Fatalf("want 2 representations, got %d", len(pkg.Representations))
	}

	// os.ReadDir lists lexically, so access comes first.
	access, master := pkg.Representations[0], pkg.Representations[1]
	if access.Name != "access" || master.Name != "master" {
		t.Fatalf("names = %q, %q", access.Name, master.Name)
	}

	if got := paths(master.Files); strings.Join(got, ",") != "scan_10.tiff,scan_2.tiff" {
		t.Errorf("master file order = %v, want lexical traversal order", got)
	}
	if master.Description != nil {
		t.Error("master has no description.csv but carries descriptive terms")
	}
	if got, ok := access.Description.(meemoo.Terms); !ok || len(got) != 1 || got[0].Key != "title" {
		t.Errorf("access descriptive = %#v, want its title term", access.Description)
	}
	if got := paths(access.Files); strings.Join(got, ",") != "book.pdf" {
		t.Errorf("access content = %v; reserved names must not count as content", got)
	}
	if len(access.Documentation) != 1 || len(access.Premis) != 1 {
		t.Errorf("access documentation/premis = %d/%d, want 1/1", len(access.Documentation), len(access.Premis))
	}

	if len(pkg.Documentation) != 1 || pkg.Documentation[0].Path != "report.pdf" {
		t.Errorf("package documentation = %v", paths(pkg.Documentation))
	}
	if len(pkg.Premis) != 1 || pkg.Premis[0].Key != "premis/vendor.xml" {
		t.Errorf("package premis = %v", pkg.Premis)
	}
	if pkg.Characterization == nil {
		t.Error("sidecar present but no characterization report decoded")
	}
}

func TestReadCollectsAllViolations(t *testing.T) {
	root := writeTree(t, map[string]string{
		// no description.csv
		"stray.tiff":                         "x", // content beside representations/
		"representations/loose.txt":          "x", // file directly inside representations/
		"representations/bad name/scan.tiff": "x", // rep-name character rule
		"representations/empty/":             "",  // no content files
	})

	_, err := Read(root, meemooVocab{})
	if err == nil {
		t.Fatal("want violations, got none")
	}
	assertViolation(t, err, "descriptive rows are missing")
	assertViolation(t, err, "stray.tiff")
	assertViolation(t, err, "loose.txt")
	assertViolation(t, err, "bad name")
	assertViolation(t, err, "representations/empty")

	var v Violations
	errors.As(err, &v)
	if len(v) < 5 {
		t.Errorf("want all 5 findings in one pass, got %d:\n%s", len(v), v.Error())
	}
}

func TestReadSymlink(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
		"scan.tiff":       "x",
	})
	if err := os.Symlink(filepath.Join(root, "scan.tiff"), filepath.Join(root, "link.tiff")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "symbolic link")
}

func TestReadIgnoresOSArtifacts(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
		"scan.tiff":       "x",
		".DS_Store":       "junk",
		"._scan.tiff":     "junk",
		"sub/Thumbs.db":   "junk",
		"sub/desktop.ini": "junk",
		"sub/0001.tiff":   "x",
	})

	pkg, err := Read(root, meemooVocab{})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	want := []string{"scan.tiff", "sub/0001.tiff"}
	got := paths(pkg.Representations[0].Files)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("content = %v, want %v (artifacts silently ignored)", got, want)
	}
}

func TestReadArtifactsAreNotContent(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                  minimalCSV,
		"representations/master/.DS_Store": "junk",
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "no content files")
}

func TestReadEmptyRepresentationsDir(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":  minimalCSV,
		"representations/": "",
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "no representation folders")
}

func TestReadNoContent(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "no content files")
}

func TestReadReservedNameWrongKind(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv/oops.txt": "x", // reserved file name used as a folder
		"documentation":            "x", // reserved folder name used as a file
		"scan.tiff":                "x",
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "description.csv is a folder")
	assertViolation(t, err, "documentation is a file")
}

// The read-time premis rule is the naming rule only: premis.xml belongs to
// the generated document. Content conformance is deliberately NOT a read
// concern; it is enforced at assembly, like the characterization MD5
// verification, so a folder with malformed premis content passes Read/check and
// fails at build.
func TestReadPremisNamingRule(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":    minimalCSV,
		"scan.tiff":          "x",
		"premis/premis.xml":  validPremis,
		"premis/garbage.xml": "not xml; read does not judge content",
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "premis.xml is reserved")

	var v Violations
	errors.As(err, &v)
	if len(v) != 1 {
		t.Errorf("want only the naming violation, got:\n%s", v.Error())
	}
}

func TestReadBadSidecar(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
		"scan.tiff":       "x",
		"siegfried.json":  `{"not":"a report"}`,
	})

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "siegfried.json")
}

func TestReadNFCCollision(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": minimalCSV,
	})
	// The same name in NFC and NFD form; they can coexist only on a
	// filesystem that does not normalize names (e.g. ext4).
	nfc := "caf\u00e9.tiff"  // é precomposed
	nfd := "cafe\u0301.tiff" // e + combining acute
	os.WriteFile(filepath.Join(root, nfc), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, nfd), []byte("y"), 0o644)
	entries, _ := os.ReadDir(root)
	if len(entries) != 3 {
		t.Skip("filesystem normalizes names; the collision cannot exist here")
	}

	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "Unicode normalization")
}
