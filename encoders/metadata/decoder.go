package metadata

import (
	"encoding/json"
	"io"
)

type Description struct {
	Identifier string
	// DublinCore
	DublinCoreTerms
	Schema
}

func (d *Description) GetLocalIdentifier(scheme string) string {
	switch scheme {
	case "dcterms":
		return d.DublinCoreTerms.Identifier
	}

	return ""
}

func (d *Description) SetObjectIdentifier(id string) {
	d.Identifier = id
}

// type DublinCore struct {
// 	Title       Text     `json:"title"`
// 	Creator     []string `json:"creator"`
// 	Subject     []Text   `json:"subject"`
// 	Description Text     `json:"description"`
// 	Publisher   []string `json:"publisher"`
// 	Contributor []string `json:"contributor"`
// 	Date        string   `json:"date"`
// 	Type        []string `json:"type"`
// 	Format      string   `json:"format"`
// 	Identifier  string   `json:"identifier"`
// 	Source      string   `json:"source"`
// 	Language    Text     `json:"language"`
// 	Relation    []string `json:"relation"`
// 	Coverage    string   `json:"coverage"`
// 	Rights      string   `json:"rights"`
// }

type DublinCoreTerms struct {
	Title        Text     `json:"dcterms:title"`
	Alternative  []string `json:"dcterms:alternative"`
	Identifier   string   `json:"dcterms:identifier"`
	Extent       string   `json:"dcterms:extent"`
	Available    string   `json:"dcterms:available"`
	Description  Text     `json:"dcterms:description"`
	Abstract     Text     `json:"dcterms:abstract"`
	Created      string   `json:"dcterms:created"`
	Issued       string   `json:"dcterms:issued"`
	Publisher    []string `json:"dcterms:publisher"`
	Contributor  []string `json:"dcterms:contributor"`
	Creator      []string `json:"dcterms:creator"`
	Spatial      []string `json:"dcterms:spatial"`
	Temporal     []string `json:"dcterms:temporal"`
	Subject      []Text   `json:"dcterms:subject"`
	Language     []string `json:"dcterms:language"`
	License      []string `json:"dcterms:license"`
	RightsHolder string   `json:"dcterms:rightsHolder"`
	Rights       string   `json:"dcterms:rights"`
	Type         []string `json:"dcterms:type"`
}

type Schema struct {
	Creator     []Contributor     `json:"schema:creator"`
	Contributor []Contributor     `json:"schema:contributor"`
	Publisher   []Contributor     `json:"schema:publisher"`
	Height      QuantitativeValue `json:"schema:height"`
	Width       QuantitativeValue `json:"schema:width"`
	Depth       QuantitativeValue `json:"schema:depth"`
	Weight      QuantitativeValue `json:"schema:weight"`
	ArtMedium   []Text            `json:"schema:artMedium"`
	ArtForm     []Text            `json:"schema:artForm"`
	IsPartOf    []any             `json:"schema:isPartOf"`
}

type Text struct {
	Value string `json:"@value"`
	Lang  string `json:"@language"`
}

type Contributor struct {
	RoleName  string `json:"@roleName"`
	Name      string `json:"schema:name"`
	BirthDate string `json:"schema:birthDate"`
	DeathDate string `json:"schmea:deathDate"`
}

type QuantitativeValue struct {
	Value    float64 `json:"schema:value"`
	UnitCode string  `json:"schema:unitCode"`
	UnitText string  `json:"schema:unitText"`
}

type CreativeWork struct {
	Name     string         `json:"schema:name"`
	Position int            `json:"schema:position"`
	HasPart  []CreativeWork `json:"schema:hasPart"`
}

func Decode(r io.Reader) *Description {
	bts, _ := io.ReadAll(r)

	// TODO create a set of mutators that gets iterated over.
	//   mutators are passed as a configuration to the decoder.
	//   a mutator can make specific changes to the description
	//   e.g. setting Text/@language to "nl" on an item in an array
	//   per the Meemoo spec.
	var description *Description

	if err := json.Unmarshal(bts, &description); err != nil {
		panic(err)
	}

	return description
}
