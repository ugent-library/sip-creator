package sip

// Description is the decoded descriptive metadata of an entity or a
// representation, in whichever descriptive standard the profile writes.
// Each standard's encoder package supplies its own implementation
// (dcschema.Terms, dc.Terms), and the profile checks the concrete type at
// build time, so nothing in sip/ depends on an encoder.
type Description interface {
	// LocalIdentifier returns the producer's own identifier for the
	// described work, or "" when the description carries none.
	LocalIdentifier() string
	// Validate returns why the description is not a valid one in its
	// standard: an unknown element, a malformed language tag, an empty
	// value, or a rule of the standard itself such as a cardinality limit.
	Validate() error
	// ValidateRequired reports each of the named vocabulary keys the
	// description does not state. Which keys a package must state is
	// profile data, so the set arrives as an argument: plain keys such as
	// identifier and title, which every standard resolves through its own
	// table. A key the table does not know is reported too, because it can
	// only be a mistake in a profile definition.
	ValidateRequired(keys ...string) error
}
