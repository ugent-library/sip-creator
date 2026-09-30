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

// dcschema is the encoder for the dc+schema document as the engine sees
// it: it accepts Terms, writes them with Encode, and swaps the entity
// identifier in. It is no DescriptiveDocumentChecker: meemoo's document must carry
// the entity identifier the build mints, which Swap writes into terms, so
// a supplied document has no place here and the engine refuses one.
type dcschema struct{}

// IdentifierSwapper is optional to the engine, so a drift in Swap's
// signature would fail silently; these assertions make it a build error.
var (
	_ build.DescriptionEncoder = dcschema{}
	_ build.IdentifierSwapper  = dcschema{}
)

func (dcschema) Check(d sip.Description) error {
	if _, ok := d.(Terms); !ok {
		return fmt.Errorf("descriptive metadata is %T, not meemoo dc+schema terms (meemoo.Terms)", d)
	}
	return nil
}

// Encode writes d, which is Terms since Check ran before anything else, as
// meemoo's dc+schema document: one element per term, order preserved.
// schemas is the relative path from the document to the package's
// schemas/ dir. The terms must be valid: Terms.Validate is the contract,
// run by the engine before any write, and Encode does not repeat it. The
// document is rendered in memory first, so a refused term writes nothing.
func (dcschema) Encode(w io.Writer, d sip.Description, schemas string) error {
	var buf bytes.Buffer
	if err := termsTemplate.ExecuteTemplate(&buf, "dcschema", termsDoc{d.(Terms), schemas}); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// Swap replaces the identifier in d with id and returns the producer's
// identifier it replaced. meemoo SIP 1.2 links dc+schema.xml to the PREMIS
// object by a shared UUID: the document carries the entity identifier, and
// the producer's own identifier travels as a MEEMOO-LOCAL-ID object
// identifier. The terms hold one identifier slot, so the producer's value
// is read before the swap overwrites it.
func (dcschema) Swap(d sip.Description, id string) string {
	terms := d.(Terms) // Check ran before anything else
	local := terms.localIdentifier()
	terms.setObjectIdentifier(id)
	return local
}

// Schemas lists the bundled XSD file names the dc+schema document points
// at, plus what those import by relative path: meemoo's
// descriptive_basic.xsd imports the Dublin Core, DCMI type, EDTF and
// schema.org schemas and xml.xsd.
func (dcschema) Schemas() []string {
	return []string{"descriptive_basic.xsd", "dc.xsd", "dcterms.xsd", "dcmitype.xsd", "edtf.xsd", "schema.xsd", "xml.xsd"}
}

// The template interpolates element names from data. Every value is
// escaped, and the element name comes from el, which maps a key to the
// element the vocabulary lists for it and admits nothing else.
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
  <{{ el .Key }}{{ with .Lang }} xml:lang="{{ esc . }}"{{ end }}{{ with xsitype .Key }} xsi:type="{{ . }}"{{ end }}>{{ esc .Value }}</{{ el .Key }}>
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

// elementName is the element a key emits, and the template's one guard:
// the element name is the only thing the template interpolates raw, and
// only a name the vocabulary lists may reach the output. Returning an
// error aborts the render.
func elementName(key string) (string, error) {
	row, ok := vocabularyByKey[key]
	if !ok {
		return "", fmt.Errorf("unknown key %q: not in the descriptive vocabulary", key)
	}
	return row.Element, nil
}

// xsiType is the xsi:type the vocabulary declares for the key's element:
// how the meemoo document types its EDTF dates ("" for untyped elements).
func xsiType(key string) string {
	return vocabularyByKey[key].XSIType
}

// escapeXML makes a data value safe as XML character data or a quoted
// attribute value; terms carry arbitrary operator input.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}
