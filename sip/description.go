package sip

import "fmt"

// Description is the descriptive metadata of an entity or a
// representation, in the descriptive standard the profile writes. Each
// profile package supplies its own type: meemoo.Terms and eark.Terms are
// lists of Term values, earkmods.Record is a struct typed by field, and
// build.DescriptiveDocument is a finished document supplied as a file, for
// the profiles that accept one. Only the profile package works with the
// concrete type, so nothing in sip/ depends on a profile.
type Description interface {
	// Validate returns every way the description is not a valid one in its
	// standard, joined into one error: an unknown key, a malformed
	// language tag, an empty value, or a rule of the standard itself such
	// as a cardinality limit. A finding about one term is a *TermError, so
	// a caller that decoded the terms from rows can point at the row. It
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

// Term is one descriptive statement: a key from the profile's vocabulary,
// an optional language tag, and the value. Every descriptive standard
// that is a flat list of elements shares this shape; the profile decides
// which keys exist, which element each one emits, and what rules apply.
type Term struct {
	// Key names what the statement says, spelled as the input
	// specification's table spells it: "title", "created", "artmedium". It is lowercase and carries no prefix; the profile maps
	// it to the element it emits.
	Key string
	// Lang is the value's language tag; empty when unspecified.
	Lang string
	// Value is the statement's text.
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
