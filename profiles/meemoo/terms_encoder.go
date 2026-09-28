package meemoo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"
)

// Schemas are the bundled XSD file names the dc+schema document points at,
// plus what those import by relative path: meemoo's descriptive_basic.xsd
// imports the Dublin Core, DCMI type, EDTF and schema.org schemas and
// xml.xsd. A profile that writes this document ships the list in its
// package's schemas/ dir.
var Schemas = []string{"descriptive_basic.xsd", "dc.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd"}

// The template interpolates element names from data. Every value is
// escaped, and the element name passes through el, which admits only names
// the vocabulary lists.
var termsTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":      elementName,
	"esc":     escapeXML,
	"xsitype": xsiType,
}).Parse(`
{{ define "dcschema" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/1.2/basic"
  xmlns:dcterms="http://purl.org/dc/terms/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:edtf="http://id.loc.gov/datatypes/edtf/"
  xmlns:schema="https://schema.org/"
  xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic {{ .Schemas }}/descriptive_basic.xsd">
{{- range .Terms }}
  <{{ el .Element }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ with xsitype .Element }} xsi:type="{{ . }}"{{ end }}>{{ esc .Value }}</{{ el .Element }}>
{{- end }}
</metadata>
{{ end }}
`))

// termsDoc is one descriptive document to render: the terms plus the
// relative path from the document's location to the package's bundled
// schemas/ dir.
type termsDoc struct {
	Terms   Terms
	Schemas string
}

// Encode writes the terms as meemoo's dc+schema document: one element per
// term, order preserved. schemas is the relative path from the document to
// the package's schemas/ dir. t must be valid: Terms.Validate is the
// contract, run by the builder before any write, and Encode does not
// repeat it. The document is rendered in memory first, so a refused term
// writes nothing.
func Encode(w io.Writer, t Terms, schemas string) error {
	var buf bytes.Buffer
	if err := termsTemplate.ExecuteTemplate(&buf, "dcschema", termsDoc{t, schemas}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// elementName is the template's one guard: the element name is the only
// thing the template interpolates raw, and only a name the vocabulary
// lists may reach the output. Returning an error aborts the render.
func elementName(element string) (string, error) {
	if _, ok := vocabularyByElement[element]; !ok {
		return "", fmt.Errorf("%q is not in the descriptive vocabulary", element)
	}
	return element, nil
}

// xsiType is the xsi:type the vocabulary declares for the element: how
// the meemoo document types its EDTF dates ("" for untyped elements).
func xsiType(element string) string {
	return vocabularyByElement[element].XSIType
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value; terms carry arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
