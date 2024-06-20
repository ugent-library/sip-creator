package structure

type Metadata struct {
	DublinCore DublinCore
	Schema     Schema
}

type DublinCore struct {
	Title        Text     `json:"dcterms:title"`
	Alternative  []string `json:"dcterms:alternative"`
	Identifier   string
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
	Lang  string `json:"@lang"`
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
