package eark

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// simpledc is the encoder for the simpledc document: it accepts Terms and
// writes them with Encode, and says which supplied document is one of its
// own. It never swaps: dc.xml keeps the producer's identifier, because
// CSIP has no rule tying it to the package identifier and the ingesting
// catalogue indexes dc.xml, so operators find the package by the
// identifier they know (ADR-0012).
type simpledc struct{}

// DocumentFormat is optional to the engine, so a drift in
// ValidateDocumentRoot's signature would fail silently; the assertion
// makes it a build error.
var (
	_ build.DescriptionEncoder = simpledc{}
	_ build.DocumentFormat     = simpledc{}
)

func (simpledc) Check(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (eark.Terms)", d)
	}
	return nil
}

// ValidateDocumentRoot returns why root is not the simpledc element this
// template emits, without namespace. A document of another shape (an oai_dc
// wrapper, a MODS record) would make the METS declare a type the file does
// not have.
func (simpledc) ValidateDocumentRoot(root xml.StartElement) error {
	if root.Name.Space != "" || root.Name.Local != "simpledc" {
		return fmt.Errorf("root element is {%s}%s, expected a simpledc document without namespace", root.Name.Space, root.Name.Local)
	}
	return nil
}

// Encode writes d as a Simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively): one
// unqualified element per term, order preserved, language tags omitted.
func (simpledc) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := simpledcTemplate.ExecuteTemplate(&buf, "simpledc", termsDoc{d.(Terms), schemasDir}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Schemas lists the bundled XSD file names the simpledc document points
// at: dc.xsd alone. Its own import of xml.xsd is an absolute W3C URL, not
// a file next to it, so nothing else needs to ship.
func (simpledc) Schemas() []string {
	return []string{"dc.xsd"}
}

// simpledcTemplate escapes every value; element names come from
// elementName.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":  elementName,
	"esc": escapeXML,
}).Parse(`
{{ define "simpledc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .SchemasDir }}/dc.xsd">
{{- range .Terms }}
  <{{ el .Key }}>{{ esc .Value }}</{{ el .Key }}>
{{- end }}
</simpledc>
{{ end }}
`))

// termsDoc is one descriptive document to render: the terms plus the path
// of the package's schemas/ directory relative to the document.
type termsDoc struct {
	Terms      Terms
	SchemasDir string
}

// elementName is the element a key emits (in Simple Dublin Core, the key
// itself), and the template's one guard: the element name is the only
// thing the template interpolates raw, and only one of the fifteen may
// reach the output. Returning an error aborts the render.
func elementName(key string) (string, error) {
	if !elementSet[key] {
		return "", fmt.Errorf("unknown key %q: not a Simple Dublin Core element", key)
	}
	return key, nil
}

// escapeXML makes a data value safe as XML character data; terms carry
// arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
