package build_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/sip"
)

// Producer and operator values carrying the XML-active characters (an
// essence file name, a local identifier, a label, a type, a content
// category, the submitter's name) build a package whose every XML document
// is well-formed, under the profile that writes PREMIS and under one that
// types its representations. The file name also carries the characters an
// href must percent-encode, and every href still names its file.
func TestBuildEscapesValues(t *testing.T) {
	const localID = `R&D <001> "a"`
	meemooTerms := testDescription()
	meemooTerms[0] = sip.Term{Key: "dcterms:identifier", Value: localID}

	cases := []struct {
		name        string
		def         build.Definition
		description sip.Description
	}{
		{"basic", basicDef(t), meemooTerms},
		{"eark", earkDef(t), eark.Terms{{Key: "identifier", Value: localID}, {Key: "title", Value: "Catus Testus"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			def, err := c.def.WithSubmitter(`R&D "Lab" <Gent>`, "OR-a1b2c3d")
			if err != nil {
				t.Fatal(err)
			}
			b, in, _ := newTestBuilder(t, def)
			in.Description = c.description
			in.ContentCategory = `Photographs & "Prints"`
			rep := &in.Representations[0]
			rep.Label = `Scans "R&D" <a>`
			rep.Type = `master & <scan>`
			rep.Files = []build.SourceFile{writeEssence(t, t.TempDir(), "R&D 'scan' 100% 1+2.tif", "essence bytes")}

			pkg, err := b.Build(in)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			requireWellFormedXML(t, pkg.Location)
			requireHrefsResolve(t, filepath.Join(pkg.Location, "METS.xml"))
			requireHrefsResolve(t, filepath.Join(pkg.Location, "representations", "master", "METS.xml"))
		})
	}
}

// requireWellFormedXML fails the test for every .xml file under dir that
// does not parse.
func requireWellFormedXML(t *testing.T, dir string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".xml") {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := xmldoc.Root(f); err != nil {
			rel, _ := filepath.Rel(dir, path)
			t.Errorf("%s: %v", rel, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

const xlinkNamespace = "http://www.w3.org/1999/xlink"

// requireHrefsResolve fails the test for every xlink:href in the METS
// document that, percent-decoded and taken relative to the document, names
// no file in the package.
func requireHrefsResolve(t *testing.T, metsPath string) {
	t.Helper()
	doc, err := os.ReadFile(metsPath)
	if err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("%s: %v", metsPath, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		for _, a := range start.Attr {
			if a.Name.Space != xlinkNamespace || a.Name.Local != "href" {
				continue
			}
			rel, err := url.PathUnescape(a.Value)
			if err != nil {
				t.Errorf("%s: href %q does not decode: %v", metsPath, a.Value, err)
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(metsPath), filepath.FromSlash(rel))); err != nil {
				t.Errorf("%s: href %q names no file: %v", metsPath, a.Value, err)
			}
		}
	}
}

// XML 1.0 carries every character except most control characters,
// U+FFFE and U+FFFF; a value must also be UTF-8. Escaping cannot carry the
// rest, so they are refused.
func TestValidateXMLText(t *testing.T) {
	tests := []struct {
		value string
		want  string // "" means accepted; else substring of the error
	}{
		{"", ""},
		{`R&D <a> "b" 'c'`, ""},
		{"two\tcolumns\nand lines\r\n", ""},
		{"caf\u00e9 \U0001F408 \uE000", ""},
		{"a\x00b", "U+0000"},
		{"a\x01b", "U+0001"},
		{"vertical\x0btab", "U+000B"},
		{"escape\x1b", "U+001B"},
		{"not a character \uFFFE", "U+FFFE"},
		{"caf\xe9", "not valid UTF-8"},
	}
	for _, tt := range tests {
		err := build.ValidateXMLText(tt.value)
		if tt.want == "" {
			if err != nil {
				t.Errorf("ValidateXMLText(%q) = %v, want accepted", tt.value, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("ValidateXMLText(%q) = %v, want an error mentioning %q", tt.value, err, tt.want)
		}
	}
}
