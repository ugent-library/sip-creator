package metadata

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
// Required and Repeat rules, the xsi:type the element carries, and the
// Simple DC parent the element dumbs down to.
type vocabularyRow struct {
	Key      string      // plain metadata.csv key
	Element  string      // emitted element name
	Required bool        // required by meemoo's basic profile (enforced per family)
	Repeat   cardinality // meemoo basic profile cardinality (enforced per family)
	XSIType  string      // xsi:type on the emitted element; "" for none
	SimpleDC string      // Simple DC parent for dumb-down; "" for no home
}

// vocabulary is the closed set of supported descriptive elements: the
// elements of meemoo's SIP 1.2 basic content profile that fit a single
// key,value row, in the input specification's table order. This table is
// the metadata model: the CSV decoder, validation, and the templates all
// read from it (ADR-0011). The Required and Repeat columns come from
// meemoo's profile; whether they are enforced is the profile family's
// call. The SimpleDC column follows the "Subproperty Of" relations in the
// DCMI Metadata Terms spec; elements without a parent among the fifteen
// Simple DC elements (rightsHolder, schema:*) carry "", because inventing
// a mapping would assert semantics DCMI doesn't.
var vocabulary = []vocabularyRow{
	{"identifier", "dcterms:identifier", true, once, "", "identifier"},
	{"title", "dcterms:title", true, oncePerLanguage, "", "title"},
	{"description", "dcterms:description", true, oncePerLanguage, "", "description"},
	{"created", "dcterms:created", true, once, "edtf:EDTF-level1", "date"},
	{"alternative", "dcterms:alternative", false, many, "", "title"},
	{"abstract", "dcterms:abstract", false, oncePerLanguage, "", "description"},
	{"creator", "dcterms:creator", false, many, "", "creator"},
	{"contributor", "dcterms:contributor", false, many, "", "contributor"},
	{"publisher", "dcterms:publisher", false, many, "", "publisher"},
	{"issued", "dcterms:issued", false, once, "edtf:EDTF-level1", "date"},
	{"available", "dcterms:available", false, once, "", "date"},
	{"subject", "dcterms:subject", false, many, "", "subject"},
	{"spatial", "dcterms:spatial", false, many, "", "coverage"},
	{"temporal", "dcterms:temporal", false, many, "", "coverage"},
	{"extent", "dcterms:extent", false, once, "", "format"},
	{"language", "dcterms:language", false, many, "", "language"},
	{"type", "dcterms:type", false, many, "", "type"},
	{"ispartof", "dcterms:isPartOf", false, many, "", "relation"},
	{"license", "dcterms:license", false, many, "", "rights"},
	{"rights", "dcterms:rights", false, oncePerLanguage, "", "rights"},
	{"rightsholder", "dcterms:rightsHolder", false, once, "", ""},
	{"artmedium", "schema:artMedium", false, many, "", ""},
	{"artform", "schema:artform", false, many, "", ""},
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

// ResolveKey maps a plain metadata.csv key onto the element
// it generates. Keys are case-insensitive per the convention.
func ResolveKey(key string) (element string, ok bool) {
	row, ok := vocabularyByKey[strings.ToLower(key)]
	return row.Element, ok
}

// RequiredElements lists the elements the vocabulary flags as required, in
// table order. Whether they are enforced is the profile family's call:
// the meemoo family reads this list; a family with its own requiredness
// rules (eark) declares its own.
func RequiredElements() []string {
	var out []string
	for _, row := range vocabulary {
		if row.Required {
			out = append(out, row.Element)
		}
	}
	return out
}
