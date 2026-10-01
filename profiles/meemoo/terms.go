// Package Meemoo is the basic profile: everything the tool knows about
// Meemoo SIP 1.2's basic content profile. Its descriptive standard is the
// dc+schema document, Dublin Core terms plus schema.org properties in the
// Meemoo namespace, with the profile's required, cardinality and language
// rules; its Definition names the rest as data, and the registry in
// profiles/ hands that out as "basic".
package meemoo

import "github.com/ugent-library/sip-creator/sip"

// Terms is an ordered list of descriptive statements in the dc+schema
// vocabulary, each keyed by the plain key the input specification's table
// lists ("title", "created", "artmedium"); the order the producer stated
// them in is preserved through to the emitted XML. Any producer constructs
// it directly (the CLI's rows-file decoder is one); Validate holds the
// rules on what a term may say.
type Terms []sip.Term

// has reports whether any term states the given key.
func (t Terms) has(key string) bool {
	for _, term := range t {
		if term.Key == key {
			return true
		}
	}
	return false
}

// localIdentifier returns the value of the identifier term: the producer's
// local catalog/inventory number ("" when absent).
func (t Terms) localIdentifier() string {
	for _, term := range t {
		if term.Key == "identifier" {
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
		if term.Key == "identifier" {
			t[i].Value = id
			return
		}
	}
}
