package mets

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/ugent-library/sip-creator/sip"
)

func identifier() string {
	return fmt.Sprintf("uuid-%s", uuid.New().String())
}

var funcs = template.FuncMap{
	"identifier": identifier,
	"encode": func(arg string) string {
		return url.QueryEscape(arg)
	},
	"now": func() string {
		return time.Now().Format(time.RFC3339Nano)
	},
	"joinIdentifiers": func(files []*sip.File) string {
		var tmp []string
		for _, f := range files {
			tmp = append(tmp, f.Identifier)
		}

		return strings.Join(tmp[:], " ")
	},
}

var dc = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "representation" -}}
<?xml version='1.0' encoding='UTF-8'?>
<mets xmlns="http://www.loc.gov/METS/" 
	xmlns:csip="https://DILCIS.eu/XML/METS/CSIPExtensionMETS" 
	xmlns:sip="https://DILCIS.eu/XML/METS/SIPExtensionMETS" 
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
	xmlns:xlink="http://www.w3.org/1999/xlink" 
	OBJID="{{ .Name }}"
	TYPE="{{ .Spec.Type }}"
	LABEL="{{ .Label }}"
	PROFILE="{{ .Spec.ProfileURL }}"
	csip:CONTENTINFORMATIONTYPE="{{ .Spec.ContentInformationType }}"
	{{ with .Spec.OtherContentInformationType }}csip:OTHERCONTENTINFORMATIONTYPE="{{ . }}"{{ end }}
	xsi:schemaLocation="http://www.loc.gov/METS/ ../../schemas/mets1_12.xsd http://www.w3.org/1999/xlink ../../schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS ../../schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS ../../schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP" />

	{{ with .DescriptionFile -}}
    <dmdSec ID="{{ .Identifier }}" CREATED="{{ now }}" STATUS="CURRENT">
        <mdRef LOCTYPE="URL" MDTYPE="{{ $.Spec.DescriptiveMDType }}"{{ with $.Spec.DescriptiveMDTypeVersion }} MDTYPEVERSION="{{ . }}"{{ end }} xlink:type="simple" xlink:href="{{ encode .Path }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
    </dmdSec>
	{{- end }}

	{{ with .PremisFiles -}}
    <amdSec>
	{{ range . -}}
        <digiprovMD ID="{{ .Identifier }}" STATUS="CURRENT">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .Path }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
	{{ end -}}
    </amdSec>
	{{- end }}

	{{ $fileGrpID := identifier -}}
    <fileSec ID="{{ identifier}}">
        <fileGrp USE="data" ID="{{ $fileGrpID }}">
		{{ range .Files -}}
            <file ID="{{ identifier }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .Path }}"/>
            </file>
		{{ end -}}
        </fileGrp>
    </fileSec>

    <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
        <div ID="{{ identifier }}" LABEL="{{ .Name }}">
            <div ID="{{ identifier }}" LABEL="Metadata"{{ with .DescriptionFile }} DMDID="{{ .Identifier }}"{{ end }} {{ with .PremisFiles }}
                ADMID="{{ joinIdentifiers . }}" {{ end }}/>
            <div ID="{{ identifier }}" LABEL="Data">
                <fptr FILEID="{{ $fileGrpID }}" />
            </div>
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
	OBJID="{{ $OBJID }}"
	LABEL=""
	TYPE="{{ .Spec.Type }}"
	PROFILE="{{ .Spec.ProfileURL }}"
	csip:CONTENTINFORMATIONTYPE="{{ .Spec.ContentInformationType }}"
	{{ with .Spec.OtherContentInformationType }}csip:OTHERCONTENTINFORMATIONTYPE="{{ . }}"{{ end }}
 	xsi:schemaLocation="http://www.loc.gov/METS/ schemas/mets1_12.xsd http://www.w3.org/1999/xlink schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}"{{ with .Spec.RecordStatus }} RECORDSTATUS="{{ . }}"{{ end }} csip:OAISPACKAGETYPE="SIP">
	{{- range .Spec.Agents }}
		<agent ROLE="{{ .Role }}"{{ if .OtherRole }} OTHERROLE="{{ .OtherRole }}"{{ end }} TYPE="{{ .Type }}"{{ if .OtherType }} OTHERTYPE="{{ .OtherType }}"{{ end }}>
			<name>{{ .Name }}</name>
			{{- if .Note }}
			<note csip:NOTETYPE="{{ .NoteType }}">{{ .Note }}</note>
			{{- end }}
		</agent>
	{{- end }}
	</metsHdr>

	<!-- ref to descriptive metadata about IE -->
	{{ range .GetDescriptiveFiles -}}
    <dmdSec ID="{{ .Identifier }}" CREATED="{{ now }}" STATUS="CURRENT">
        <mdRef LOCTYPE="URL" MDTYPE="{{ $.Spec.DescriptiveMDType }}"{{ with $.Spec.DescriptiveMDTypeVersion }} MDTYPEVERSION="{{ . }}"{{ end }} xlink:type="simple" xlink:href="{{ encode .Path }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
    </dmdSec>
	{{- end }}

	<!-- ref to the PREMIS metadata about IE/subIE(s)/package -->
    {{ with .PremisFiles -}}
    <amdSec>
	{{ range . -}}
        <digiprovMD ID="{{ .Identifier }}" STATUS="CURRENT">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .Path }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
	{{ end -}}
    </amdSec>
    {{- end }}

    <!-- file section -->
    <fileSec ID="{{ identifier }}">
		<fileGrp ID="{{ $SCHEMAID }}" USE="Schemas">
			{{ range .SchemaFiles -}}
			<file ID="{{ .Identifier }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
				<FLocat xlink:type="simple" xlink:href="{{ .Path }}" LOCTYPE="URL"/>
			</file>
			{{ end}}
		</fileGrp>
		{{ with .DocumentationFiles -}}
		<fileGrp ID="{{ $DOCID }}" USE="Documentation">
			{{ range . -}}
			<file ID="{{ .Identifier }}" MIMETYPE="{{ .Mime }}" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
				<FLocat xlink:type="simple" xlink:href="{{ .Path }}" LOCTYPE="URL"/>
			</file>
			{{ end -}}
		</fileGrp>
		{{ end -}}
		{{ range .Root.Representations -}}
        <fileGrp ID="{{ .Identifier }}" USE="Representations/{{ .Name }}">
            <file ID="{{ .MetsFile.Identifier }}" MIMETYPE="{{ .MetsFile.Mime }}" SIZE="{{ .MetsFile.Size }}" CREATED="{{ .MetsFile.Created }}" CHECKSUM="{{ .MetsFile.Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .MetsFile.Path }}"/>
            </file>
        </fileGrp>
		{{ end }}
    </fileSec>

    <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
        <div ID="{{ identifier }}" LABEL="{{ $OBJID }}">
            <div ID="{{ identifier }}" LABEL="Metadata" DMDID="{{ .Root.DescriptionFile.Identifier }}"{{ with .PremisFiles }} ADMID="{{ joinIdentifiers . }}"{{ end }}/>
            <div ID="{{ identifier }}" LABEL="Schemas">
                <fptr FILEID="{{ $SCHEMAID }}"/>
            </div>
			{{ with .DocumentationFiles -}}
            <div ID="{{ identifier }}" LABEL="Documentation">
                <fptr FILEID="{{ $DOCID }}"/>
            </div>
			{{ end -}}
			{{ range .Root.Representations -}}
			<div ID="{{ identifier }}" LABEL="Representations/{{ .Name }}">
				<mptr xlink:type="simple" xlink:href="{{ .MetsFile.Path }}" LOCTYPE="URL" xlink:title="{{ .Identifier }}" />
			</div>
			{{ end }}
        </div>
    </structMap>
</mets>
{{ end }}
`))

// repView pairs a representation with the package-level spec its METS
// template needs; the embedded Representation keeps .Name/.Label/.Files/etc.
// resolving unchanged.
type repView struct {
	*sip.Representation
	Spec *sip.Spec
}

func EncodeRepresentation(w io.Writer, r *sip.Representation, spec *sip.Spec) error {
	return dc.ExecuteTemplate(w, "representation", repView{r, spec})
}

func EncodePackage(w io.Writer, p *sip.Package) error {
	return dc.ExecuteTemplate(w, "package", p)
}
