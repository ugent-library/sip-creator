package meemoo

import "testing"

// The template's guard admits exactly the elements the table lists, spelled
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

// The table's own invariant: unique elements, or the lookup index would
// silently drop rows.
func TestElementsTable(t *testing.T) {
	if len(elementsByName) != len(elements) {
		t.Fatalf("duplicate elements in the table: %d rows, %d elements", len(elements), len(elementsByName))
	}
}
