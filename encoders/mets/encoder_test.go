package mets

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// descriptiveGraph returns the smallest package and representation whose
// METS documents carry a dmdSec: a root entity with a descriptive file
// node, and one representation with its own, both labeled mdType and
// version. The graph is built by assigning fields, as the assembler does.
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

// The dmdSec mdRef types the descriptive document from its file node, in
// the package METS and the representation METS alike: MDTYPE always,
// MDTYPEVERSION only when the node carries one (Meemoo's model declares
// none; Simple DC and MODS do), never as an empty attribute.
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

// Every namespace xsi:schemaLocation pairs with a schema is one the
// document declares, spelled exactly: namespace names compare character by
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
