// Package mets writes the METS documents of a package: the package METS at
// its root and one METS per representation, following the E-ARK CSIP and
// SIP specifications. Every difference between profiles arrives as data
// on sip.MetsDeclaration.
package mets

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/template"
	"time"
	"uuid"

	"github.com/ugent-library/sip-creator/sip"
)

// Schemas are the bundled XSD file names the METS documents point at with
// xsi:schemaLocation: METS 1.12, xlink and the two E-ARK extension schemas.
// Every package holds them in its schemas/ directory.
var Schemas = []string{"mets1_12.xsd", "xlink.xsd", "DILCISExtensionMETS.xsd", "DILCISExtensionSIPMETS.xsd"}

// mdTypes is the METS 1.12 MDTYPE vocabulary, the closed list of values
// mets1_12.xsd allows for mdRef/@MDTYPE.
var mdTypes = []string{
	"MARC", "MODS", "EAD", "DC", "NISOIMG", "LC-AV", "VRA", "TEIHDR", "DDI",
	"FGDC", "LOM", "PREMIS", "PREMIS:OBJECT", "PREMIS:AGENT", "PREMIS:RIGHTS",
	"PREMIS:EVENT", "TEXTMD", "METSRIGHTS", "ISO 19115:2003 NAP", "EAC-CPF",
	"LIDO", "OTHER",
}

// MDType returns the MDTYPE and OTHERMDTYPE values that METS records for a
// metadata format. A format the MDTYPE vocabulary lists, with the same
// spelling, is its own MDTYPE and has no OTHERMDTYPE. Any other format is
// MDTYPE OTHER, with the format as OTHERMDTYPE.
func MDType(format string) (mdType, otherMDType string) {
	if slices.Contains(mdTypes, format) {
		return format, ""
	}
	return "OTHER", format
}

// identifier mints a fresh uuid-<uuid> METS ID.
func identifier() string {
	return fmt.Sprintf("uuid-%s", uuid.NewV4().String())
}

var funcs = template.FuncMap{
	"identifier": identifier,
	"href":       href,
	"now": func() string {
		return time.Now().Format(time.RFC3339Nano)
	},
	"joinIdentifiers": func(files []*sip.File) string {
		var ids []string
		for _, f := range files {
			ids = append(ids, f.Identifier)
		}
		return strings.Join(ids, " ")
	},
	"esc": escapeXML,
}

