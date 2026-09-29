package earkmods

// mmsIDType is the type attribute on the mods:identifier the identifier
// key emits: the record's catalogue number, an Alma MMS ID at UGent
// Library. MODS leaves the type vocabulary open; "local" is the value its
// own list suggests for a system-internal identifier. The owner of the
// repository side settles the final value (eark-mods plan, open question),
// and this constant is the one place it changes.
const mmsIDType = "local"

// vocabularyRow is one entry of the MODS vocabulary: the key a term
// states, the element it emits, and the fixed attribute value that element
// carries.
type vocabularyRow struct {
	Key     string // the key a term states, as the input specification lists it
	Element string // the MODS element the key emits, rendered complete by the template
	Type    string // the value of the element's type attribute; "" where it has none
}

// vocabulary is the closed set of supported MODS keys (ADR-0011). One key
// emits one complete element with fixed attributes (ADR-0015): the row
// names the element, and the template renders it whole, subelements
// included (titleInfo holds its title). Roles, title types and identifier
// types are rows, never key syntax; a further identifier type is a second
// row emitting identifier with its own Type. The table starts with the two
// columns every record has. Validation and the template both read it: the
// element a key emits is known nowhere else. A further element is a row
// here, a sub-template in the encoder when the element is new, and a line
// in the input specification.
var vocabulary = []vocabularyRow{
	{"identifier", "identifier", mmsIDType},
	{"title", "titleInfo", ""},
}

// required lists the keys a package-level record must state: an
// identifier, the record's catalogue number, which the eark profiles keep
// in the document as the value operators search by (ADR-0012), and a title,
// the one name every consumer shows. A representation's record need not
// state them.
var required = []string{"identifier", "title"}

var vocabularyByKey = func() map[string]vocabularyRow {
	byKey := make(map[string]vocabularyRow, len(vocabulary))
	for _, row := range vocabulary {
		byKey[row.Key] = row
	}
	return byKey
}()
