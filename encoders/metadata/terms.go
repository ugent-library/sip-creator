package metadata

// Term is one descriptive statement: an element name from the descriptive
// vocabulary, an optional language tag, and the value.
type Term struct {
	// Element is the vocabulary element name, e.g. "dcterms:title".
	Element string
	// Lang is the xml:lang value; empty when unspecified.
	Lang string
	// Value is the term's text.
	Value string
}

// Terms is an ordered list of descriptive statements; the order the
// producer stated them in is preserved through to the emitted XML.
// Any producer constructs it directly (the CLI's metadata.csv decoder is
// one); Validate holds the rules on what a term may say, and Encode
// refuses invalid terms.
type Terms []Term

// Has reports whether any term states the given element.
func (t Terms) Has(element string) bool {
	for _, term := range t {
		if term.Element == element {
			return true
		}
	}
	return false
}

// LocalIdentifier returns the value of the dcterms:identifier term: the
// producer's local catalog/inventory number ("" when absent).
func (t Terms) LocalIdentifier() string {
	for _, term := range t {
		if term.Element == "dcterms:identifier" {
			return term.Value
		}
	}
	return ""
}

// SetObjectIdentifier replaces the dcterms:identifier term's value in
// place (a no-op when the terms carry none). Terms holds one identifier
// slot, so the swap overwrites the local identifier: read it with
// LocalIdentifier first.
func (t Terms) SetObjectIdentifier(id string) {
	for i, term := range t {
		if term.Element == "dcterms:identifier" {
			t[i].Value = id
			return
		}
	}
}
