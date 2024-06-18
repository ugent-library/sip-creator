package metadata

type DublinCore struct {
	Title        Text
	Alternative  []string `json:"alternative"`
	Identifier   string
	Extent       string   `json:"extent"`
	Available    string   `json:"available"`
	Description  Text     `json:"description,omitempty"`
	Abstract     Text     `json:"abstract"`
	Created      string   `json:"created"`
	Issued       string   `json:"issued"`
	Publisher    []string `json:"publisher"`
	Contributor  []string `json:"contributor"`
	Creator      []string `json:"creator"`
	Spatial      []string `json:"spatial"`
	Temporal     []string `json:"temporal"`
	Subject      []Text   `json:"subject"`
	Language     []string `json:"language"`
	License      []string `json:"license"`
	RightsHolder string   `json:"rightsHolder"`
	Rights       string   `json:"rights"`
	Type         []string `json:"type"`
}

type Text struct {
	Value string `json:"value"`
	Lang  string `json:"lang"`
}
