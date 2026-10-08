package meemoo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// dcschema is Meemoo's dc+schema.org metadata model, built from Terms. It
// accepts no supplied dc+schema document, because Meemoo's document must
// carry the entity identifier, which Swap writes into the terms
// (ADR-0021).
type dcschema struct{}

// Builder.Build applies Swap only when the model implements
// IdentifierSwapper. If Swap's signature changed, Builder.Build would skip
// Swap without an error, and dc+schema.xml would keep the producer's
// identifier. These assertions turn that into a compile error.
var (
	_ build.MetadataModel     = dcschema{}
	_ build.IdentifierSwapper = dcschema{}
)

func (dcschema) ValidateType(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not Meemoo dc+schema terms (meemoo.Terms)", d)
	}
	return nil
}

// Encode writes d as Meemoo's dc+schema document: one element per term, in
// the order of the terms.
func (dcschema) Encode(w io.Writer, d sip.Description, schemasDir string) error {
	var buf bytes.Buffer
	if err := termsTemplate.ExecuteTemplate(&buf, "dcschema", termsDoc{d.(Terms), schemasDir}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Swap replaces the identifier in d with id and returns the producer's
// identifier it replaced. Meemoo SIP 1.2 links dc+schema.xml to the PREMIS
// object by a shared UUID: the document carries the entity identifier, and
// the producer's own identifier travels as a MEEMOO-LOCAL-ID object
// identifier.
func (dcschema) Swap(d sip.Description, id string) string {
	terms := d.(Terms) // Definition.ValidateSource has checked the type
	local := terms.localIdentifier()
	terms.setObjectIdentifier(id)
	return local
}

// ModelType returns DC, because Meemoo's model is built on Dublin Core
// terms.
func (dcschema) ModelType() string {
	return "DC"
}

// ModelTypeVersion returns an empty string, because the model is Meemoo's
// own and no Dublin Core version names it.
func (dcschema) ModelTypeVersion() string {
	return ""
}

// Schemas returns the bundled XSDs the dc+schema document points at, plus
// what those import by relative path: Meemoo's descriptive_basic.xsd
// imports dc.xsd, dcterms.xsd, edtf.xsd and schema.xsd, and dcterms.xsd
// imports dcmitype.xsd. descriptive_basic.xsd imports xml.xsd from the file
// next to it, so xml.xsd ships too. The other XSDs import xml.xsd from its
// W3C URL, not by relative path.
func (dcschema) Schemas() []build.Schema {
	return build.BundledSchemas("descriptive_basic.xsd", "dc.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd")
}

// termsTemplate escapes every value. Element names come from elementName,
// which lets only the names the table lists through.
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
  xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic {{ .SchemasDir }}/descriptive_basic.xsd">
{{- range .Terms }}
  <{{ el .Key }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ with xsitype .Key }} xsi:type="{{ . }}"{{ end }}>{{ esc .Value }}</{{ el .Key }}>
{{- end }}
</metadata>
{{ end }}
`))

// termsDoc is one descriptive document to render: the terms plus the path
// of the package's schemas/ directory relative to the document.
type termsDoc struct {
	Terms      Terms
	SchemasDir string
}

// elementName returns element if the table lists it. It returns an error
// otherwise, which aborts the render, because the template writes the name
// unescaped.
func elementName(element string) (string, error) {
	if _, ok := elementsByName[element]; !ok {
		return "", fmt.Errorf("unknown element %q: not an element of Meemoo's basic content profile", element)
	}
	return element, nil
}

// xsiType returns the xsi:type the table declares for the element, which
// is how the Meemoo document types its EDTF dates. It returns an empty
// string for an untyped element.
func xsiType(element string) string {
	return elementsByName[element].XSIType
}

// escapeXML returns s escaped for use as XML character data or as a quoted
// attribute value.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
