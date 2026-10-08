package meemoo

import "testing"

// elementName admits exactly the elements the table lists, spelled
// as Meemoo's specification spells them: camel case where it writes one,
// the prefix always, and nothing outside the profile. A key of
// description.csv is not an element.
func TestElementName(t *testing.T) {
	tests := []struct {
		element string
		ok      bool
	}{
		{"dcterms:identifier", true},
		{"dcterms:isPartOf", true},       // camel-cased element
		{"schema:artMedium", true},       // schema.org, camel-cased element
		{"schema:artform", true},         // schema.org spells this one flat
		{"dcterms:ispartof", false},      // element names are exact
		{"isPartOf", false},              // the prefix is part of the name
		{"dcterms:accrualPolicy", false}, // DCMI term outside the profile
		{"title", false},                 // a key of description.csv is not an element
	}
	for _, tt := range tests {
		got, err := elementName(tt.element)
		if !tt.ok {
			if err == nil {
				t.Errorf("elementName(%q) = %q, want refused", tt.element, got)
			}
			continue
		}
		if err != nil || got != tt.element {
			t.Errorf("elementName(%q) = %q, %v; want it admitted", tt.element, got, err)
		}
	}
}

// Every element appears in the table once. A second row for an element
// would be dropped from the lookup index without an error.
func TestElementsTable(t *testing.T) {
	if len(elementsByName) != len(elements) {
		t.Fatalf("duplicate elements in the table: %d rows, %d elements", len(elements), len(elementsByName))
	}
}
