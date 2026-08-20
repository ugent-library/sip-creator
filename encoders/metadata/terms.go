package metadata

// Term is one descriptive statement: an element name from the descriptive
// vocabulary, an optional language tag, and the value.
type Term struct {
	Element string // vocabulary element name, e.g. "dcterms:title"
	Lang    string // xml:lang value; empty when unspecified
	Value   string
}

// Terms is an ordered list of descriptive statements; the order the
// producer stated them in is preserved through to the emitted XML.
//
// Terms is plainly constructible by design: any producer builds it
// directly — the CLI's metadata.csv decoder is one, an embedding system
// mapping its own records is another (the input-convention plan: the file
// is one transport, not the API). Whatever the producer, Validate holds
// the rules on what a term may say; Encode refuses invalid terms.
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

// LocalIdentifier returns the value of the dcterms:identifier term —
// the producer's local catalog/inventory number ("" when absent or the
// scheme is unknown).
func (t Terms) LocalIdentifier(scheme string) string {
	if scheme != "dcterms" {
		return ""
	}
	for _, term := range t {
		if term.Element == "dcterms:identifier" {
			return term.Value
		}
	}
	return ""
}

// SetObjectIdentifier replaces the dcterms:identifier term's value in
// place (a no-op when the terms carry none). Unlike Description, Terms
// keeps no second identifier slot: read the local identifier with
// LocalIdentifier before swapping in the object identifier. What to do
// with each identifier is assembly policy, not data behavior.
func (t Terms) SetObjectIdentifier(id string) {
	for i, term := range t {
		if term.Element == "dcterms:identifier" {
			t[i].Value = id
			return
		}
	}
}
