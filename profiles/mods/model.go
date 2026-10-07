package mods

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/encoders/xmldoc"
	"github.com/ugent-library/sip-creator/sip"
)

// Model is the MODS 3.7 metadata model, for a profile's definition.
var Model build.MetadataModel = model{}

// model is the MODS 3.7 metadata model: it accepts Record, writes it as a
// MODS document with Encode, and says which supplied document is one of its
// own. It never swaps: mods.xml keeps the
// producer's identifier, the catalogue number the ingesting repository
// indexes and operators search by (ADR-0012).
type model struct{}

// DocumentFormat is optional to the engine, so a drift in
// ValidateDocumentRoot's signature would fail silently; the assertion
// makes it a build error.
var _ build.DocumentFormat = model{}

// The namespace and version the template declares and a supplied document
// must declare. ModelTypeVersion gives the METS dmdSec the same version,
// so a document of another version would contradict it.
const (
	namespace = "http://www.loc.gov/mods/v3"
	version   = "3.7"
)

func (model) ValidateType(d sip.Description) error {
	if _, ok := d.(Record); !ok {
		return fmt.Errorf("descriptive metadata is %T, not a MODS record (mods.Record)", d)
	}
	return nil
}

// ValidateDocumentRoot returns why root is not a mods:mods element in the
// MODS v3 namespace declaring the version the package's METS declares.
func (model) ValidateDocumentRoot(root xml.StartElement) error {
	if root.Name.Space != namespace || root.Name.Local != "mods" {
		return fmt.Errorf("root element is {%s}%s, expected a mods:mods document in the MODS v3 namespace (%s)", root.Name.Space, root.Name.Local, namespace)
	}
	got, ok := xmldoc.Attr(root, "version")
	switch {
	case !ok:
		return fmt.Errorf("the root declares no version; the package declares MODS %s, so the document must carry version=%q", version, version)
	case got != version:
		return fmt.Errorf("the root declares version=%q, but the package declares MODS %s", got, version)
	}
	return nil
}

// Encode writes d as a MODS 3.7 document: the identifier when the record
// states one, one titleInfo per title in the order given, then the items
// as one location/holdingSimple with one copyInformation each, omitted
// when there are none.
func (model) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := modsTemplate.ExecuteTemplate(&buf, "mods", recordDoc{d.(Record), schemasDir, localIdentifierType, namespace, version}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// ModelType types the document as MODS.
func (model) ModelType() string {
	return "MODS"
}

// ModelTypeVersion is the MODS version the template writes and a supplied
// document must declare.
func (model) ModelTypeVersion() string {
	return version
}

// Schemas returns the bundled XSDs the MODS document points at:
// mods-3-7.xsd alone. Its own imports of xml.xsd and xlink.xsd are absolute
// loc.gov URLs, not files next to it, so nothing else needs to ship.
func (model) Schemas() []build.Schema {
	return build.BundledSchemas("mods-3-7.xsd")
}

// localIdentifierType is the type attribute on the mods:identifier the
// record's identifier emits. MODS leaves the type vocabulary open; "local"
// is the value its own list suggests for an identifier local to the
// describing institution's system, such as a record number in a
// library catalogue. With the mods-coverage plan the type becomes a choice
// per record from a closed set; until then this constant is the one place
// it lives.
const localIdentifierType = "local"

// modsTemplate renders a record field by field. Every value is escaped;
// the only raw interpolations are the identifier's type, the namespace
// and version, all constants, and the schemas path the writer supplies.
var modsTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"esc": escapeXML,
}).Parse(`
{{ define "mods" -}}
<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="{{ .Namespace }}" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="{{ .Version }}" xsi:schemaLocation="{{ .Namespace }} {{ .SchemasDir }}/mods-3-7.xsd">
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

// recordDoc is one MODS document to render: the record, the path of the
// package's schemas/ directory relative to the document, the type
// attribute the identifier carries, and the namespace and version the
// root declares.
type recordDoc struct {
	Record         Record
	SchemasDir     string
	IdentifierType string
	Namespace      string
	Version        string
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value; the record's values are arbitrary producer input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
