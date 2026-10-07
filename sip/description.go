package sip

import "fmt"

// Description is the descriptive metadata of an entity or a
// representation, in the descriptive standard the profile writes. Each
// metadata model supplies its own type: meemoo.Terms and simpledc.Terms are
// lists of Term values, mods.Record is a struct typed by field, and
// build.EncodedDescription is a finished document supplied as a file, for
// the profiles that accept one. Only the model's package works with the
// concrete type, so nothing in sip/ depends on a profile.
type Description interface {
	// Validate returns every way the description is not a valid one in its
	// standard, joined into one error: an unknown key, a malformed
	// language tag, an empty value, or a rule of the standard itself such
	// as a cardinality limit. A finding about one term is a *TermError, so
	// code that decoded the terms from rows, such as the CLI's input
	// reader, can point at the row. It
	// is the one contract a description must meet before it is written;
	// the encoders trust it.
	Validate() error
	// ValidateRequired reports each key the standard requires of a
	// package-level description that this one does not state, such as an
	// identifier and a title. A supplied document states what its schema
	// requires and reports nothing here. A representation's description
	// need not state them, which is why Validate does not include this
	// check.
	ValidateRequired() error
}

// Term is one key, an optional language tag, and a value: one element of a
// description in a flat metadata model (Dublin Core calls its elements
// terms), and the shape of one line of a description.csv. Every model that
// is a flat list of elements shares this shape; the profile decides which
// elements exist and what rules apply.
type Term struct {
	// Key names what the term states. In a description it is the
	// element's name in the model, as the standard spells it:
	// "dcterms:title" in Meemoo's model, "title" in Simple Dublin Core.
	// Read from a description.csv it is the key the row spells, until the
	// profile's mapping turns it into an element.
	Key string
	// Lang is the value's language tag; empty when unspecified.
	Lang string
	// Value is the term's text.
	Value string
}

// TermError is a finding about one term of a description: which term, by
// position, and what is wrong with it.
type TermError struct {
	// Index is the term's position in the description, counted from zero.
	Index int
	// Err says what is wrong with the term.
	Err error
}

func (e *TermError) Error() string { return fmt.Sprintf("term %d: %v", e.Index+1, e.Err) }

func (e *TermError) Unwrap() error { return e.Err }
