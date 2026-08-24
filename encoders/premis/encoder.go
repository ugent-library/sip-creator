package premis

import (
	"io"
	"text/template"

	"github.com/ugent-library/sip-creator/sip"
)

var premis = template.Must(template.New("").Parse(`
{{ define "entity" -}}
<?xml version='1.0' encoding='UTF-8'?>
<premis:premis version="3.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:premis="http://www.loc.gov/premis/v3" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd">

  <premis:object xsi:type="premis:intellectualEntity">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>{{ .Identifier }}</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    {{- range $k, $v := .AdditionalIdentifiers }}
    <premis:objectIdentifier>
      <premis:objectIdentifierType>{{ $k }}</premis:objectIdentifierType>
      <premis:objectIdentifierValue>{{ $v }}</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    {{- end }}
    {{- range .Representations }}
      {{- template "isRepresentedBy" . }}
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
      <premis:objectIdentifierValue>{{ .Identifier }}</premis:objectIdentifierValue>
    </premis:objectIdentifier>

    <!-- relationship between representation and its files -->
    {{- range .Files }}
      {{- template "includes" . }}
    {{- end }}

    <!-- relationship between representation and its IE/subIE -->
    {{- template "represents" .Entity }}
  </premis:object>

  <!-- Files -->
  {{- range .Files }}
{{ template "file" . }}
  {{- end }}
</premis:premis>
{{ end }}

{{ define "file" }}
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>UUID</premis:objectIdentifierType>
      <premis:objectIdentifierValue>{{ .Identifier }}</premis:objectIdentifierValue>
    </premis:objectIdentifier>

    <premis:objectCharacteristics>
      <premis:fixity>
        <premis:messageDigestAlgorithm authority="cryptographicHashFunctions" authorityURI="http://id.loc.gov/vocabulary/preservation/cryptographicHashFunctions" valueURI="http://id.loc.gov/vocabulary/preservation/cryptographicHashFunctions/md5">MD5</premis:messageDigestAlgorithm>
        <premis:messageDigest>{{ .Checksum }}</premis:messageDigest>
      </premis:fixity>
      <premis:size>{{ .Size }}</premis:size>
      {{- with .Format }}
      <premis:format>
        <premis:formatRegistry>
          <premis:formatRegistryName>{{ .FormatRegistry.Name }}</premis:formatRegistryName>
          <premis:formatRegistryKey>{{ .FormatRegistry.Key }}</premis:formatRegistryKey>
          <premis:formatRegistryRole authority="formatRegistryRole" authorityURI="http://id.loc.gov/vocabulary/preservation/formatRegistryRole" valueURI="http://id.loc.gov/vocabulary/preservation/formatRegistryRole/spe">{{ .FormatRegistry.Role }}</premis:formatRegistryRole>
        </premis:formatRegistry>
      </premis:format>
      {{- end }}
    </premis:objectCharacteristics>

    <premis:originalName>{{ .Name }}</premis:originalName>

    <!-- relationship between file and its representation -->
    {{- template "isIncludedIn" .Representation }}
  </premis:object>
{{- end }}

{{ define "isRepresentedBy" }}
    <premis:relationship>
      <premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
      <premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/isr">is represented by</premis:relationshipSubType>
      <premis:relatedObjectIdentifier>
        <premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
        <premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
      </premis:relatedObjectIdentifier>
    </premis:relationship>
{{- end }}

{{ define "represents" }}
    <premis:relationship>
      <premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
      <premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/rep">represents</premis:relationshipSubType>
      <premis:relatedObjectIdentifier>
        <premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
        <premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
      </premis:relatedObjectIdentifier>
    </premis:relationship>
{{- end }}

{{ define "includes" }}
    <premis:relationship>
      <premis:relationshipType authority="relationshipType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipType/str">structural</premis:relationshipType>
      <premis:relationshipSubType authority="relationshipSubType" authorityURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType" valueURI="http://id.loc.gov/vocabulary/preservation/relationshipSubType/inc">includes</premis:relationshipSubType>
      <premis:relatedObjectIdentifier>
        <premis:relatedObjectIdentifierType>UUID</premis:relatedObjectIdentifierType>
        <premis:relatedObjectIdentifierValue>{{ .Identifier }}</premis:relatedObjectIdentifierValue>
      </premis:relatedObjectIdentifier>
    </premis:relationship>
{{- end }}

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

// EncodeEntity writes the package PREMIS document: the intellectual entity
// and its relationships to the representations.
func EncodeEntity(w io.Writer, e *sip.Entity) error {
	return premis.ExecuteTemplate(w, "entity", e)
}

// EncodeRepresentation writes a representation's PREMIS document: the
// representation object, its files with fixity and format, and the
// relationships tying them together.
func EncodeRepresentation(w io.Writer, r *sip.Representation) error {
	return premis.ExecuteTemplate(w, "representation", r)
}
