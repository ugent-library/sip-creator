package dc

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

// The template interpolates element names from data, so encoding validates
// first (element names come from the closed set) and every value is
// escaped.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"esc": escapeXML,
}).Parse(`
{{ define "simpledc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .Schemas }}/dc.xsd">
{{- range .Terms }}
  <{{ .Element }}>{{ esc .Value }}</{{ .Element }}>
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
// dir.
func Encode(w io.Writer, t Terms, schemas string) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return simpledcTemplate.ExecuteTemplate(w, "simpledc", termsDoc{t, schemas})
}

// escapeXML makes a data value safe as XML character data; terms carry
// arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
