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

// simpledc is the encoder for the simpledc document as the engine sees it:
// it accepts Terms and writes them with Encode; building Terms from flat
// statements is the Definition's NewDescription. It never swaps: dc.xml
// keeps the producer's identifier, because CSIP has no rule tying it to
// the package identifier and the ingesting catalogue indexes dc.xml, so
// operators find the package by the identifier they know (ADR-0012).
type simpledc struct{}

var _ build.DescriptionEncoder = simpledc{}

func (simpledc) Check(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Simple Dublin Core terms (eark.Terms)", d)
	}
	return nil
}

// Encode writes d, which is Terms since Check ran before anything else, as
// a Simple Dublin Core document (the dc_SimpleDC20021212 shape RODA
// renders and indexes natively): one unqualified element per term, order
// preserved, language tags omitted. schemas is the relative path from the
// document to the package's schemas/ dir. The terms must be valid:
// Terms.Validate is the contract, run by the engine before any write, and
// Encode does not repeat it. The document is rendered in memory first, so
// a refused term writes nothing.
func (simpledc) Encode(w io.Writer, d sip.Description, schemas string) error {
	var buf bytes.Buffer
	if err := simpledcTemplate.ExecuteTemplate(&buf, "simpledc", termsDoc{d.(Terms), schemas}); err != nil {
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

// The template interpolates element names from data. Every value is
// escaped, and the element name comes from el, which admits only a key
// naming one of the fifteen.
var simpledcTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"el":  elementName,
	"esc": escapeXML,
}).Parse(`
{{ define "simpledc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="{{ .Schemas }}/dc.xsd">
{{- range .Terms }}
  <{{ el .Key }}>{{ esc .Value }}</{{ el .Key }}>
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
