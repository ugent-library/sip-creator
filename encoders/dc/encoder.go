package dc

import (
	"io"
	"text/template"

	"github.com/ugent-library/meemoo-sip-creator/metadata"
)

var funcs = template.FuncMap{}

var dc = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "descriptive" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/1.0/basic" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xs="http://www.w3.org/2001/XMLSchema/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:edtf="http://id.loc.gov/datatypes/edtf/">
	<!-- linking id between dc and premis -->
	<dcterms:identifier>{{ .Identifier }}</dcterms:identifier>
	
	{{- if .Description.Title }}
	<dcterms:title xml:lang="nl">{{ .Description.Title }}</dcterms:title>
	{{- end }}
	
	{{- if .Description.Description }}
	<dcterms:description xml:lang="nl">{{ .Description.Description }}</dcterms:description>
	{{- end }}
	
	{{- if .Description.Created}}
	<dcterms:created xsi:type="edtf:EDTF-level1">{{ .Description.Created }}</dcterms:created>
	{{- end }}
</metadata>
{{ end}}
`))

func EncodeDescriptive(w io.Writer, e metadata.Entity) error {
	return dc.ExecuteTemplate(w, "descriptive", e)
}
