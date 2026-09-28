package eark

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"
)

// Schemas are the bundled XSD file names the simpledc document points at:
// dc.xsd alone. Its own import of xml.xsd is an absolute W3C URL, not a
// file next to it, so nothing else needs to ship. A profile that writes
// this document ships the list in its package's schemas/ dir.
var Schemas = []string{"dc.xsd"}

// The template interpolates element names from data. Every value is
// escaped, and the element name passes through el, which admits only the
// fifteen.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":  elementName,
	"esc": escapeXML,
}).Parse(`
{{ define "simpledc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .Schemas }}/dc.xsd">
{{- range .Terms }}
  <{{ el .Element }}>{{ esc .Value }}</{{ el .Element }}>
{{- end }}
</simpledc>
{{ end }}
`))

// termsDoc is one descriptive document to render: the terms plus the
// relative path from the document's location to the package's bundled
// schemas/ dir.
type termsDoc struct {
	Terms   Terms
	Schemas string
}

// Encode writes the terms as a Simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively): one
// unqualified element per term, order preserved, language tags omitted.
// schemas is the relative path from the document to the package's schemas/
// dir. t must be valid: Terms.Validate is the contract, run by the builder
// before any write, and Encode does not repeat it. The document is
// rendered in memory first, so a refused term writes nothing.
func Encode(w io.Writer, t Terms, schemas string) error {
	var buf bytes.Buffer
	if err := simpledcTemplate.ExecuteTemplate(&buf, "simpledc", termsDoc{t, schemas}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// elementName is the template's one guard: the element name is the only
// thing the template interpolates raw, and only one of the fifteen may
// reach the output. Returning an error aborts the render.
func elementName(element string) (string, error) {
	if !elementSet[element] {
		return "", fmt.Errorf("%q is not a Simple Dublin Core element", element)
	}
	return element, nil
}

// escapeXML makes a data value safe as XML character data; terms carry
// arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
