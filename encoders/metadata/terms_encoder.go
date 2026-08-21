package metadata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"
)

// The terms templates interpolate element names from data, which the
// Description templates never did — so encoding validates first (element
// names come from the closed vocabularies) and every value is escaped.
var termsTemplates = template.Must(template.New("").Funcs(template.FuncMap{
	"esc":     escapeXML,
	"xsitype": xsiType,
}).Parse(`
{{ define "terms" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/1.2/basic"
	xmlns:dcterms="http://purl.org/dc/terms/"
	xmlns:xs="http://www.w3.org/2001/XMLSchema/"
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xmlns:edtf="http://id.loc.gov/datatypes/edtf/"
	xmlns:schema="https://schema.org/"
	xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic {{ .Schemas }}/descriptive_basic.xsd">
{{ range .Terms }}
	<{{ .Element }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ with xsitype .Element }} xsi:type="{{ . }}"{{ end }}>{{ esc .Value }}</{{ .Element }}>
{{- end }}
</metadata>
{{ end }}
{{ define "dc-terms" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .Schemas }}/dc.xsd">
{{ range .Terms }}
	<{{ .Element }}>{{ esc .Value }}</{{ .Element }}>
{{- end }}
</simpledc>
{{ end }}
`))

// termsDoc is one descriptive document to render: the terms plus the
// relative path from the document's location to the package's bundled
// schemas/ dir — the writer knows where the document lands, the template
// only carries the hint.
type termsDoc struct {
	Terms   Terms
	Schemas string
}

// PackageSchemas is the schemas/ dir as seen from a package-level
// descriptive document (metadata/descriptive/*.xml).
const PackageSchemas = "../../schemas"

// EncodeTerms writes the terms as the meemoo descriptive document — the
// same document shape as the retired dc+schema define, one element per
// term, order preserved. schemas is the relative path from the document to
// the package's schemas/ dir.
func EncodeTerms(w io.Writer, t Terms, schemas string) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsTemplates.ExecuteTemplate(w, "terms", termsDoc{t, schemas})
}

// EncodeDCTerms writes the terms as a simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively), dumbing
// qualified terms down to their Simple DC parent element. schemas is the
// relative path from the document to the package's schemas/ dir.
func EncodeDCTerms(w io.Writer, t Terms, schemas string) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsTemplates.ExecuteTemplate(w, "dc-terms", termsDoc{dumbDown(t), schemas})
}

// dumbDown maps each term onto the vocabulary row's Simple DC parent (the
// dumb-down principle); rows with no parent are omitted. Callers validate
// first, so every element has a row.
func dumbDown(t Terms) Terms {
	out := make(Terms, 0, len(t))
	for _, term := range t {
		simple := vocabularyByElement[term.Element].SimpleDC
		if simple == "" {
			continue
		}
		out = append(out, Term{Element: simple, Value: term.Value})
	}
	return out
}

// xsiType is the xsi:type the vocabulary declares for the element — how
// the meemoo document marks its EDTF dates ("" for untyped elements).
func xsiType(element string) string {
	return vocabularyByElement[element].XSIType
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value. The Description templates never needed this — their
// fixture data was controlled — but terms carry arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
