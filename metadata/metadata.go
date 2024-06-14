package metadata

type DublinCore struct {
	Identifier  string `json:"identifier"`
	Title       string `json:"string"`
	Description string `json:"description"`
	Created     string `json:"created"`
}
