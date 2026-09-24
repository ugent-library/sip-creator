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
// Required and Repeat rules, and the xsi:type the element carries.
type vocabularyRow struct {
	Key      string      // plain key in the CSV rows file
	Element  string      // emitted element name
	Required bool        // required by meemoo's basic profile
	Repeat   cardinality // meemoo basic profile cardinality
	XSIType  string      // xsi:type on the emitted element; "" for none
}

// vocabulary is the closed set of supported descriptive elements: the
// elements of meemoo's SIP 1.2 basic content profile that fit a single
// key,value row, in the input specification's table order. This table is
// the metadata model: the CSV decoder, validation, and the template all
// read from it (ADR-0011). The Required and Repeat columns come from
// meemoo's profile; the cardinality limits are enforced by Validate, the
// required set by the profile through ValidateRequired.
var vocabulary = []vocabularyRow{
	{"identifier", "dcterms:identifier", true, once, ""},
	{"title", "dcterms:title", true, oncePerLanguage, ""},
	{"description", "dcterms:description", true, oncePerLanguage, ""},
	{"created", "dcterms:created", true, once, "edtf:EDTF-level1"},
	{"alternative", "dcterms:alternative", false, many, ""},
	{"abstract", "dcterms:abstract", false, oncePerLanguage, ""},
	{"creator", "dcterms:creator", false, many, ""},
	{"contributor", "dcterms:contributor", false, many, ""},
	{"publisher", "dcterms:publisher", false, many, ""},
	{"issued", "dcterms:issued", false, once, "edtf:EDTF-level1"},
	{"available", "dcterms:available", false, once, ""},
	{"subject", "dcterms:subject", false, many, ""},
	{"spatial", "dcterms:spatial", false, many, ""},
	{"temporal", "dcterms:temporal", false, many, ""},
	{"extent", "dcterms:extent", false, once, ""},
	{"language", "dcterms:language", false, many, ""},
	{"type", "dcterms:type", false, many, ""},
	{"ispartof", "dcterms:isPartOf", false, many, ""},
	{"license", "dcterms:license", false, many, ""},
	{"rights", "dcterms:rights", false, oncePerLanguage, ""},
	{"rightsholder", "dcterms:rightsHolder", false, once, ""},
	{"artmedium", "schema:artMedium", false, many, ""},
	{"artform", "schema:artform", false, many, ""},
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

// RequiredElements lists the elements the vocabulary flags as required, in
// table order: meemoo's basic content profile requires them at package
// level, and the profile reads this list.
func RequiredElements() []string {
	var out []string
	for _, row := range vocabulary {
		if row.Required {
			out = append(out, row.Element)
		}
	}
	return out
}
