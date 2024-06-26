package mets

import (
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/ugent-library/sip-creator/structure"
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
	"now": func() string {
		return time.Now().Format(time.RFC3339Nano)
	},
	"joinIdentifiers": func(files []*structure.File) string {
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
	TYPE="Photographs - Digital"
	PROFILE="https://earksip.dilcis.eu/profile/E-ARK-SIP.xml"
	csip:CONTENTINFORMATIONTYPE="OTHER"
	csip:OTHERCONTENTINFORMATIONTYPE="https://data.hetarchief.be/id/sip/1.0/basic"
	xsi:schemaLocation="https://www.w3.org./1999/xlink http://www.loc.gov/standards/xlink/xlink.xsd http://www.loc.gov/METS/ https://www.loc.gov/standards/mets/mets.xsd https://DILCIS.eu/XML/METS/CSIPExtensionMETS https://earkcsip.dilcis.eu/schema/DILCISExtensionMETS.xsd https://DILCIS.eu/XML/METS/SIPExtensionMETS https://earksip.dilcis.eu/schema/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" />

	{{ $provMDID := identifier -}}
    <amdSec>
        <digiprovMD ID="{{ $provMDID }}">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .PremisFile.Name }}" MIMETYPE="text/xml" SIZE="{{ .PremisFile.Size }}" CREATED="{{ .PremisFile.Created }}" CHECKSUM="{{ .PremisFile.Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
    </amdSec>

	{{ $fileGrpID := identifier -}}
    <fileSec ID="{{ identifier}}">
        <fileGrp USE="data" ID="{{ $fileGrpID }}">
		{{ range .Files -}}
            <file ID="{{ identifier }}" MIMETYPE="text/xml" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .Name }}"/>
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
<?xml version='1.0' encoding='UTF-8'?>
<mets xmlns="http://www.loc.gov/METS/" 
	xmlns:csip="https://DILCIS.eu/XML/METS/CSIPExtensionMETS" 
	xmlns:sip="https://DILCIS.eu/XML/METS/SIPExtensionMETS" 
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
	xmlns:xlink="http://www.w3.org/1999/xlink" 
	OBJID="{{ identifier }}"
	TYPE="Photographs - Digital"
	PROFILE="https://earksip.dilcis.eu/profile/E-ARK-SIP.xml"
	csip:CONTENTINFORMATIONTYPE="OTHER"
	csip:OTHERCONTENTINFORMATIONTYPE="https://data.hetarchief.be/id/sip/1.0/basic"
	xsi:schemaLocation="https://www.w3.org./1999/xlink http://www.loc.gov/standards/xlink/xlink.xsd http://www.loc.gov/METS/ https://www.loc.gov/standards/mets/mets.xsd https://DILCIS.eu/XML/METS/CSIPExtensionMETS https://earkcsip.dilcis.eu/schema/DILCISExtensionMETS.xsd https://DILCIS.eu/XML/METS/SIPExtensionMETS https://earksip.dilcis.eu/schema/DILCISExtensionSIPMETS.xsd">

	<metsHdr CREATEDATE="{{ now }}" />

	<!-- ref to descriptive metadata about IE -->
	{{ range .DescriptiveFiles -}}
    <dmdSec ID="{{ .Identifier }}">
        <mdRef LOCTYPE="URL" MDTYPE="DC" xlink:type="simple" xlink:href="{{ .Name }}" MIMETYPE="text/xml" SIZE="{{ .Size }}" CREATED="{{ .Created }}" CHECKSUM="{{ .Checksum }}" CHECKSUMTYPE="MD5" />
    </dmdSec>
	{{ end -}}

	<!-- ref to the PREMIS metadata about IE/subIE(s)/package -->
    <amdSec>
        <digiprovMD ID="{{ .PremisFile.Identifier }}">
            <mdRef LOCTYPE="URL" MDTYPE="PREMIS" xlink:type="simple" xlink:href="{{ .PremisFile.Name }}" MIMETYPE="text/xml" SIZE="{{ .PremisFile.Size }}" CREATED="{{ .PremisFile.Created }}" CHECKSUM="{{ .PremisFile.Checksum }}" CHECKSUMTYPE="MD5" />
        </digiprovMD>
    </amdSec>

    <!-- file section -->
    <fileSec ID="{{ identifier }}">
		{{ range .Root.Representations -}}
        <fileGrp USE="Representations/{{ .Label }}" ID="{{ .Identifier }}">
            <file ID="uuid-ba89c101-109a-4fe3-a4ed-0c276b20e14c" MIMETYPE="text/xml" SIZE="{{ .MetsFile.Size }}" CREATED="{{ .MetsFile.Created }}" CHECKSUM="{{ .MetsFile.Checksum }}" CHECKSUMTYPE="MD5">
                <FLocat LOCTYPE="URL" xlink:type="simple" xlink:href="{{ .MetsFile.Name }}"/>
            </file>
        </fileGrp>
		{{ end }}
    </fileSec>

    <structMap ID="{{ identifier }}" TYPE="PHYSICAL" LABEL="CSIP">
        <div ID="{{ identifier }}" LABEL="basic-package">
            <div ID="{{ identifier }}" LABEL="Metadata" DMDID="{{ .Root.DescriptionFile}}" ADMID="{{ .PremisFile.Identifier }}"/>
            <div ID="uuid-47ea3b5a-cc10-491c-94fc-46189fb266ae" LABEL="Representations/representation_1">
                <mptr xlink:type="simple" xlink:href="./representations/representation_1/mets.xml" LOCTYPE="URL" xlink:title="uuid-6ec1cf49-0d4a-413d-9170-4dcb4bd09473"/>
            </div>
        </div>
    </structMap>
</mets>
{{ end }}
`))

func EncodeRepresentation(w io.Writer, r *structure.Representation) error {
	return dc.ExecuteTemplate(w, "representation", r)
}

func EncodePackage(w io.Writer, p *structure.Package) error {
	return dc.ExecuteTemplate(w, "package", p)
}