// href returns a path, relative to the METS document, as a URI reference.
// It percent-encodes every byte except the unreserved characters of RFC 3986
// and the slashes between segments, so "R&D 1+2.tif" becomes
// R%26D%201%2B2.tif. Readers decode an href before they open the file.
// commons-ip, and RODA through it, decodes with java.net.URLDecoder, which
// also reads a raw "+" as a space and rejects a "%" that is not followed by
// two hex digits. Because "+" and "%" are encoded too, both kinds of decoder
// get the file's name back.
func href(path string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for _, c := range []byte(path) {
		if isUnreserved(c) || c == '/' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

// isUnreserved reports whether c is one of RFC 3986's unreserved
// characters, which a URI carries as they are.
func isUnreserved(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// escapeXML returns s escaped for use as XML character data or as a quoted
// attribute value. Labels, representation names and agent names come from
// producers and operators, and a label such as "R&D scans" is ordinary.
func escapeXML(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s)) // never fails on a bytes.Buffer
	return b.String()
}

// templates hold the package and representation METS documents. They
// escape every value they read from the package graph and write every path
// as an href. Only the IDs and timestamps they mint themselves are written
// as they are. An ID that two elements share ($fileGrpID, $docGrpID,
// $SCHEMAID, $DOCID) is minted once, up front, so a fileGrp and the
// structMap fptr that points at it carry the same value.
var templates = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "representation" -}}
{{ $fileGrpID := identifier -}}
{{ $docGrpID := identifier -}}
<?xml version='1.0' encoding='UTF-8'?>
<mets xmlns="http://www.loc.gov/METS/"
  xmlns:csip="https://DILCIS.eu/XML/METS/CSIPExtensionMETS"
  xmlns:sip="https://DILCIS.eu/XML/METS/SIPExtensionMETS"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:xlink="http://www.w3.org/1999/xlink"
  OBJID="{{ esc .Name }}"
  TYPE="{{ esc .Declaration.Type }}"
  {{- with .Declaration.OtherType }}
  csip:OTHERTYPE="{{ esc . }}"
  {{- end }}
  LABEL="{{ esc .Label }}"
  PROFILE="{{ esc .Declaration.ProfileURL }}"
  csip:CONTENTINFORMATIONTYPE="{{ esc .Declaration.ContentInformationType }}"
  {{- with .Declaration.OtherContentInformationType }}
  csip:OTHERCONTENTINFORMATIONTYPE="{{ esc . }}"
  {{- end }}
  xsi:schemaLocation="http://www.loc.gov/METS/ ../../schemas/mets1_12.xsd http://www.w3.org/1999/xlink ../../schemas/xlink.xsd https://DILCIS.eu/XML/METS/CSIPExtensionMETS ../../schemas/DILCISExtensionMETS.xsd https://DILCIS.eu/XML/METS/SIPExtensionMETS ../../schemas/DILCISExtensionSIPMETS.xsd">

  <metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP" />
  {{- with .DescriptionFile }}

  <dmdSec ID="{{ esc .Identifier }}" CREATED="{{ now }}" STATUS="CURRENT">
    <mdRef LOCTYPE="URL" MDTYPE="{{ esc .MDType }}"{{ with .OtherMDType }} OTHERMDTYPE="{{ esc . }}"{{ end }}{{ with .MDTypeVersion }} MDTYPEVERSION="{{ esc . }}"{{ end }} xlink:type="simple" xlink:href="{{ href .Path }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5" />
  </dmdSec>
  {{- end }}
  {{- with .PremisFiles }}

  <amdSec>
    {{- range . }}
    <digiprovMD ID="{{ esc .Identifier }}" STATUS="CURRENT">
      <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ href .Path }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5" />
    </digiprovMD>
    {{- end }}
  </amdSec>
  {{- end }}

  <fileSec ID="{{ identifier }}">
    <fileGrp USE="data" ID="{{ $fileGrpID }}">
      {{- range .Files }}
      <file ID="{{ esc .Identifier }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5">
        <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ href .Path }}"/>
      </file>
      {{- end }}
    </fileGrp>
    {{- with .DocumentationFiles }}
    <fileGrp USE="Documentation" ID="{{ $docGrpID }}">
      {{- range . }}
      <file ID="{{ esc .Identifier }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5">
        <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ href .Path }}"/>
      </file>
      {{- end }}
    </fileGrp>
    {{- end }}
  </fileSec>

  <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
    <div ID="{{ identifier }}" LABEL="{{ esc .Name }}">
      <div ID="{{ identifier }}" LABEL="Metadata"{{ with .DescriptionFile }} DMDID="{{ esc .Identifier }}"{{ end }}{{ with .PremisFiles }} ADMID="{{ joinIdentifiers . | esc }}"{{ end }}/>
      <div ID="{{ identifier }}" LABEL="Data">
        <fptr FILEID="{{ $fileGrpID }}" />
      </div>
      {{- with .DocumentationFiles }}
      <div ID="{{ identifier }}" LABEL="Documentation">
        <fptr FILEID="{{ $docGrpID }}"/>
      </div>
      {{- end }}
    </div>
  </structMap>
