package earkmods

import "testing"

// The table is the model: every key once, every element rendered by the
// sub-template of its name, and the required keys among the rows.
func TestVocabulary(t *testing.T) {
	seen := make(map[string]bool, len(vocabulary))
	for _, row := range vocabulary {
		if seen[row.Key] {
			t.Errorf("key %q is listed twice", row.Key)
		}
		seen[row.Key] = true
		if modsTemplate.Lookup(row.Element) == nil {
			t.Errorf("key %q emits %q, which no sub-template renders", row.Key, row.Element)
		}
	}
	for _, key := range required {
		if _, ok := vocabularyByKey[key]; !ok {
			t.Errorf("required key %q is not in the vocabulary", key)
		}
	}
}
