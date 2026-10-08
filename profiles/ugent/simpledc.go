package ugent

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// simpledc is the Simple Dublin Core metadata model of ugent/basic, for
// Terms. dc.xml keeps the producer's identifier, because the catalogue
// indexes dc.xml and operators search by that identifier (ADR-0012).
type simpledc struct{}

// Definition.ValidateSource accepts a supplied document only from a model
// that implements DocumentFormat. If ValidateDocumentRoot's signature
// changed, it would refuse every supplied dc.xml. These assertions turn
// that into a compile error.
var (
	_ build.MetadataModel  = simpledc{}
	_ build.DocumentFormat = simpledc{}
)

func (simpledc) ValidateType(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (ugent.Terms)", d)
	}
	return nil
}

// ValidateDocumentRoot checks that root is a simpledc element without a
// namespace, as the template writes it. It returns an error if it is not.
// A document of another shape, such as an oai_dc wrapper or a MODS record,
// would make the METS declare a type the file does not have.
func (simpledc) ValidateDocumentRoot(root xml.StartElement) error {
	if root.Name.Space != "" || root.Name.Local != "simpledc" {
		return fmt.Errorf("root element is {%s}%s, expected a simpledc document without namespace", root.Name.Space, root.Name.Local)
	}
	return nil
}

// Encode writes d as a Simple Dublin Core document: a simpledc root with
// one unqualified element per term, in the order of the terms, without
// language tags.
func (simpledc) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := simpledcTemplate.ExecuteTemplate(&buf, "simpledc", termsDoc{d.(Terms), schemasDir}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// ModelType returns DC.
func (simpledc) ModelType() string {
	return "DC"
}

// ModelTypeVersion returns the version of Simple Dublin Core the template
// writes.
func (simpledc) ModelTypeVersion() string {
	return "SimpleDC20021212"
}

// Schemas returns the bundled XSDs the simpledc document points at:
// simpledc.xsd, the Simple DC container with its elements in no namespace
// as the template writes them, and xml.xsd, which simpledc.xsd imports
// from the file next to it. This simpledc.xsd is not DCMI's file of that
// name, which expects the elements in the DCMES namespace and would reject
// the document. Its header records how it is built from DCMI's files.
func (simpledc) Schemas() []build.Schema {
	return build.BundledSchemas("simpledc.xsd", "xml.xsd")
}

// simpledcTemplate escapes every value. Element names come from
// dcElementName, which lets only the fifteen elements through.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":  dcElementName,
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

// dcElementName returns the element name for a key, which in Simple Dublin
// Core is the key itself. It returns an error for a key outside the
// fifteen elements, which aborts the render, because the template writes
// the name unescaped.
func dcElementName(key string) (string, error) {
	if !dcElementSet[key] {
		return "", fmt.Errorf("unknown key %q: not a Simple Dublin Core element", key)
	}
	return key, nil
}

// escapeXML returns s escaped for use as XML character data or as a quoted
// attribute value.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
