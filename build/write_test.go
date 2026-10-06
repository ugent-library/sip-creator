package build_test

import (
	"io/fs"
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
			requireReferencesMatchDisk(t, filepath.Join(pkg.Location, "METS.xml"))
			requireReferencesMatchDisk(t, filepath.Join(pkg.Location, "representations", "master", "METS.xml"))
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
