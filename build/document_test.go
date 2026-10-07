package build_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
)

// Sample documents of the shapes the two eark profiles emit themselves,
// plus the wrong shapes the checks must refuse.
const (
	simpleDCDocument = `<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/">
  <title>Example book</title>
  <identifier>example-0001</identifier>
</simpledc>
`
	modsDocument = `<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" version="3.7">
  <mods:identifier type="local">example-0001</mods:identifier>
  <mods:titleInfo><mods:title>Example book</mods:title></mods:titleInfo>
  <mods:name type="personal"><mods:namePart>Doe, Jane</mods:namePart></mods:name>
</mods:mods>
`
	modsOldVersion    = `<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" version="3.6"><mods:titleInfo><mods:title>x</mods:title></mods:titleInfo></mods:mods>`
	modsNoVersion     = `<mods:mods xmlns:mods="http://www.loc.gov/mods/v3"><mods:titleInfo><mods:title>x</mods:title></mods:titleInfo></mods:mods>`
	oaiDCDocument     = `<oai_dc:dc xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/"><title>x</title></oai_dc:dc>`
	malformedDocument = `<simpledc><title>x</simpledc>`
)

// writeDocument puts a document on disk and returns it as a build.EncodedDescription.
func writeDocument(t *testing.T, dir, name, content string) build.EncodedDescription {
	t.Helper()
	src := filepath.Join(dir, name)
	if err := os.WriteFile(src, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return build.EncodedDescription{Source: src}
}

// A document validates as a document: well-formed XML, whatever its root;
// the root is the profile's rule. A missing or malformed file is refused,
// and ValidateRequired trusts every document.
func TestDocumentValidate(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		doc  build.EncodedDescription
		want string // "" means valid; else substring of the error
	}{
		{"simpledc", writeDocument(t, dir, "dc.xml", simpleDCDocument), ""},
		{"mods", writeDocument(t, dir, "mods.xml", modsDocument), ""},
		{"another standard", writeDocument(t, dir, "oai.xml", oaiDCDocument), ""},
		{"malformed", writeDocument(t, dir, "bad.xml", malformedDocument), "not well-formed"},
		{"not xml", writeDocument(t, dir, "text.xml", "not xml"), "not an XML document"},
		{"missing", build.EncodedDescription{Source: filepath.Join(dir, "nope.xml")}, "no such file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				if err := tt.doc.ValidateRequired(); err != nil {
					t.Errorf("ValidateRequired = %v, want nothing required of a document", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}

// The two eark profiles build a package from a supplied document of their
// standard: the file lands under metadata/descriptive as it is, with
// fixity, and the METS types it as the profile declares. No swap, no
// MEEMOO-LOCAL-ID.
func TestBuildSuppliedDocument(t *testing.T) {
	cases := []struct {
		name     string
		def      build.Definition
		content  string
		wantFile string
		wantMETS string
	}{
		{"dc.xml under eark/dc", earkDef(t), simpleDCDocument, "dc.xml", `MDTYPE="DC"`},
		{"mods.xml under eark/mods", earkmodsDef(t), modsDocument, "mods.xml", `MDTYPE="MODS" MDTYPEVERSION="3.7"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, _ := newTestBuilder(t, c.def)
			doc := writeDocument(t, t.TempDir(), "supplied.xml", c.content)
			in.Description = doc

			pkg, err := b.Build(in)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			written, err := os.ReadFile(filepath.Join(pkg.Location, "metadata", "descriptive", c.wantFile))
			if err != nil {
				t.Fatalf("%s not written: %v", c.wantFile, err)
			}
			if string(written) != c.content {
				t.Errorf("the document was not copied as it is:\n%s", written)
			}
			df := pkg.Root.DescriptionFile
			if df.Checksum != fileMD5(t, doc.Source) || df.Size == "" {
				t.Errorf("fixity = %q/%q, want the source file's MD5 and a size", df.Checksum, df.Size)
			}
			mets, err := os.ReadFile(filepath.Join(pkg.Location, "METS.xml"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(mets), c.wantMETS) {
				t.Errorf("package METS lacks %s", c.wantMETS)
			}
			if _, ok := pkg.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
				t.Error("MEEMOO-LOCAL-ID lifted onto the entity; the eark profiles have no swap")
			}
		})
	}
}

// A document on a representation lands in that representation's
// descriptive dir and is referenced from its METS.
func TestBuildSuppliedDocumentOnRepresentation(t *testing.T) {
	b, in, _ := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = writeDocument(t, t.TempDir(), "dc.xml", simpleDCDocument)

	pkg, err := b.Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	rep := filepath.Join(pkg.Location, "representations", "master")
	if _, err := os.Stat(filepath.Join(rep, "metadata", "descriptive", "dc.xml")); err != nil {
		t.Fatalf("representation dc.xml not written: %v", err)
	}
	mets, err := os.ReadFile(filepath.Join(rep, "METS.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mets), `MDTYPE="DC" MDTYPEVERSION="SimpleDC20021212" xlink:type="simple" xlink:href="metadata/descriptive/dc.xml"`) {
		t.Errorf("representation METS does not reference the supplied document:\n%s", mets)
	}
}

// A document the profile cannot take is refused before any write: another
// standard's root, a MODS version other than the declared one, a version
// missing, a malformed file, a missing file, and any document at all under
// basic, whose metadata model does not implement DocumentFormat
// because its document needs the swap.
func TestBuildRefusesWrongDocument(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		def  build.Definition
		doc  build.EncodedDescription
		want string
	}{
		{"dc document to eark/mods", earkmodsDef(t), writeDocument(t, dir, "dc.xml", simpleDCDocument), "expected a mods:mods document"},
		{"mods document to eark", earkDef(t), writeDocument(t, dir, "mods.xml", modsDocument), "expected a simpledc document"},
		{"oai_dc document to eark", earkDef(t), writeDocument(t, dir, "oai.xml", oaiDCDocument), "expected a simpledc document"},
		{"mods 3.6 to eark/mods", earkmodsDef(t), writeDocument(t, dir, "old.xml", modsOldVersion), `version="3.6"`},
		{"mods without a version to eark/mods", earkmodsDef(t), writeDocument(t, dir, "nov.xml", modsNoVersion), "declares no version"},
		{"malformed to eark", earkDef(t), writeDocument(t, dir, "bad.xml", malformedDocument), "not well-formed"},
		{"missing file to eark", earkDef(t), build.EncodedDescription{Source: filepath.Join(dir, "nope.xml")}, "no such file"},
		{"any document to basic", basicDef(t), writeDocument(t, dir, "dcschema.xml", simpleDCDocument), "supplied descriptive document is not accepted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, c.def)
			in.Description = c.doc
			_, err := b.Build(in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Build error = %v, want %q", err, c.want)
			}
			requireEmpty(t, outDir)
		})
	}
}
