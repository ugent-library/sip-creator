package meemoo

// cardinality says how often an element may occur in one descriptive
// document. Meemoo counts lang-tagged elements per language:
// oncePerLanguage allows a title in Dutch plus one in English, but not two
// in Dutch. The zero value is many so an element outside the table never
// trips a false repeat error.
type cardinality int

const (
	many cardinality = iota
	once
	oncePerLanguage
)

// elementRow is one element of Meemoo's basic content profile that a term
// can state: its name, Meemoo's cardinality limit, and the xsi:type it
// carries.
type elementRow struct {
	Element string      // qualified name, as Meemoo's specification writes it
	Repeat  cardinality // Meemoo basic profile cardinality
	XSIType string      // xsi:type on the emitted element; "" for none
}

// elements is the closed set of elements Terms may state: the elements of
// Meemoo's SIP 1.2 basic content profile that one term can express (one
// value, an optional language), in the input specification's table order.
// Validation and the template both read it (ADR-0011): Validate refuses an
// element outside it, and the template writes only names it lists. The
// Repeat column is Meemoo's upper cardinality limit, enforced by Validate.
// Which elements a package-level description must state is the required
// list below, enforced by ValidateRequired.
var elements = []elementRow{
	{"dcterms:identifier", once, ""},
	{"dcterms:title", oncePerLanguage, ""},
	{"dcterms:description", oncePerLanguage, ""},
	{"dcterms:created", once, "edtf:EDTF-level1"},
	{"dcterms:alternative", many, ""},
	{"dcterms:abstract", oncePerLanguage, ""},
	{"dcterms:creator", many, ""},
	{"dcterms:contributor", many, ""},
	{"dcterms:publisher", many, ""},
	{"dcterms:issued", once, "edtf:EDTF-level1"},
	{"dcterms:available", once, ""},
	{"dcterms:subject", many, ""},
	{"dcterms:spatial", many, ""},
	{"dcterms:temporal", many, ""},
	{"dcterms:extent", once, ""},
	{"dcterms:language", many, ""},
	{"dcterms:type", many, ""},
	{"dcterms:isPartOf", many, ""},
	{"dcterms:license", many, ""},
	{"dcterms:rights", oncePerLanguage, ""},
	{"dcterms:rightsHolder", once, ""},
	{"schema:artMedium", many, ""},
	{"schema:artform", many, ""},
}

// identifierElement is the element holding the producer's local
// identifier, which the swap replaces with the entity identifier.
const identifierElement = "dcterms:identifier"

// required lists the elements a package-level description must state:
// Meemoo's basic content profile requires an identifier, a title, a
// description and a creation date (Meemoo SIP 1.2, basic profile). The
// first two are also the identity every package states whatever the
// profile (input specification §3): the identifier is what the swap
// overwrites and lifts onto the entity as MEEMOO-LOCAL-ID, the title the
// one name every consumer shows. A representation's description need not
// state any of them.
var required = []string{identifierElement, "dcterms:title", "dcterms:description", "dcterms:created"}

var elementsByName = func() map[string]elementRow {
	byName := make(map[string]elementRow, len(elements))
	for _, row := range elements {
		byName[row.Element] = row
	}
	return byName
}()
