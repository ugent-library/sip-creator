package build_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

// Producer and operator values that contain &, <, > and quotes build a
// package in which every XML document is well-formed. The values are an
// essence file name, a local identifier, a label, a type, a content
// category and the submitter's name. The test runs under the profile that
// writes PREMIS and under one that types its representations. The file
// name also contains characters an href must percent-encode, and every
// href still names its file.
func TestBuildEscapesValues(t *testing.T) {
	const localID = `R&D <001> "a"`
	meemooTerms := testDescription()
	meemooTerms[0] = sip.Term{Key: "dcterms:identifier", Value: localID}

	cases := []struct {
		name        string
		def         build.Definition
		description sip.Description
	}{
		{"meemoo/basic", basicDef(t), meemooTerms},
		// Without its vocabulary, so that the representation's type is free
		// text and its escaping is exercised.
		{"ugent/basic", ugentBasicWithoutVocabulary(t), ugent.Terms{{Key: "identifier", Value: localID}, {Key: "title", Value: "Catus Testus"}}},
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
			requireReferencesMatchDisk(t, filepath.Join(pkg.Location, "representations", "archival", "METS.xml"))
		})
	}
}

// requireWellFormedXML checks that every .xml file under dir is well-formed
// XML. It fails the test for each file that is not.
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
