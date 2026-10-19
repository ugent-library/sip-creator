package ugent

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

// mods is the MODS 3.7 metadata model of ugent/bibliographic, for a Record.
// mods.xml keeps the producer's identifier, because it is the catalogue
// number operators search by (ADR-0012).
type mods struct{}

// Definition.ValidateSource accepts a supplied document only from a model
// that implements DocumentFormat. If ValidateDocumentRoot's signature
// changed, it would refuse every supplied mods.xml. These assertions turn
// that into a compile error.
var (
	_ build.MetadataModel  = mods{}
	_ build.DocumentFormat = mods{}
)

// The namespace and version the template declares and a supplied document
// must declare. ModelTypeVersion gives the METS dmdSec the same version,
// so a document of another version would contradict it.
const (
	modsNamespace = "http://www.loc.gov/mods/v3"
	modsVersion   = "3.7"
)

func (mods) ValidateType(d sip.Description) error {
	if _, ok := d.(Record); !ok {
		return fmt.Errorf("descriptive metadata is %T, not a MODS record (ugent.Record)", d)
	}
	return nil
}

// ValidateDocumentRoot checks that root is a mods:mods element in the MODS
// v3 namespace, and that it declares the MODS version the package's METS
// declares. It returns an error if either check fails.
func (mods) ValidateDocumentRoot(root xml.StartElement) error {
	if root.Name.Space != modsNamespace || root.Name.Local != "mods" {
		return fmt.Errorf("root element is {%s}%s, expected a mods:mods document in the MODS v3 namespace (%s)", root.Name.Space, root.Name.Local, modsNamespace)
	}
	got, ok := xmldoc.Attr(root, "version")
	switch {
	case !ok:
		return fmt.Errorf("the root declares no version; the package declares MODS %s, so the document must carry version=%q", modsVersion, modsVersion)
	case got != modsVersion:
		return fmt.Errorf("the root declares version=%q, but the package declares MODS %s", got, modsVersion)
	}
	return nil
}

// Encode writes d as a MODS 3.7 document. A record without items gets no
// mods:location at all: an empty holdingSimple would claim the library
// holds no copy. A record without an identifier gets no mods:recordInfo.
func (mods) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := modsTemplate.ExecuteTemplate(&buf, "mods", recordDoc{d.(Record), schemasDir, recordIdentifierSource, modsNamespace, modsVersion}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// ModelType returns MODS.
func (mods) ModelType() string {
	return "MODS"
}

// ModelTypeVersion returns the MODS version the template writes and a
// supplied document must declare.
func (mods) ModelTypeVersion() string {
	return modsVersion
}

// Schemas returns the bundled XSDs the MODS document points at:
// mods-3-7.xsd alone. Its own imports of xml.xsd and xlink.xsd are absolute
// loc.gov URLs, not files next to it, so nothing else needs to ship.
func (mods) Schemas() []build.Schema {
	return build.BundledSchemas("mods-3-7.xsd")
}

// recordIdentifierSource is the source attribute on the
// mods:recordIdentifier written for the record's identifier: UGent
// Library's code in the MARC Code List for Organizations. The Library of
// Congress's MARC to MODS mapping takes the source from 003. The catalogue
// does not deliver 003, and every MMS ID this profile writes is a number in
// UGent Library's catalogue, so the code is fixed here.
const recordIdentifierSource = "BE-GnUNI"

// modsTemplate escapes every value from the record. The only values it
// writes unescaped are constants (the record identifier's source, the
// namespace and the version) and the schemas path the writer supplies.
var modsTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"esc": escapeXML,
}).Parse(`
{{ define "mods" -}}
<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="{{ .Namespace }}" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="{{ .Version }}" xsi:schemaLocation="{{ .Namespace }} {{ .SchemasDir }}/mods-3-7.xsd">
{{- range .Record.OtherIdentifiers }}
  <mods:identifier>{{ esc . }}</mods:identifier>
{{- end }}
{{- range .Record.Titles }}
  <mods:titleInfo{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}>
    <mods:title>{{ esc .Value }}</mods:title>
  </mods:titleInfo>
{{- end }}
{{- range .Record.Contributors }}
  <mods:name>
    <mods:namePart>{{ esc . }}</mods:namePart>
  </mods:name>
{{- end }}
{{- with .Record.Items }}
  <mods:location>
    <mods:holdingSimple>
      {{- range . }}
      <mods:copyInformation>
        {{- with .CallNumber }}
        <mods:shelfLocator>{{ esc . }}</mods:shelfLocator>
        {{- end }}
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
{{- with .Record.Identifier }}
  <mods:recordInfo>
    <mods:recordIdentifier source="{{ $.RecordIdentifierSource }}">{{ esc . }}</mods:recordIdentifier>
  </mods:recordInfo>
{{- end }}
</mods:mods>
{{ end }}
`))

// recordDoc is one MODS document to render: the record, the path of the
// package's schemas/ directory relative to the document, the source
// attribute the record identifier carries, and the namespace and version
// the root declares.
type recordDoc struct {
	Record                 Record
	SchemasDir             string
	RecordIdentifierSource string
	Namespace              string
	Version                string
}
