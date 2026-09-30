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
// accepts Record and writes it with Encode. It never swaps: mods.xml
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
// a MODS 3.7 document: the identifier when the record states one, the
// titles in the order given, then the items as one location. schemas is
// the relative path from the document to the package's schemas/ dir. The
// record must be valid: Record.Validate is the contract, run by the engine
// before any write, and Encode does not repeat it. The document is
// rendered in memory first, so a failed render writes nothing.
func (mods) Encode(w io.Writer, d sip.Description, schemas string) error {
	var buf bytes.Buffer
	if err := modsTemplate.ExecuteTemplate(&buf, "mods", recordDoc{d.(Record), schemas, mmsIDType}); err != nil {
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

// mmsIDType is the type attribute on the mods:identifier the record's
// identifier emits: the catalogue number, an Alma MMS ID at UGent Library.
// MODS leaves the type vocabulary open; "local" is the value its own list
// suggests for a system-internal identifier. The owner of the repository
// side settles the final value (descriptive-model plan, open question),
// and this constant is the one place it changes.
const mmsIDType = "local"

// modsTemplate renders a record field by field: the identifier when the
// record states one, one titleInfo per title, and the items as one
// location/holdingSimple with one copyInformation each, omitted when
// there are none. Every value is escaped; the only raw interpolations are
// the identifier's type, a constant, and the schemas path the writer
// supplies.
var modsTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"esc": escapeXML,
}).Parse(`
{{ define "mods" -}}
<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="3.7" xsi:schemaLocation="http://www.loc.gov/mods/v3 {{ .Schemas }}/mods-3-7.xsd">
{{- with .Record.Identifier }}
  <mods:identifier type="{{ $.IdentifierType }}">{{ esc . }}</mods:identifier>
{{- end }}
{{- range .Record.Titles }}
  <mods:titleInfo{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}>
    <mods:title>{{ esc .Value }}</mods:title>
  </mods:titleInfo>
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
`))

// recordDoc is one MODS document to render: the record, the relative path
// from the document's location to the package's bundled schemas/ dir, and
// the type attribute the identifier carries.
type recordDoc struct {
	Record         Record
	Schemas        string
	IdentifierType string
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value; the record's values are arbitrary producer input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
