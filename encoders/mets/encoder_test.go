package mets

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// descriptiveGraph returns the smallest package and representation whose
// METS documents carry a dmdSec: a root entity with a descriptive file
// node, and one representation with its own, both with the given mdType
// and version.
func descriptiveGraph(t *testing.T, mdType, version string) (*sip.Package, *sip.Representation) {
	t.Helper()
	generated := func(path string) *sip.File {
		f := sip.NewFile()
		f.Path = path
		f.Mime = "text/xml"
		return f
	}
	descriptive := func() *sip.File {
		f := generated("metadata/descriptive/descriptive.xml")
		f.MDType, f.MDTypeVersion = mdType, version
		return f
	}

	pkg := sip.NewPackage(t.TempDir(), "")
	pkg.Declaration = &sip.MetsDeclaration{}
	pkg.Root = sip.NewEntity()
	pkg.Root.DescriptionFile = descriptive()

	rep := sip.NewRepresentation("master")
	rep.Label = "master"
	rep.Declaration = &sip.MetsDeclaration{}
	rep.Entity = pkg.Root
	rep.DescriptionFile = descriptive()
	rep.MetsFile = generated("representations/master/METS.xml") // the package METS references it
	pkg.Root.Representations = []*sip.Representation{rep}
	return pkg, rep
}

// The dmdSec mdRef takes the type of the descriptive document from its file
// node, in the package METS and in the representation METS. MDTYPE is always
// written. MDTYPEVERSION is written only when the node carries one, as the
// Simple DC and MODS nodes do, and never as an empty attribute.
func TestDmdSecTyping(t *testing.T) {
	cases := []struct {
		name            string
		mdType, version string
		want            string
		wantNot         string
	}{
		{"type and version", "MODS", "3.7", `MDTYPE="MODS" MDTYPEVERSION="3.7" xlink:type`, ""},
		{"type alone", "DC", "", `MDTYPE="DC" xlink:type`, "MDTYPEVERSION"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pkg, rep := descriptiveGraph(t, c.mdType, c.version)
			documents := map[string]func(io.Writer) error{
				"package":        func(w io.Writer) error { return EncodePackage(w, pkg) },
				"representation": func(w io.Writer) error { return EncodeRepresentation(w, rep) },
			}
			for name, encode := range documents {
				var buf bytes.Buffer
				if err := encode(&buf); err != nil {
					t.Fatalf("%s METS: %v", name, err)
				}
				out := buf.String()
				if got := strings.Count(out, "<dmdSec "); got != 1 {
					t.Errorf("%s METS has %d dmdSec elements, want 1\n%s", name, got, out)
				}
				if !strings.Contains(out, c.want) {
					t.Errorf("%s METS dmdSec lacks %s\n%s", name, c.want, out)
				}
				if c.wantNot != "" && strings.Contains(out, c.wantNot) {
					t.Errorf("%s METS dmdSec carries %s, which the file node leaves empty\n%s", name, c.wantNot, out)
				}
			}
		})
	}
}

// Every namespace that xsi:schemaLocation pairs with a schema is one the
// document declares, spelled exactly. Namespace names compare character by
// character, so a hint for https://dilcis.eu/... does not apply to the
// https://DILCIS.eu/... namespace that CSIP and the SIP specification use.
func TestSchemaLocationNamesDeclaredNamespaces(t *testing.T) {
	pkg, rep := descriptiveGraph(t, "DC", "")
	documents := map[string]func(io.Writer) error{
		"package":        func(w io.Writer) error { return EncodePackage(w, pkg) },
		"representation": func(w io.Writer) error { return EncodeRepresentation(w, rep) },
	}
	for name, encode := range documents {
		var buf bytes.Buffer
		if err := encode(&buf); err != nil {
			t.Fatalf("%s METS: %v", name, err)
		}
		root, err := xmldoc.Root(&buf)
		if err != nil {
			t.Fatalf("%s METS: %v", name, err)
		}

		declared := map[string]bool{}
		var schemaLocation string
		for _, a := range root.Attr {
			switch {
			case a.Name.Space == "xmlns", a.Name.Space == "" && a.Name.Local == "xmlns":
				declared[a.Value] = true
			case a.Name.Space == "http://www.w3.org/2001/XMLSchema-instance" && a.Name.Local == "schemaLocation":
				schemaLocation = a.Value
			}
		}

		pairs := strings.Fields(schemaLocation)
		if len(pairs) == 0 || len(pairs)%2 != 0 {
			t.Fatalf("%s METS xsi:schemaLocation %q is not namespace/schema pairs", name, schemaLocation)
		}
		for i := 0; i < len(pairs); i += 2 {
			if !declared[pairs[i]] {
				t.Errorf("%s METS xsi:schemaLocation names %s, which the document does not declare", name, pairs[i])
			}
		}
	}
}

