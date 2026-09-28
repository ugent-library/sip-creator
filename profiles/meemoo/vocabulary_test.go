package meemoo

import "testing"

// The key to element mapping the input specification documents (§7):
// flat keys, camel-cased elements where the vocabulary spells them so,
// and nothing outside the table. Keys are exact: case folding is the rows
// file's convention, not the vocabulary's.
func TestElementName(t *testing.T) {
	tests := []struct {
		key     string
		element string // "" means the key must be refused
	}{
		{"identifier", "dcterms:identifier"},
		{"ispartof", "dcterms:isPartOf"},  // flat key, camel-cased element
		{"artmedium", "schema:artMedium"}, // schema.org, camel-cased element
		{"artform", "schema:artform"},     // schema.org spells this one flat
		{"abstract", "dcterms:abstract"},
		{"Title", ""},         // keys are lowercase
		{"titel", ""},         // typo
		{"accrualpolicy", ""}, // DCMI term outside the profile
		{"dcterms:title", ""}, // an element name is not a key
	}
	for _, tt := range tests {
		element, err := elementName(tt.key)
		if tt.element == "" {
			if err == nil {
				t.Errorf("elementName(%q) = %q, want refused", tt.key, element)
			}
			continue
		}
		if err != nil || element != tt.element {
			t.Errorf("elementName(%q) = %q, %v; want %q", tt.key, element, err, tt.element)
		}
	}
}

// The table's own invariant: unique keys, or the lookup index would
// silently drop rows, and unique elements, or two keys would emit the
// same element.
func TestVocabularyTable(t *testing.T) {
	if len(vocabularyByKey) != len(vocabulary) {
		t.Fatalf("duplicate keys in the vocabulary: %d rows, %d keys", len(vocabulary), len(vocabularyByKey))
	}
	elements := map[string]bool{}
	for _, row := range vocabulary {
		if elements[row.Element] {
			t.Errorf("element %s is emitted by two keys", row.Element)
		}
		elements[row.Element] = true
	}
}
