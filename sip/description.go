package sip

// Description is the decoded descriptive metadata of an entity or a
// representation, in whichever descriptive standard the profile writes.
// Each standard's encoder package supplies its own implementation
// (dcschema.Terms, dc.Terms). The profile's descriptive standard knows the
// concrete type and does everything that needs it, so nothing in sip/
// depends on an encoder.
type Description interface {
	// Validate returns every way the description is not a valid one in its
	// standard, joined into one error: an unknown element, a malformed
	// language tag, an empty value, or a rule of the standard itself such
	// as a cardinality limit. It is the one contract a description must
	// meet before it is written; the encoders trust it.
	Validate() error
	// ValidateRequired reports each element the standard requires of a
	// package-level description that this one does not state: an
	// identifier and a title at least, whatever the standard. A
	// representation's description need not state them, which is why
	// Validate does not include this check.
	ValidateRequired() error
}
