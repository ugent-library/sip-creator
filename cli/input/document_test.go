package input

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input/mapping"
	"github.com/ugent-library/sip-creator/profiles"
)

// validDC and validMODS are the smallest documents the two eark profiles
// accept as supplied: well-formed XML with the standard's root element.
const (
	validDC   = `<?xml version="1.0"?><simpledc xmlns:dc="http://purl.org/dc/elements/1.1/"><title>Test</title></simpledc>`
	validMODS = `<?xml version="1.0"?><mods:mods xmlns:mods="http://www.loc.gov/mods/v3" version="3.7"><mods:titleInfo><mods:title>Test</mods:title></mods:titleInfo></mods:mods>`
)

// Under a profile that takes a document, dc.xml at the root is the
// package's description, an EncodedDescription pointing at the file on
// disk, and not content.
func TestDocumentAtRoot(t *testing.T) {
	root := writeTree(t, map[string]string{
		"dc.xml":    validDC,
		"scan.tiff": "x",
	})
	pkg, err := Read(root, mapping.EarkDC{}, earkDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	doc, ok := pkg.Description.(build.EncodedDescription)
	if !ok || doc.Source != filepath.Join(root, "dc.xml") {
		t.Errorf("Description = %#v, want an EncodedDescription at %s", pkg.Description, filepath.Join(root, "dc.xml"))
	}
	if got := paths(pkg.Representations[0].Files); strings.Join(got, ",") != "scan.tiff" {
		t.Errorf("content = %v; the document must not count as content", got)
	}
}

// A representation may carry a document of its own while the package is
// described by rows.
func TestDocumentInRepresentation(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                  minimalDC,
		"representations/master/scan.tiff": "x",
		"representations/master/dc.xml":    validDC,
	})
	pkg, err := Read(root, mapping.EarkDC{}, earkDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	rep := pkg.Representations[0]
	doc, ok := rep.Description.(build.EncodedDescription)
	if !ok || doc.Source != filepath.Join(root, "representations", "master", "dc.xml") {
		t.Errorf("representation Description = %#v, want its dc.xml", rep.Description)
	}
	if got := paths(rep.Files); strings.Join(got, ",") != "scan.tiff" {
		t.Errorf("content = %v; the document must not count as content", got)
	}
}

// One description per level: rows and a document together are a violation
// at the root and inside a representation alike.
func TestDocumentAndRowsTogether(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalDC,
		"dc.xml":                                 validDC,
		"representations/master/scan.tiff":       "x",
		"representations/master/description.csv": "key,value\ntitle,T\n",
		"representations/master/dc.xml":          validDC,
	})
	_, err := Read(root, mapping.EarkDC{}, earkDocumentSpec)
	assertViolation(t, err, "description.csv and dc.xml are both present; describe the package")
	assertViolation(t, err, "representations/master/description.csv and representations/master/dc.xml are both present; describe the representation")
}

// The package level needs one of the two, and the message names both under
// a profile that takes a document; under one that takes rows only it names
// the rows file alone.
func TestDocumentOrRowsRequired(t *testing.T) {
	_, err := Read(writeTree(t, map[string]string{"scan.tiff": "x"}), mapping.EarkDC{}, earkDocumentSpec)
	assertViolation(t, err, "needs a description.csv or a dc.xml")
	_, err = Read(writeTree(t, map[string]string{"scan.tiff": "x"}), mapping.Meemoo{}, meemooDocumentSpec)
	assertViolation(t, err, "needs a description.csv describing")
}

// A document must be well-formed XML with the profile's root element; the
// finding names the file, as the rows findings do.
func TestDocumentViolations(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string // substring of the expected violation
	}{
		{"not well-formed", "<simpledc><title>unclosed</simpledc>", "dc.xml: not well-formed XML"},
		{"not xml", "just text", "dc.xml: not an XML document"},
		{"empty", "", "dc.xml: not an XML document"},
		{"another standard's root", validMODS, "dc.xml: root element is {http://www.loc.gov/mods/v3}mods"},
		{"a wrapper", `<oai_dc:dc xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/"><title>T</title></oai_dc:dc>`, "dc.xml: root element is {http://www.openarchives.org/OAI/2.0/oai_dc/}dc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Read(writeTree(t, map[string]string{"dc.xml": tt.doc, "scan.tiff": "x"}), mapping.EarkDC{}, earkDocumentSpec)
			assertViolation(t, err, tt.want)
		})
	}
}

// The reserved name is for a file, at both levels.
func TestDocumentIsAFolder(t *testing.T) {
	root := writeTree(t, map[string]string{
		"dc.xml/oops.txt":                  "x",
		"description.csv":                  minimalDC,
		"representations/master/scan.tiff": "x",
		"representations/master/dc.xml/":   "",
	})
	_, err := Read(root, mapping.EarkDC{}, earkDocumentSpec)
	assertViolation(t, err, "dc.xml is a folder")
	assertViolation(t, err, "representations/master/dc.xml is a folder")
}

// Under a profile that takes no document the name is not reserved: a
// dc.xml under basic is content like any other file, as is a mods.xml
// under eark, another standard's document.
func TestDocumentNameIsContentElsewhere(t *testing.T) {
	pkg, err := Read(writeTree(t, map[string]string{
		"description.csv": minimalCSV,
		"dc.xml":          validDC,
		"scan.tiff":       "x",
	}), mapping.Meemoo{}, meemooDocumentSpec)
	if err != nil {
		t.Fatalf("Read under basic: %v", err)
	}
	if got := paths(pkg.Representations[0].Files); strings.Join(got, ",") != "dc.xml,scan.tiff" {
		t.Errorf("content under basic = %v, want dc.xml packaged as content", got)
	}

	pkg, err = Read(writeTree(t, map[string]string{
		"description.csv": minimalDC,
		"mods.xml":        validMODS,
		"scan.tiff":       "x",
	}), mapping.EarkDC{}, earkDocumentSpec)
	if err != nil {
		t.Fatalf("Read under eark: %v", err)
	}
	if got := paths(pkg.Representations[0].Files); strings.Join(got, ",") != "mods.xml,scan.tiff" {
		t.Errorf("content under eark = %v, want mods.xml packaged as content", got)
	}
}

// The folder's document builds with the real eark profile: the engine
// copies the file Read pointed at, byte for byte.
func TestDocumentBuilds(t *testing.T) {
	root := writeTree(t, map[string]string{"dc.xml": validDC, "scan.tiff": "x"})
	source, err := Read(root, mapping.EarkDC{}, earkDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	def, ok := profiles.Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}
	def, err = def.WithSubmitter("Test Org", "")
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	builder, err := build.New(&build.Config{Profile: def, Destination: dest, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := builder.Build(source); err != nil {
		t.Fatalf("Build: %v", err)
	}
	copies, _ := filepath.Glob(filepath.Join(dest, "uuid-*", "metadata", "descriptive", "dc.xml"))
	if len(copies) != 1 {
		t.Fatalf("want one dc.xml in the package, found %v", copies)
	}
	got, err := os.ReadFile(copies[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != validDC {
		t.Errorf("packaged dc.xml differs from the supplied file:\n%s", got)
	}
}
