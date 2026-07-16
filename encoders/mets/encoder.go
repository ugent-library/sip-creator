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
	OBJID="{{ .Label }}"
	TYPE="{{ .Spec.Type }}"
	LABEL=""
	PROFILE="{{ .Spec.ProfileURL }}"
	csip:CONTENTINFORMATIONTYPE="{{ .Spec.ContentInformationType }}"
	csip:OTHERCONTENTINFORMATIONTYPE="{{ .Spec.OtherContentInformationType }}"
	xsi:schemaLocation="http://www.loc.gov/METS/ ../../schemas/mets1_12.xsd http://www.w3.org/1999/xlink ../../schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS ../../schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS ../../schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP" />

	{{ $provMDID := identifier -}}
    <amdSec>
        <digiprovMD ID="{{ $provMDID }}" STATUS="CURRENT">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .PremisFile.Path }}" MIMETYPE="text/xml" SIZE="{{ .PremisFile.Size }}" CREATED="{{ .PremisFile.Created }}" CHECKSUM="{{ .PremisFile.Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
    </amdSec>

	{{ $fileGrpID := identifier -}}
    <fileSec ID="{{ identifier}}">
        <fileGrp USE="data" ID="{{ $fileGrpID }}">
		{{ range .Files -}}
            <file ID="{{ identifier }}" MIMETYPE="text/xml" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .Path }}"/>
            </file>
		{{ end -}}
        </fileGrp>
    </fileSec>

    <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
        <div ID="{{ identifier }}" LABEL="{{ .Label }}">
            <div ID="{{ identifier }}" LABEL="Metadata" 
                ADMID="{{ $provMDID }}" />
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
	csip:OTHERCONTENTINFORMATIONTYPE="{{ .Spec.OtherContentInformationType }}"
 	xsi:schemaLocation="http://www.loc.gov/METS/ schemas/mets1_12.xsd http://www.w3.org/1999/xlink schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP">
	{{- range .Spec.Agents }}
		<agent ROLE="{{ .Role }}"{{ if .OtherRole }} OTHERROLE="{{ .OtherRole }}"{{ end }} TYPE="{{ .Type }}"{{ if .OtherType }} OTHERTYPE="{{ .OtherType }}"{{ end }}>
			<name>{{ .Name }}</name>
			{{- if .Note }}
			<note csip:NOTETYPE="SOFTWARE VERSION">{{ .Note }}</note>
			{{- end }}
		</agent>
	{{- end }}
	</metsHdr>

	<!-- ref to descriptive metadata about IE -->
	{{ range .GetDescriptiveFiles -}}
    <dmdSec ID="{{ .Identifier }}" CREATED="{{ now }}" STATUS="CURRENT">
        <mdRef LOCTYPE="URL" MDTYPE="DC" xlink:type="simple" xlink:href="{{ encode .Path }}" MIMETYPE="text/xml" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
    </dmdSec>
	{{- end }}

	<!-- ref to the PREMIS metadata about IE/subIE(s)/package -->
    <amdSec>
        <digiprovMD ID="{{ .PremisFile.Identifier }}" STATUS="CURRENT">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .PremisFile.Path }}" MIMETYPE="text/xml" SIZE="{{ .PremisFile.Size }}" CREATED="{{ .PremisFile.Created }}" CHECKSUM="{{ .PremisFile.Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
    </amdSec>

    <!-- file section -->
    <fileSec ID="{{ identifier }}">
		<fileGrp ID="{{ $SCHEMAID }}" USE="Schemas">
			{{ range .SchemaFiles -}}
			<file ID="{{ .Identifier }}" MIMETYPE="application/octet-stream" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
				<FLocat xlink:type="simple" xlink:href="{{ .Path }}" LOCTYPE="URL"/>
			</file>
			{{ end}}
		</fileGrp>
		{{ range .Root.Representations -}}
        <fileGrp ID="{{ .Identifier }}" USE="Representations/{{ .Label }}">
            <file ID="{{ .MetsFile.Identifier }}" MIMETYPE="text/xml" SIZE="{{ .MetsFile.Size }}" CREATED="{{ .MetsFile.Created }}" CHECKSUM="{{ .MetsFile.Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .MetsFile.Path }}"/>
            </file>
        </fileGrp>
		{{ end }}
    </fileSec>

    <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
        <div ID="{{ identifier }}" LABEL="{{ $OBJID }}">
            <div ID="{{ identifier }}" LABEL="Metadata" DMDID="{{ .Root.DescriptionFile.Identifier }}" ADMID="{{ .PremisFile.Identifier }}"/>
            <div ID="{{ identifier }}" LABEL="Schemas">
                <fptr FILEID="{{ $SCHEMAID }}"/>
            </div>
			{{ range .Root.Representations -}}
			<div ID="{{ identifier }}" LABEL="Representations/{{ .Label }}">
				<mptr xlink:type="simple" xlink:href="{{ .MetsFile.Path }}" LOCTYPE="URL" xlink:title="{{ .Identifier }}" />
			</div>
			{{ end }}
        </div>
    </structMap>
</mets>
{{ end }}
`))

// repView pairs a representation with the package-level spec its METS
// template needs; the embedded Representation keeps .Label/.Files/etc.
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
