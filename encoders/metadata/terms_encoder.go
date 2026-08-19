package metadata

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"text/template"
)

// The terms templates interpolate element names from data, which the
// Description templates never do — so encoding validates first (element
// names come from the closed vocabularies) and every value is escaped.
var termsMD = template.Must(template.New("").Funcs(template.FuncMap{
	"esc":  escapeXML,
	"edtf": isEDTFTyped,
}).Parse(`
{{ define "terms" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/1.2/basic"
	xmlns:dcterms="http://purl.org/dc/terms/"
	xmlns:xs="http://www.w3.org/2001/XMLSchema/"
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xmlns:edtf="http://id.loc.gov/datatypes/edtf/"
	xmlns:schema="https://schema.org/"
	xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic ../../schemas/descriptive_basic.xsd">
{{ range . }}
	<{{ .Element }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ if edtf .Element }} xsi:type="edtf:EDTF-level1"{{ end }}>{{ esc .Value }}</{{ .Element }}>
{{- end }}
</metadata>
{{ end }}
{{ define "dc-terms" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../schemas/dc.xsd">
{{ range . }}
	<{{ .Element }}>{{ esc .Value }}</{{ .Element }}>
{{- end }}
</simpledc>
{{ end }}
`))

// Encode writes the terms as the meemoo descriptive document — the same
// document shape as the dc+schema define, one element per term, order
// preserved. It implements sip.Descriptive; the family seam selects
// EncodeDCTerms for the eark shape instead.
func (t Terms) Encode(w io.Writer) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsMD.ExecuteTemplate(w, "terms", t)
}

// EncodeDCTerms writes the terms as a simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively), dumbing
// qualified terms down to their Simple DC parent element.
func EncodeDCTerms(w io.Writer, t Terms) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsMD.ExecuteTemplate(w, "dc-terms", dumbDown(t))
}

// simpleDC maps each Dublin Core term onto the Simple DC element it dumbs
// down to: the fifteen elements onto themselves, refinements onto their
// DCMI parent. Source: the "Subproperty Of" relations in the DCMI Metadata
// Terms spec (dublincore.org/specifications/dublin-core/dcmi-terms/),
// applied per DCMI's dumb-down principle. Terms with no declared parent
// among the fifteen (rightsHolder, provenance, audience refinements, all
// schema:*) are omitted — inventing a mapping would assert semantics DCMI
// doesn't.
var simpleDC = map[string]string{
	"title": "title", "creator": "creator", "subject": "subject",
	"description": "description", "publisher": "publisher",
	"contributor": "contributor", "date": "date", "type": "type",
	"format": "format", "identifier": "identifier", "source": "source",
	"language": "language", "relation": "relation", "coverage": "coverage",
	"rights": "rights",

	"alternative": "title",
	"abstract":    "description", "tableOfContents": "description",
	"created": "date", "valid": "date", "available": "date",
	"issued": "date", "modified": "date", "dateAccepted": "date",
	"dateCopyrighted": "date", "dateSubmitted": "date",
	"extent": "format", "medium": "format",
	"isVersionOf": "relation", "hasVersion": "relation",
	"isReplacedBy": "relation", "replaces": "relation",
	"isRequiredBy": "relation", "requires": "relation",
	"isPartOf": "relation", "hasPart": "relation",
	"isReferencedBy": "relation", "references": "relation",
	"isFormatOf": "relation", "hasFormat": "relation",
	"conformsTo": "relation",
	"spatial":    "coverage", "temporal": "coverage",
	"accessRights": "rights", "license": "rights",
	"bibliographicCitation": "identifier",
}

func dumbDown(t Terms) Terms {
	out := make(Terms, 0, len(t))
	for _, term := range t {
		base, ok := strings.CutPrefix(term.Element, "dcterms:")
		if !ok {
			continue // schema:* has no Simple DC home
		}
		simple, ok := simpleDC[base]
		if !ok {
			continue // no DCMI dumb-down relationship
		}
		out = append(out, Term{Element: simple, Value: term.Value})
	}
	return out
}

// isEDTFTyped marks the terms the meemoo document types as EDTF dates —
// the same two the dc+schema define types.
func isEDTFTyped(element string) bool {
	return element == "dcterms:created" || element == "dcterms:issued"
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value. The Description templates never needed this — their
// fixture data was controlled — but terms carry arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
