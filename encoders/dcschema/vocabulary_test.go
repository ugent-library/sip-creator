package dcschema

import "testing"

func TestResolveKey(t *testing.T) {
	tests := []struct {
		key     string
		element string // "" means the key must be unknown
	}{
		{"identifier", "dcterms:identifier"},
		{"ispartof", "dcterms:isPartOf"},  // flat key, camel-cased element
		{"artmedium", "schema:artMedium"}, // schema.org, camel-cased element
		{"artform", "schema:artform"},     // schema.org spells this one flat
		{"abstract", "dcterms:abstract"},
		{"Title", "dcterms:title"}, // keys are case-insensitive
		{"titel", ""},              // typo
		{"accrualpolicy", ""},      // DCMI term outside the profile
		{"dcterms:title", ""},      // prefixed keys are not supported
	}
	for _, tt := range tests {
		element, ok := ResolveKey(tt.key)
		if tt.element == "" {
			if ok {
				t.Errorf("ResolveKey(%q) resolved to %q, want unknown", tt.key, element)
			}
			continue
		}
		if !ok || element != tt.element {
			t.Errorf("ResolveKey(%q) = %q, %v; want %q", tt.key, element, ok, tt.element)
		}
	}
}

// The table's own invariant: unique keys and elements, or the two lookup
// indexes would silently drop rows.
func TestVocabularyTable(t *testing.T) {
	if len(vocabularyByKey) != len(vocabulary) || len(vocabularyByElement) != len(vocabulary) {
		t.Fatalf("duplicate keys or elements in the vocabulary: %d rows, %d keys, %d elements",
			len(vocabulary), len(vocabularyByKey), len(vocabularyByElement))
	}
}
