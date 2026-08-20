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
// Description templates never did — so encoding validates first (element
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
	xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic {{ .Schemas }}/descriptive_basic.xsd">
{{ range .Terms }}
	<{{ .Element }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ if edtf .Element }} xsi:type="edtf:EDTF-level1"{{ end }}>{{ esc .Value }}</{{ .Element }}>
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

// Encode writes the terms as a package-level meemoo descriptive document.
// It satisfies sip.Descriptive; the writer's family seam calls EncodeTerms
// with an explicit schema location instead, because only the writer knows
// where a document lands.
func (t Terms) Encode(w io.Writer) error {
	return EncodeTerms(w, t, PackageSchemas)
}

// EncodeTerms writes the terms as the meemoo descriptive document — the
// same document shape as the retired dc+schema define, one element per
// term, order preserved. schemas is the relative path from the document to
// the package's schemas/ dir.
func EncodeTerms(w io.Writer, t Terms, schemas string) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsMD.ExecuteTemplate(w, "terms", termsDoc{t, schemas})
}

// EncodeDCTerms writes the terms as a simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively), dumbing
// qualified terms down to their Simple DC parent element. schemas is the
// relative path from the document to the package's schemas/ dir.
func EncodeDCTerms(w io.Writer, t Terms, schemas string) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("descriptive terms: %w", err)
	}
	return termsMD.ExecuteTemplate(w, "dc-terms", termsDoc{dumbDown(t), schemas})
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
// the same two the dc+schema define typed.
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
