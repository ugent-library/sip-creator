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
<simpledc xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../schemas/dc.xsd">

	<identifier>{{ .Identifier }}</identifier>
	{{- with .DublinCoreTerms.Identifier }}
	<identifier>{{ . }}</identifier>
	{{- end }}

	{{- if .Title.Value }}
	<title>{{ .Title.Value }}</title>
	{{- end }}

	{{- range .DublinCoreTerms.Creator }}
	<creator>{{ . }}</creator>
	{{- end }}

	{{- range .Subject }}
	<subject>{{ .Value }}</subject>
	{{- end }}

	{{- if .Description.Value }}
	<description>{{ .Description.Value }}</description>
	{{- end }}
	{{- if .Abstract.Value }}
	<description>{{ .Abstract.Value }}</description>
	{{- end }}

	{{- range .DublinCoreTerms.Publisher }}
	<publisher>{{ . }}</publisher>
	{{- end }}

	{{- range .DublinCoreTerms.Contributor }}
	<contributor>{{ . }}</contributor>
	{{- end }}

	{{- with .Created }}
	<date>{{ . }}</date>
	{{- end }}

	{{- range .Type }}
	<type>{{ . }}</type>
	{{- end }}

	{{- range .Language }}
	<language>{{ . }}</language>
	{{- end }}

	{{- with .Rights }}
	<rights>{{ . }}</rights>
	{{- end }}
</simpledc>
{{ end}}
`))

func Encode(w io.Writer, d *Description) error {
	return md.ExecuteTemplate(w, "dc+schema", d)
}

// EncodeDC writes d as a simple Dublin Core document (the
// dc_SimpleDC20021212 shape RODA renders and indexes natively).
func EncodeDC(w io.Writer, d *Description) error {
	return md.ExecuteTemplate(w, "dc", d)
}
