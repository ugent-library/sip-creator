package mets

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/ugent-library/sip-creator/sip"
)

var idStore []string

// Make sure METS identifiers are only minted once across SIP
func identifier() string {
	id := fmt.Sprintf("uuid-%s", uuid.New().String())

	if lo.Contains(idStore, id) {
		return identifier()
	}

	return id
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
	TYPE="Photographs – Digital"
	PROFILE="https://earkcsip.dilcis.eu/profile/E-ARK-CSIP.xml"
	csip:CONTENTINFORMATIONTYPE="OTHER"
	csip:OTHERCONTENTINFORMATIONTYPE="https://data.hetarchief.be/id/sip/2.0/basic"
	xsi:schemaLocation="http://www.loc.gov/METS/ ../../schemas/mets1_12.xsd http://www.w3.org/1999/xlink ../../schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS ../../schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS ../../schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP" />

	{{ $provMDID := identifier -}}
    <amdSec>
        <digiprovMD ID="{{ $provMDID }}">
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
            <div ID="{{ identifier }}" LABEL="Representations">
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
	TYPE="Photographs – Digital"
	PROFILE="https://earkcsip.dilcis.eu/profile/E-ARK-CSIP.xml"
	csip:CONTENTINFORMATIONTYPE="OTHER"
	csip:OTHERCONTENTINFORMATIONTYPE="https://data.hetarchief.be/id/sip/2.0/basic"
 	xsi:schemaLocation="http://www.loc.gov/METS/ schemas/mets1_12.xsd http://www.w3.org/1999/xlink schemas/xlink.xsd https://dilcis.eu/XML/METS/CSIPExtensionMETS schemas/DILCISExtensionMETS.xsd https://dilcis.eu/XML/METS/SIPExtensionMETS schemas/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" csip:OAISPACKAGETYPE="SIP">
		<agent ROLE="CREATOR" TYPE="OTHER" OTHERTYPE="SOFTWARE">
			<name>SIP creator</name>
			<note csip:NOTETYPE="SOFTWARE VERSION">0.1.</note>
	  	</agent>
	  	<agent ROLE="CREATOR" OTHERROLE="OTHERROLE" TYPE="ORGANIZATION">
			<name>Universiteitsbibliotheek Gent</name>
	  	</agent>
	</metsHdr>

	<!-- ref to descriptive metadata about IE -->
	{{ range .GetDescriptiveFiles -}}
    <dmdSec ID="{{ .Identifier }}" CREATED="{{ now }}">
        <mdRef LOCTYPE="URL" MDTYPE="DC" xlink:type="simple" xlink:href="{{ encode .Path }}" MIMETYPE="text/xml" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
    </dmdSec>
	{{- end }}

	<!-- ref to the PREMIS metadata about IE/subIE(s)/package -->
    <amdSec>
        <digiprovMD ID="{{ .PremisFile.Identifier }}" STATUS="new">
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

func EncodeRepresentation(w io.Writer, r *sip.Representation) error {
	return dc.ExecuteTemplate(w, "representation", r)
}

func EncodePackage(w io.Writer, p *sip.Package) error {
	return dc.ExecuteTemplate(w, "package", p)
}
