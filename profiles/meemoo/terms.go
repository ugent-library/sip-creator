// Package meemoo holds Meemoo SIP 1.2's basic content profile,
// "meemoo/basic". Its descriptive metadata is the dc+schema document:
// Dublin Core terms plus schema.org properties in the Meemoo namespace,
// with the profile's required, cardinality and language rules. Definition
// holds the profile's other values.
package meemoo

import "github.com/ugent-library/sip-creator/sip"

// Terms is the descriptive metadata of meemoo/basic: an ordered list of
// terms in Meemoo's dc+schema model. Each term is keyed by the element it
// states, named as Meemoo's specification names it ("dcterms:title",
// "dcterms:created", "schema:artMedium"). dc+schema.xml keeps the order
// the producer gave. Validate holds the rules on what a term may say.
type Terms []sip.Term

func (t Terms) has(element string) bool {
	for _, term := range t {
		if term.Key == element {
			return true
		}
	}
	return false
}

// localIdentifier returns the value of the identifier term: the producer's
// local catalogue or inventory number. It returns an empty string when
// there is no identifier term.
func (t Terms) localIdentifier() string {
	for _, term := range t {
		if term.Key == identifierElement {
			return term.Value
		}
	}
	return ""
}

// setObjectIdentifier replaces the identifier term's value in place. Terms
// without an identifier stay as they are.
func (t Terms) setObjectIdentifier(id string) {
	for i, term := range t {
		if term.Key == identifierElement {
			t[i].Value = id
			return
		}
	}
}
