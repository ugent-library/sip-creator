// Package Meemoo is the basic profile: everything the tool knows about
// Meemoo SIP 1.2's basic content profile. Its descriptive standard is the
// dc+schema document, Dublin Core terms plus schema.org properties in the
// Meemoo namespace, with the profile's required, cardinality and language
// rules; its Definition names the rest as data, and the registry in
// profiles/ hands that out as "basic".
package meemoo

import "github.com/ugent-library/sip-creator/sip"

// Terms is an ordered list of terms in Meemoo's dc+schema model, each
// keyed by the element it states, named as Meemoo's specification names
// it ("dcterms:title", "dcterms:created", "schema:artMedium"); the order
// the producer gave them in is preserved through to the emitted XML. Any
// producer constructs it directly (the CLI's mapping of description.csv
// is one); Validate holds the rules on what a term may say.
type Terms []sip.Term

// has reports whether any term states the given element.
func (t Terms) has(element string) bool {
	for _, term := range t {
		if term.Key == element {
			return true
		}
	}
	return false
}

// localIdentifier returns the value of the identifier term: the producer's
// local catalog/inventory number ("" when absent).
func (t Terms) localIdentifier() string {
	for _, term := range t {
		if term.Key == identifierElement {
			return term.Value
		}
	}
	return ""
}

// setObjectIdentifier replaces the identifier term's value in place (a
// no-op when the terms carry none). Terms holds one identifier slot, so
// the swap overwrites the local identifier: read it with localIdentifier
// first.
func (t Terms) setObjectIdentifier(id string) {
	for i, term := range t {
		if term.Key == identifierElement {
			t[i].Value = id
			return
		}
	}
}
