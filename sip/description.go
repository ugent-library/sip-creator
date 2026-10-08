package sip

import "fmt"

// Description is the descriptive metadata of an entity or a
// representation, in the descriptive standard the profile writes: a
// metadata model the profile's encoder renders, or a finished document
// supplied as a file.
type Description interface {
	// Validate checks the description against the rules of its standard:
	// known keys, well-formed language tags, values that are not empty, and
	// rules of the standard itself such as a cardinality limit. It returns
	// every finding, joined into one error. A finding about one term is a
	// *TermError, so a caller that decoded the terms from rows can point at
	// the row. The encoders check nothing further: a description that
	// passes Validate is written as it is.
	Validate() error
	// ValidateRequired checks that the description states every key its
	// standard requires of a package-level description, such as an
	// identifier and a title. It returns an error naming each missing key.
	// The check is separate from Validate because a representation's
	// description need not state these keys.
	ValidateRequired() error
}

// Term is one key, an optional language tag, and a value: one element of a
// description in a flat metadata model (Dublin Core calls its elements
// terms). The description type that holds the terms decides which keys
// exist and what rules apply.
type Term struct {
	// Key names the element the term states, as the metadata model spells it.
	Key string
	// Lang is the value's language tag, or empty when the value has none.
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
