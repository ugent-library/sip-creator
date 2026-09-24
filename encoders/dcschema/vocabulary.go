package dcschema

import "strings"

// cardinality says how often a key may occur in one descriptive document.
// meemoo counts lang-tagged elements per language: oncePerLanguage allows
// title[nl] plus title[en], but not two title[nl] rows. The zero value is
// many so an element outside the table never trips a false repeat error.
type cardinality int

const (
	many cardinality = iota
	once
	oncePerLanguage
)

// vocabularyRow is one entry of the descriptive vocabulary. It holds
// everything the tool knows about one key: the element it emits, meemoo's
// cardinality limit, and the xsi:type the element carries.
type vocabularyRow struct {
	Key     string      // plain key in the CSV rows file
	Element string      // emitted element name
	Repeat  cardinality // meemoo basic profile cardinality
	XSIType string      // xsi:type on the emitted element; "" for none
}

// vocabulary is the closed set of supported descriptive elements: the
// elements of meemoo's SIP 1.2 basic content profile that fit a single
// key,value row, in the input specification's table order. This table is
// the metadata model: the CSV decoder, validation, and the template all
// read from it (ADR-0011). The Repeat column is meemoo's upper cardinality
// limit, enforced by Validate. Which keys a package must state is not
// table data: identifier and title are every package's identity, and what
// meemoo's basic profile requires on top is that profile's own data
// (profiles.Definition.RequiredKeys), both checked through
// ValidateRequired.
var vocabulary = []vocabularyRow{
	{"identifier", "dcterms:identifier", once, ""},
	{"title", "dcterms:title", oncePerLanguage, ""},
	{"description", "dcterms:description", oncePerLanguage, ""},
	{"created", "dcterms:created", once, "edtf:EDTF-level1"},
	{"alternative", "dcterms:alternative", many, ""},
	{"abstract", "dcterms:abstract", oncePerLanguage, ""},
	{"creator", "dcterms:creator", many, ""},
	{"contributor", "dcterms:contributor", many, ""},
	{"publisher", "dcterms:publisher", many, ""},
	{"issued", "dcterms:issued", once, "edtf:EDTF-level1"},
	{"available", "dcterms:available", once, ""},
	{"subject", "dcterms:subject", many, ""},
	{"spatial", "dcterms:spatial", many, ""},
	{"temporal", "dcterms:temporal", many, ""},
	{"extent", "dcterms:extent", once, ""},
	{"language", "dcterms:language", many, ""},
	{"type", "dcterms:type", many, ""},
	{"ispartof", "dcterms:isPartOf", many, ""},
	{"license", "dcterms:license", many, ""},
	{"rights", "dcterms:rights", oncePerLanguage, ""},
	{"rightsholder", "dcterms:rightsHolder", once, ""},
	{"artmedium", "schema:artMedium", many, ""},
	{"artform", "schema:artform", many, ""},
}

var (
	vocabularyByKey     = make(map[string]vocabularyRow, len(vocabulary))
	vocabularyByElement = make(map[string]vocabularyRow, len(vocabulary))
)

func init() {
	for _, row := range vocabulary {
		vocabularyByKey[row.Key] = row
		vocabularyByElement[row.Element] = row
	}
}

// ResolveKey maps a plain key from the CSV rows file onto the element
// it generates. Keys are case-insensitive per the convention.
func ResolveKey(key string) (element string, ok bool) {
	row, ok := vocabularyByKey[strings.ToLower(key)]
	return row.Element, ok
}
