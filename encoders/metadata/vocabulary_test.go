package metadata

import (
	"slices"
	"testing"
)

// The Required marks are a spec contract: meemoo's basic content profile
// mandates exactly these four elements. The basic profile derives its
// required list from the table, so an accidental table edit would silently
// change conformance — this pins it.
func TestRequiredElements(t *testing.T) {
	want := []string{"dcterms:identifier", "dcterms:title", "dcterms:description", "dcterms:created"}
	if got := RequiredElements(); !slices.Equal(got, want) {
		t.Errorf("RequiredElements() = %v, want %v", got, want)
	}
}

func TestResolveKey(t *testing.T) {
	tests := []struct {
		key     string
		element string // "" means the key must be unknown
	}{
		{"identifier", "dcterms:identifier"},
		{"ispartof", "dcterms:isPartOf"},  // renamed spelling
		{"artmedium", "schema:artMedium"}, // schema.org, camel-cased element
		{"artform", "schema:artform"},     // schema.org spells this one flat
		{"abstract", "dcterms:abstract"},  // extended beyond the spec §3 origin
		{"Title", "dcterms:title"},        // keys are case-insensitive
		{"titel", ""},                     // typo
		{"accrualpolicy", ""},             // DCMI term outside the profile
		{"dcterms:title", ""},             // prefixed keys left the convention
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

// The table's own invariants: unique keys and elements (the two lookup
// indexes must not silently drop rows), and Simple DC parents drawn from
// the fifteen Simple DC elements.
func TestVocabularyTable(t *testing.T) {
	if len(vocabularyByKey) != len(vocabulary) || len(vocabularyByElement) != len(vocabulary) {
		t.Fatalf("duplicate keys or elements in the vocabulary: %d rows, %d keys, %d elements",
			len(vocabulary), len(vocabularyByKey), len(vocabularyByElement))
	}
	simpleDC := map[string]bool{
		"contributor": true, "coverage": true, "creator": true, "date": true,
		"description": true, "format": true, "identifier": true,
		"language": true, "publisher": true, "relation": true, "rights": true,
		"source": true, "subject": true, "title": true, "type": true,
	}
	for _, row := range vocabulary {
		if row.SimpleDC != "" && !simpleDC[row.SimpleDC] {
			t.Errorf("%s dumbs down to %q, which is not a Simple DC element", row.Element, row.SimpleDC)
		}
	}
}
