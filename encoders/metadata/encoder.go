package metadata

import (
	"io"
	"text/template"
)

var funcs = template.FuncMap{}

var md = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "dc+schema" -}}
<?xml version='1.0' encoding='UTF-8'?>
<metadata xmlns="https://data.hetarchief.be/id/sip/2.0/basic" 
	xmlns:dcterms="http://purl.org/dc/terms/" 
	xmlns:xs="http://www.w3.org/2001/XMLSchema/" 
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" 
	xmlns:edtf="http://id.loc.gov/datatypes/edtf/"
	xmlns:schema="https://schema.org/"
	xsi:schemaLocation="https://data.hetarchief.be/id/sip/2.0/basic ../../schemas/descriptive_basic.xsd">

	<dcterms:identifier>{{ .Identifier }}</dcterms:identifier>
	
	{{- if .Title }}
	<dcterms:title xml:lang="{{ .Title.Lang }}">{{ .Title.Value }}</dcterms:title>
	{{- end }}
	
	{{- if .Description }}
	<dcterms:description xml:lang="{{ .Description.Lang }}">{{ .Description.Value }}</dcterms:description>
	{{- end }}
	
	{{- if .Created }}
	<dcterms:created xsi:type="edtf:EDTF-level1">{{ .Created }}</dcterms:created>
	{{- end }}

	{{- if .Alternative }}
	{{ range .Alternative }}
	<dcterms:alternative xml:lang="nl">{{ . }}</dcterms:alternative>
	{{ end }}
	{{- end }}

	{{- if .Extent }}
	<dcterms:extent>{{ .Extent }}</dcterms:extent>
	{{- end }}

	{{- if .Available }}
	<dcterms:available>{{ .Available }}</dcterms:available>
	{{- end }}

	{{- if .Abstract }}
	<dcterms:abstract xml:lang="{{ .Abstract.Lang }}">{{ .Abstract.Value }}</dcterms:abstract>
	{{- end }}

	{{- if .Issued }}
	<dcterms:issued xsi:type="edtf:EDTF-level1">{{ .Issued }}</dcterms:issued>
	{{- end }}

	{{- if .DublinCoreTerms.Creator }}
	{{ range .DublinCoreTerms.Creator }}
	<dcterms:creator>{{ . }}</dcterms:creator>
	{{ end }}
	{{- end}}	

	{{- if .DublinCoreTerms.Publisher }}
	{{ range .DublinCoreTerms.Publisher }}
	<dcterms:publisher>{{ . }}</dcterms:publisher>
	{{ end }}
	{{- end}}

	{{- if .DublinCoreTerms.Contributor }}
	{{ range .DublinCoreTerms.Contributor }}
	<dcterms:contributor>{{ . }}</dcterms:contributor>
	{{ end }}
	{{- end }}

	{{- if .Spatial }}
	{{ range .Spatial }}
	<dcterms:spatial>{{ . }}</dcterms:spatial>
	{{ end }}
	{{- end }}

	{{- if .Temporal }}
	{{ range .Temporal }}
	<dcterms:temporal>{{ . }}</dcterms:temporal>
	{{ end }}
	{{- end }}

	{{- if .Subject }}
	{{ range .Subject }}
	<dcterms:subject xml:lang="{{ .Subject.Lang }}">{{ .Subject.Value }}</dcterms:subject>
	{{ end }}
	{{- end }}

	{{- if .Language }}
	{{ range .Language }}
	<dcterms:language>{{ . }}</dcterms:language>
	{{ end }}
	{{- end }}

	{{- if .License }}
	{{ range .License }}
	<dcterms:license>{{ . }}</dcterms:license>
	{{ end }}
	{{- end }}

	{{- if .RightsHolder }}
	<dcterms:rightsHolder>{{ .RightsHolder }}</dcterms:rightsHolder>
	{{- end }}

	{{- if .Rights }}
	<dcterms:rights xml:lang="nl">{{ .Rights }}</dcterms:rights>
	{{- end }}

	{{- if .Type }}
	{{ range .Type }}
	<dcterms:type>{{ . }}</dcterms:type>
	{{ end }}
	{{- end }}
</metadata>
{{ end}}
{{ define "dc" -}}
<?xml version='1.0' encoding='UTF-8'?>
<simpledc xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:noNamespaceSchemaLocation="../../schemas/dc.xsd">

	<identifier>{{ .Identifier }}</identifier>
	
	{{- if .Title }}
	<title>{{ .Title.Value }}</title>
	{{- end }}

	{{- if .DublinCore.Creator }}
	{{ range .DublinCore.Creator }}
	<creator>{{ . }}</creator>
	{{ end }}
	{{- end}}

	{{- if .Subject }}
	{{ range .Subject }}
	<subject>{{ .Subject.Value }}</subject>
	{{ end }}
	{{- end }}
	
	{{- if .Description }}
	<description>{{ .Description.Value }}</description>
	{{- end }}

	{{- if .DublinCore.Publisher }}
	{{ range .DublinCore.Publisher }}
	<publisher>{{ . }}</publisher>
	{{ end }}
	{{- end}}
	
	{{- if .DublinCore.Contributor }}
	{{ range .DublinCore.Contributor }}
	<contributor>{{ . }}</contributor>
	{{ end }}
	{{- end }}

	{{- if .Date }}
	<dated>{{ .Date }}</dated>
	{{- end }}

	{{- if .Type }}
	{{ range .Type }}
	<type>{{ . }}</type>
	{{ end }}
	{{- end }}

	{{- if .Format }}
	<format>{{ . }}</format>
	{{- end }}

	{{- if .Source }}
	<source>{{ . }}</source>
	{{- end }}

	{{- if .Language }}
	{{ range .Language }}
	<language>{{ . }}</language>
	{{ end }}
	{{- end }}

	{{- if .Relation }}
	{{ range .Relation }}
	<relation>{{ . }}</relation>
	{{ end }}
	{{- end }}

	{{- if .Coverage }}
	<coverage>{{ . }}</coverage>
	{{- end }}

	{{- if .Rights }}
	<rights>{{ .Rights }}</rights>
	{{- end }}
</simpledc>
{{ end}}
`))

func Encode(w io.Writer, d *Description) error {
	return md.ExecuteTemplate(w, "dc+schema", d)
}