</mets>
{{ end}}
{{ define "package" -}}
{{ $OBJID := .Identifier -}}
{{ $SCHEMAID := identifier -}}
{{ $DOCID := identifier -}}
<?xml version='1.0' encoding='UTF-8'?>
<mets xmlns="http://www.loc.gov/METS/"
  xmlns:csip="https://DILCIS.eu/XML/METS/CSIPExtensionMETS"
  xmlns:sip="https://DILCIS.eu/XML/METS/SIPExtensionMETS"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:xlink="http://www.w3.org/1999/xlink"
  OBJID="{{ esc $OBJID }}"
  LABEL=""
  TYPE="{{ esc .Declaration.Type }}"
  {{- with .Declaration.OtherType }}
  csip:OTHERTYPE="{{ esc . }}"
  {{- end }}
  PROFILE="{{ esc .Declaration.ProfileURL }}"
  csip:CONTENTINFORMATIONTYPE="{{ esc .Declaration.ContentInformationType }}"
  {{- with .Declaration.OtherContentInformationType }}
  csip:OTHERCONTENTINFORMATIONTYPE="{{ esc . }}"
  {{- end }}
  xsi:schemaLocation="http://www.loc.gov/METS/ schemas/mets1_12.xsd http://www.w3.org/1999/xlink schemas/xlink.xsd https://DILCIS.eu/XML/METS/CSIPExtensionMETS schemas/DILCISExtensionMETS.xsd https://DILCIS.eu/XML/METS/SIPExtensionMETS schemas/DILCISExtensionSIPMETS.xsd">

  <metsHdr CREATEDATE="{{ now }}"{{ with .Declaration.RecordStatus }} RECORDSTATUS="{{ print . | esc }}"{{ end }} csip:OAISPACKAGETYPE="SIP">
    {{- range .Declaration.Agents }}
    <agent ROLE="{{ esc .Role }}"{{ if .OtherRole }} OTHERROLE="{{ esc .OtherRole }}"{{ end }} TYPE="{{ esc .Type }}"{{ if .OtherType }} OTHERTYPE="{{ esc .OtherType }}"{{ end }}>
      <name>{{ esc .Name }}</name>
      {{- if .Note }}
      <note csip:NOTETYPE="{{ esc .NoteType }}">{{ esc .Note }}</note>
      {{- end }}
    </agent>
    {{- end }}
  </metsHdr>

  <!-- ref to descriptive metadata about IE -->
  {{- range .DescriptiveFiles }}
  <dmdSec ID="{{ esc .Identifier }}" CREATED="{{ now }}" STATUS="CURRENT">
    <mdRef LOCTYPE="URL" MDTYPE="{{ esc .MDType }}"{{ with .OtherMDType }} OTHERMDTYPE="{{ esc . }}"{{ end }}{{ with .MDTypeVersion }} MDTYPEVERSION="{{ esc . }}"{{ end }} xlink:type="simple" xlink:href="{{ href .Path }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5" />
  </dmdSec>
  {{- end }}
  {{- with .PremisFiles }}

  <!-- ref to the PREMIS metadata about IE/subIE(s)/package -->
  <amdSec>
    {{- range . }}
    <digiprovMD ID="{{ esc .Identifier }}" STATUS="CURRENT">
      <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ href .Path }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5" />
    </digiprovMD>
    {{- end }}
  </amdSec>
  {{- end }}

  <!-- file section -->
  <fileSec ID="{{ identifier }}">
    <fileGrp ID="{{ $SCHEMAID }}" USE="Schemas">
      {{- range .SchemaFiles }}
      <file ID="{{ esc .Identifier }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5">
        <FLocat xlink:type="simple" xlink:href="{{ href .Path }}" LOCTYPE="URL"/>
      </file>
      {{- end }}
    </fileGrp>
    {{- with .DocumentationFiles }}
    <fileGrp ID="{{ $DOCID }}" USE="Documentation">
      {{- range . }}
      <file ID="{{ esc .Identifier }}" MIMETYPE="{{ esc .Mime }}" SIZE="{{ esc .Size }}" CREATED="{{ esc .Created }}" CHECKSUM="{{ esc .Checksum }}" CHECKSUMTYPE="MD5">
        <FLocat xlink:type="simple" xlink:href="{{ href .Path }}" LOCTYPE="URL"/>
      </file>
      {{- end }}
    </fileGrp>
    {{- end }}
    {{- range .Root.Representations }}
    <fileGrp ID="{{ esc .Identifier }}" USE="Representations/{{ esc .Name }}">
      <file ID="{{ esc .MetsFile.Identifier }}" MIMETYPE="{{ esc .MetsFile.Mime }}" SIZE="{{ esc .MetsFile.Size }}" CREATED="{{ esc .MetsFile.Created }}" CHECKSUM="{{ esc .MetsFile.Checksum }}" CHECKSUMTYPE="MD5">
        <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ href .MetsFile.Path }}"/>
      </file>
    </fileGrp>
    {{- end }}
  </fileSec>

  <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
    <div ID="{{ identifier }}" LABEL="{{ esc $OBJID }}">
      <div ID="{{ identifier }}" LABEL="Metadata" DMDID="{{ esc .Root.DescriptionFile.Identifier }}"{{ with .PremisFiles }} ADMID="{{ joinIdentifiers . | esc }}"{{ end }}/>
      <div ID="{{ identifier }}" LABEL="Schemas">
        <fptr FILEID="{{ $SCHEMAID }}"/>
      </div>
      {{- with .DocumentationFiles }}
      <div ID="{{ identifier }}" LABEL="Documentation">
        <fptr FILEID="{{ $DOCID }}"/>
      </div>
      {{- end }}
      {{- range .Root.Representations }}
      <div ID="{{ identifier }}" LABEL="Representations/{{ esc .Name }}">
        <mptr xlink:type="simple" xlink:href="{{ href .MetsFile.Path }}" LOCTYPE="URL" xlink:title="{{ esc .Identifier }}" />
      </div>
      {{- end }}
    </div>
  </structMap>
</mets>
{{ end }}
`))

// EncodeRepresentation writes the METS document of r, with the profile
// values r.Declaration holds.
func EncodeRepresentation(w io.Writer, r *sip.Representation) error {
	return templates.ExecuteTemplate(w, "representation", r)
}

// EncodePackage writes the package METS document.
func EncodePackage(w io.Writer, p *sip.Package) error {
	return templates.ExecuteTemplate(w, "package", p)
}
