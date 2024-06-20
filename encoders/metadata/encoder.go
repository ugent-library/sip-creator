package metadata

import (
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/structure"
)

var funcs = template.FuncMap{}

var md = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "descriptive" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/1.0/basic" 
	xmlns:dcterms="http://purl.org/dc/terms/" 
	xmlns:xs="http://www.w3.org/2001/XMLSchema/" 
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
	xmlns:edtf="http://id.loc.gov/datatypes/edtf/"
	xmlns:schema="https://schema.org/">

	<!-- linking id between dc and premis -->
	<dcterms:identifier>{{ .Identifier }}</dcterms:identifier>
	
	{{- if .Description.Title }}
	<dcterms:title xml:lang="{{ .Description.Title.Lang }}">{{ .Description.Title.Value }}</dcterms:title>
	{{- end }}
	
	{{- if .Description.Description }}
	<dcterms:description xml:lang="{{ .Description.Description.Lang }}">{{ .Description.Description.Value }}</dcterms:description>
	{{- end }}
	
	{{- if .Description.Created }}
	<dcterms:created xsi:type="edtf:EDTF-level1">{{ .Description.Created }}</dcterms:created>
	{{- end }}

	{{- if .Description.Alternative }}
	{{ range .Description.Alternative }}
	<dcterms:alternative xml:lang="nl">{{ . }}</dcterms:alternative>
	{{ end }}
	{{- end }}

	{{- if .Description.Extent }}
	<dcterms:extent>{{ .Description.Extent }}</dcterms:extent>
	{{- end }}

	{{- if .Description.Available }}
	<dcterms:available>{{ .Description.Available }}</dcterms:available>
	{{- end }}

	{{- if .Description.Abstract }}
	<dcterms:abstract xml:lang="{{ .Description.Abstract.Lang }}">{{ .Description.Abstract.Value }}</dcterms:abstract>
	{{- end }}

	{{- if .Description.Issued }}
	<dcterms:issued xsi:type="edtf:EDTF-level1">{{ .Description.Issued }}</dcterms:issued>
	{{- end }}

	{{- if .Description.Publisher }}
	{{ range .Description.Publisher }}
	<dcterms:publisher>{{ . }}</dcterms:publisher>
	{{ end }}
	{{- end}}

	{{- if .Description.Contributor }}
	{{ range .Description.Contributor }}
	<dcterms:contributor>{{ . }}</dcterms:contributor>
	{{ end }}
	{{- end }}

	{{- if .Description.Spatial }}
	{{ range .Description.Spatial }}
	<dcterms:spatial>{{ . }}</dcterms:spatial>
	{{ end }}
	{{- end }}

	{{- if .Description.Temporal }}
	{{ range .Description.Temporal }}
	<dcterms:temporal>{{ . }}</dcterms:temporal>
	{{ end }}
	{{- end }}

	{{- if .Description.Subject }}
	{{ range .Description.Subject }}
	<dcterms:subject xml:lang="{{ .Description.Subject.Lang }}">{{ .Description.Subject.Value }}</dcterms:subject>
	{{ end }}
	{{- end }}

	{{- if .Description.Language }}
	{{ range .Description.Language }}
	<dcterms:language>{{ . }}</dcterms:language>
	{{ end }}
	{{- end }}

	{{- if .Description.License }}
	{{ range .Description.License }}
	<dcterms:license>{{ . }}</dcterms:license>
	{{ end }}
	{{- end }}

	{{- if .Description.RightsHolder }}
	<dcterms:rightsHolder>{{ .Description.RightsHolder }}</dcterms:rightsHolder>
	{{- end }}

	{{- if .Description.Rights }}
	<dcterms:rights xml:lang="nl">{{ .Description.Rights }}</dcterms:rights>
	{{- end }}

	{{- if .Description.Type }}
	{{ range .Description.Type }}
	<dcterms:type>{{ . }}</dcterms:type>
	{{ end }}
	{{- end }}
</metadata>
{{ end}}
`))

func EncodeMetadata(w io.Writer, e structure.Entity) error {
	return md.ExecuteTemplate(w, "descriptive", e)
}
