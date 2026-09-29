// Package earkmods is the eark-mods profile: a plain E-ARK SIP for
// RODA-class repositories whose descriptive world is MODS 3.7. A package
// describes one bibliographic record, the statements about the work plus
// the physical copies the library holds of it, and the vocabulary maps
// each plain key onto one complete MODS element (ADR-0015).
package earkmods

import "github.com/ugent-library/sip-creator/sip"

// Record is the descriptive metadata of one bibliographic record in MODS
// 3.7: the statements about the work, and the physical copies of it. It is
// what a package describes; a representation describes a version of the
// same content and states terms alone.
type Record struct {
	// Terms are the statements about the work, each keyed by a plain key of
	// the MODS vocabulary ("identifier", "title"); the order the producer
	// stated them in is preserved through to the emitted XML.
	Terms []sip.Term
	// Items are the physical copies of the record, one per copy. Items
	// describe the package level only (ADR-0015): a copy is never a
	// representation, so a representation's record carries none.
	Items []Item
}

// Item is one physical copy of a bibliographic record: where it is shelved,
// how it is identified, and which part of a multi-part work it is.
type Item struct {
	// CallNumber is the copy's shelf mark, the one value every item states.
	CallNumber string
	// Barcode is the copy's item barcode; empty when the copy has none.
	// Unique across a record's items when set.
	Barcode string
	// Enumeration is the volume or issue designation of a copy of a
	// journal, periodical or newspaper; empty for a single-part work.
	Enumeration string
}
