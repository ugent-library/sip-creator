package input

import "github.com/ugent-library/sip-creator/sip"

// Vocabulary gives the rows of a description.csv their meaning under one
// profile: which keys exist, what each one says, and how the rows become
// the profile's description. The --profile flag picks it. One
// implementation per profile lives in a package beside this one and
// imports that profile's package; this package imports none, so the
// reader is the same for every profile.
type Vocabulary interface {
	// Description builds the profile's description from the rows of one
	// level, in file order, and reports each row it cannot place as a
	// Finding naming the row's line. It returns a description even when it
	// reports findings and even for no rows at all, so the reader can run
	// the description's own rules (Validate, ValidateRequired) and report
	// everything in one pass. The statements of the returned description
	// keep the rows' order, so a *sip.TermError from Validate names the row
	// at that position. items are the rows of the package's items.csv,
	// nil when the level has none; a vocabulary with no place for copies
	// reports them.
	Description(rows []Row, items []ItemRow) (sip.Description, []Finding)
}

// Row is one statement decoded from a description.csv: the key and the
// language tag parsed from the first column, the value of the second, and
// the line it was read from.
type Row struct {
	// Key is the plain vocabulary key as the first column spells it,
	// lowercased, without the language tag.
	Key string
	// Lang is the language tag written in square brackets after the key;
	// empty when the row has none.
	Lang string
	// Value is the second column as written.
	Value string
	// Line is the row's line in the file, counted from one.
	Line int
}

// ItemRow is one row of an items.csv: one physical copy of the described
// record, and the line it was read from.
type ItemRow struct {
	// CallNumber is the copy's call number; every row states one.
	CallNumber string
	// Barcode is the copy's barcode; empty when the row has none.
	Barcode string
	// Enumeration is the copy's volume or issue designation; empty when the
	// row has none.
	Enumeration string
	// Line is the row's line in the file, counted from one.
	Line int
}

// Finding is one thing a Vocabulary could not place: the line it concerns
// and what is wrong.
type Finding struct {
	// Line is the row's line in the file, counted from one; zero for a
	// finding about the file as a whole.
	Line int
	// Err says what is wrong.
	Err error
}
