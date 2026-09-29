package earkmods

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// mods is the encoder for the MODS document as the engine sees it: it
// accepts Record and writes it with Encode; building a Record from flat
// statements is the Definition's NewDescription. It never swaps: mods.xml
// keeps the producer's identifier, the catalogue number the ingesting
// repository indexes and operators search by (ADR-0012).
type mods struct{}

var _ build.DescriptionEncoder = mods{}

func (mods) Check(d sip.Description) error {
	if _, ok := d.(Record); !ok {
		return fmt.Errorf("descriptive metadata is %T, not a MODS record (earkmods.Record)", d)
	}
	return nil
}

// Encode writes d, which is Record since Check ran before anything else, as
// a MODS 3.7 document: one complete element per term, order preserved, then
// the items as one location. schemas is the relative path from the
// document to the package's schemas/ dir. The record must be valid:
// Record.Validate is the contract, run by the engine before any write, and
// Encode does not repeat it. The document is rendered in memory first, so
// a refused key writes nothing.
func (mods) Encode(w io.Writer, d sip.Description, schemas string) error {
	var buf bytes.Buffer
	if err := modsTemplate.ExecuteTemplate(&buf, "mods", recordDoc{d.(Record), schemas}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Schemas lists the bundled XSD file names the MODS document points at:
// mods-3-7.xsd alone. Its own imports of xml.xsd and xlink.xsd are absolute
// loc.gov URLs, not files next to it, so nothing else needs to ship.
func (mods) Schemas() []string {
	return []string{"mods-3-7.xsd"}
}

// modsTemplate renders a record: the "mods" document around one
// sub-template per element the vocabulary names. The element func renders
// one of those sub-templates, so it needs the template it belongs to;
// capturing the local before Parse gives it that without an
// initialization cycle on the package variable. Every value is escaped;
// the only raw interpolation is the element's type attribute, a constant
// from the table. The sub-templates carry no indentation on their first
// line: the caller indents it, and their later lines indent themselves.
var modsTemplate = func() *template.Template {
	var t *template.Template
	t = template.Must(template.New("").Funcs(template.FuncMap{
		"element": func(term sip.Term) (string, error) { return renderElement(t, term) },
		"esc":     escapeXML,
	}).Parse(`
{{ define "mods" -}}
<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="3.7" xsi:schemaLocation="http://www.loc.gov/mods/v3 {{ .Schemas }}/mods-3-7.xsd">
{{- range .Record.Terms }}
  {{ element . }}
{{- end }}
{{- with .Record.Items }}
  <mods:location>
    <mods:holdingSimple>
      {{- range . }}
      <mods:copyInformation>
        <mods:shelfLocator>{{ esc .CallNumber }}</mods:shelfLocator>
        {{- with .Enumeration }}
        <mods:enumerationAndChronology>{{ esc . }}</mods:enumerationAndChronology>
        {{- end }}
        {{- with .Barcode }}
        <mods:itemIdentifier type="barcode">{{ esc . }}</mods:itemIdentifier>
        {{- end }}
      </mods:copyInformation>
      {{- end }}
    </mods:holdingSimple>
  </mods:location>
{{- end }}
</mods:mods>
{{ end }}

{{ define "identifier" -}}
<mods:identifier type="{{ .Row.Type }}">{{ esc .Term.Value }}</mods:identifier>
{{- end }}

{{ define "titleInfo" -}}
<mods:titleInfo{{ with .Term.Lang }} xml:lang="{{ esc . }}"{{ end }}>
    <mods:title>{{ esc .Term.Value }}</mods:title>
  </mods:titleInfo>
{{- end }}
`))
	return t
}()

// recordDoc is one MODS document to render: the record plus the relative
// path from the document's location to the package's bundled schemas/ dir.
type recordDoc struct {
	Record  Record
	Schemas string
}

// elementData is one term with the vocabulary row it renders through; the
// row supplies the element's fixed attribute value.
type elementData struct {
	Term sip.Term
	Row  vocabularyRow
}

// renderElement renders one term as the complete element its key emits,
// through the sub-template named after that element. It is the template's
// one guard: only a key the vocabulary lists reaches the output, and an
// unknown key aborts the render.
func renderElement(t *template.Template, term sip.Term) (string, error) {
	row, ok := vocabularyByKey[term.Key]
	if !ok {
		return "", fmt.Errorf("unknown key %q: not in the MODS vocabulary", term.Key)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, row.Element, elementData{term, row}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value; terms and items carry arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
