package simpledc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Model is the Simple Dublin Core metadata model, for a profile's
// definition.
var Model build.MetadataModel = model{}

// model is the Simple Dublin Core metadata model: it accepts Terms,
// writes them as a simpledc document with Encode, and says which supplied
// document is one of its own. It never swaps: dc.xml keeps the producer's identifier, because
// CSIP has no rule tying it to the package identifier and the ingesting
// catalogue indexes dc.xml, so operators find the package by the
// identifier they know (ADR-0012).
type model struct{}

// DocumentFormat is optional to the engine, so a drift in
// ValidateDocumentRoot's signature would fail silently; the assertion
// makes it a build error.
var _ build.DocumentFormat = model{}

func (model) ValidateType(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (simpledc.Terms)", d)
	}
	return nil
}

// ValidateDocumentRoot returns why root is not the simpledc element this
// template emits, without namespace. A document of another shape (an oai_dc
// wrapper, a MODS record) would make the METS declare a type the file does
// not have.
func (model) ValidateDocumentRoot(root xml.StartElement) error {
	if root.Name.Space != "" || root.Name.Local != "simpledc" {
		return fmt.Errorf("root element is {%s}%s, expected a simpledc document without namespace", root.Name.Space, root.Name.Local)
	}
	return nil
}

// Encode writes d as a Simple Dublin Core document: a simpledc root with
// one unqualified element per term, order preserved, language tags omitted.
func (model) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := simpledcTemplate.ExecuteTemplate(&buf, "simpledc", termsDoc{d.(Terms), schemasDir}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// ModelType types the document as DC.
func (model) ModelType() string {
	return "DC"
}

// ModelTypeVersion names the version of Simple Dublin Core the template
// writes, as the METS dmdSec declares it in MDTYPEVERSION.
func (model) ModelTypeVersion() string {
	return "SimpleDC20021212"
}

// Schemas returns the bundled XSDs the simpledc document points
// at: simpledc.xsd, the Simple DC container with its elements in no
// namespace as the template writes them, and xml.xsd, which it imports
// from the file next to it. It is not DCMI's file of that name, which
// expects the elements in the DCMES namespace and would reject the
// document; the schema's header records how it is built from DCMI's files.
func (model) Schemas() []build.Schema {
	return build.BundledSchemas("simpledc.xsd", "xml.xsd")
}

// simpledcTemplate escapes every value; element names come from
// elementName.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":  elementName,
	"esc": escapeXML,
}).Parse(`
{{ define "simpledc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .SchemasDir }}/simpledc.xsd">
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
