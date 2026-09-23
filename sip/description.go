package sip

// Description is the decoded descriptive metadata of an entity or a
// representation, in whichever descriptive standard the profile's family
// speaks. Each standard's encoder package supplies its own implementation
// (dc.Terms today), and the family that owns the standard checks the
// concrete type at build time, so nothing in sip/ depends on an encoder.
type Description interface {
	// LocalIdentifier returns the producer's own identifier for the
	// described work, or "" when the description carries none.
	LocalIdentifier() string
	// Validate returns why the description cannot be encoded: an unknown
	// element, a malformed language tag, an empty value.
	Validate() error
}