// Values from producers and operators reach the documents escaped. A file
// path, a label and an agent name that carry the XML-active characters
// leave both documents well-formed. An XML reader gets the label and the
// name back as they were given, and the path as its href.
func TestEscapesGraphValues(t *testing.T) {
	const (
		path     = `data/R&D "1" <a>.tif`
		wantHref = `data/R%26D%20%221%22%20%3Ca%3E.tif`
		label    = `Scans "R&D" <a>`
		agent    = `R&D <Lab> "Gent"`
	)
	pkg, rep := descriptiveGraph(t, "DC", "")
	pkg.Declaration.Agents = []sip.Agent{{Role: "CREATOR", Type: "ORGANIZATION", Name: agent}}
	rep.Label = label
	essence := sip.NewFile()
	essence.Path = path
	essence.Mime = "image/tiff"
	rep.Files = []*sip.File{essence}

	var repMETS, pkgMETS bytes.Buffer
	if err := EncodeRepresentation(&repMETS, rep); err != nil {
		t.Fatalf("representation METS: %v", err)
	}
	if err := EncodePackage(&pkgMETS, pkg); err != nil {
		t.Fatalf("package METS: %v", err)
	}

	repValues := decodedValues(t, repMETS.Bytes())
	for _, want := range []string{wantHref, label} {
		if !repValues[want] {
			t.Errorf("representation METS does not carry %q as a value\n%s", want, repMETS.String())
		}
	}
	if !decodedValues(t, pkgMETS.Bytes())[agent] {
		t.Errorf("package METS does not carry the agent name %q\n%s", agent, pkgMETS.String())
	}
}

// decodedValues reads the whole document and returns every attribute value
// and text node as an XML reader decodes them. It fails the test if the
// document is not well-formed.
func decodedValues(t *testing.T, doc []byte) map[string]bool {
	t.Helper()
	values := map[string]bool{}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return values
		}
		if err != nil {
			t.Fatalf("not well-formed: %v\n%s", err, doc)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			for _, a := range tok.Attr {
				values[a.Value] = true
			}
		case xml.CharData:
			values[string(tok)] = true
		}
	}
}

// An href keeps the unreserved characters and the slashes between
// segments, percent-encodes every other byte, and decodes to the original
// path.
func TestHref(t *testing.T) {
	tests := []struct{ path, want string }{
		{"data/image-001.jpg", "data/image-001.jpg"},
		{"metadata/descriptive/dc+schema.xml", "metadata/descriptive/dc%2Bschema.xml"},
		{"data/R&D scan.tif", "data/R%26D%20scan.tif"},
		{"data/100%.jpg", "data/100%25.jpg"},
		{"data/a#b?.jpg", "data/a%23b%3F.jpg"},
		{"data/sub/~v1_final.tif", "data/sub/~v1_final.tif"},
		{"data/caf\u00e9.tif", "data/caf%C3%A9.tif"},
	}
	for _, tt := range tests {
		if got := href(tt.path); got != tt.want {
			t.Errorf("href(%q) = %q, want %q", tt.path, got, tt.want)
		}
		if decoded, err := url.PathUnescape(tt.want); err != nil || decoded != tt.path {
			t.Errorf("%q decodes to %q (%v), want %q", tt.want, decoded, err, tt.path)
		}
	}
}
