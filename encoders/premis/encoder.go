package premis

import (
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/structure"
)

type representationWithEntity struct {
	Representation *structure.Representation
	Entity         structure.Entity
}

var funcs = template.FuncMap{
	"fileWithRepresentation": func(f *structure.File, r *structure.Representation) any {
		return struct {
			File           *structure.File
			Representation *structure.Representation
		}{
			File:           f,
			Representation: r,
		}
	},
}

var premis = template.Must(template.New("").Funcs(funcs).Parse(`
{{ define "entity" -}}
<?xml version='1.0' encoding='UTF-8'?>
<premis:premis version="3.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:premis="http://www.loc.gov/premis/v3" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd">

	<premis:object xsi:type="premis:intellectualEntity">
		<premis:objectIdentifier>
			<premis:objectIdentifierType>UUID</premis:objectIdentifierType>
			<premis:objectIdentifierValue>{{ .Identifier }}</premis:objectIdentifierValue>
		</premis:objectIdentifier>

		{{- range .Representations }}
			{{ template "isRepresentedBy" . -}}
		{{- end }}
	</premis:object>
</premis:premis>
{{ end }}

{{ define "representation" -}}
<?xml version='1.0' encoding='UTF-8'?>
<premis:premis version="3.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:premis="http://www.loc.gov/premis/v3" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd">

	<premis:object xsi:type="premis:representation">

		<premis:objectIdentifier>
		<premis:objectIdentifierType>UUID</premis:objectIdentifierType>
		<premis:objectIdentifierValue>{{ .Representation.Identifier }}</premis:objectIdentifierValue>
		</premis:objectIdentifier>

		<!-- relationship between representation and its files -->
		{{- range .Representation.Files }}
			{{ template "includes" . }}
		{{ end -}}

		<!-- relationship between representation and its IE/subIE -->
		{{ template "represents" .Entity }}
	</premis:object>

	<!-- Files -->
	{{- $representation := .Representation }}
	{{- range .Representation.Files }}
		{{ template "file" (fileWithRepresentation . $representation) }}
	{{- end }}
</premis:premis>
{{ end }}

{{ define "file" }}
	<premis:object xsi:type="premis:file">

		<premis:objectIdentifier>
			<premis:objectIdentifierType>UUID</premis:objectIdentifierType>
			<premis:objectIdentifierValue>{{ .File.Identifier }}</premis:objectIdentifierValue>
		</premis:objectIdentifier>

		<premis:objectCharacteristics>
			<premis:fixity>
				<premis:messageDigestAlgorithm authority="cryptographicHashFunctions" authorityURI="http://id.loc.gov/vocabulary/preservation/cryptographicHashFunctions" valueURI="http://id.loc.gov/vocabulary/preservation/cryptographicHashFunctions/md5">
						MD5
				</premis:messageDigestAlgorithm>
				<premis:messageDigest>{{ .File.Checksum }}</premis:messageDigest>
			</premis:fixity>
			<premis:size>{{ .File.Size }}</premis:size>
			<premis:format>
				<premis:formatRegistry>
				<premis:formatRegistryName>PRONOM</premis:formatRegistryName>
				<premis:formatRegistryKey>{{ .File.Format }}</premis:formatRegistryKey>
				<premis:formatRegistryRole authority="formatRegistryRole" authorityURI="http://id.loc.gov/vocabulary/preservation/formatRegistryRole" valueURI="http://id.loc.gov/vocabulary/preservation/formatRegistryRole/spe">specification</premis:formatRegistryRole>
				</premis:formatRegistry>
				<premis:formatNote></premis:formatNote>
			</premis:format>
		</premis:objectCharacteristics>

		<premis:originalName>{{ .File.Name }}</premis:originalName>

		<!-- relationship between file and its representation -->
		{{ template "isIncludedIn" .Representation }}
		
	</premis:object>
{{end }}

{{ define "isRepresentedBy" }}
		<premis:relationship>
			<premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
			<premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/isr">is represented by</premis:relationshipSubType>
			<premis:relatedObjectIdentifier>
				<premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
				<premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
			</premis:relatedObjectIdentifier>
		</premis:relationship>
{{ end }}

{{ define "represents" }}
		<premis:relationship>
			<premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
			<premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/rep">represents</premis:relationshipSubType>
			<premis:relatedObjectIdentifier>
				<premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
				<premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
			</premis:relatedObjectIdentifier>
		</premis:relationship>
{{ end }}

{{ define "includes" }}
		<premis:relationship>
			<premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
			<premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/inc">includes</premis:relationshipSubType>
			<premis:relatedObjectIdentifier>
				<premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
				<premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
			</premis:relatedObjectIdentifier>
		</premis:relationship>
{{ end }}

{{ define "isIncludedIn" }}
		<premis:relationship>
			<premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
			<premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/isi">is included in</premis:relationshipSubType>
			<premis:relatedObjectIdentifier>
				<premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
				<premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
			</premis:relatedObjectIdentifier>
		</premis:relationship>
{{- end }}
`))

func EncodeEntity(w io.Writer, e structure.Entity) error {
	return premis.ExecuteTemplate(w, "entity", e)
}

func EncodeRepresentation(w io.Writer, e structure.Entity, r *structure.Representation) error {
	return premis.ExecuteTemplate(w, "representation", &representationWithEntity{
		Entity:         e,
		Representation: r,
	})
}
