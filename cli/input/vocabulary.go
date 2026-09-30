package input

import (
	"fmt"

	"github.com/ugent-library/sip-creator/sip"
)

// Vocabulary gives the statements of a description.csv their meaning under
// one profile: which keys exist, what each one says, and how the
// statements become the profile's description. The --profile flag picks
// it. One implementation per profile lives in a package beside this one
// and imports that profile's package; this package imports none, so the
// reader is the same for every profile.
type Vocabulary interface {
	// Description builds the profile's description from the statements of
	// one level, in file order, and reports each statement it cannot place
	// as a *StatementError naming the row's line; an error that is not
	// about one statement concerns the file as a whole. It returns a
	// description even when it reports errors and even for no statements
	// at all, so the reader can run the description's own rules (Validate,
	// ValidateRequired) and report everything in one pass. Where the
	// description is a list of terms it keeps the statements' order, so a
	// *sip.TermError from Validate names the statement at that position.
	Description(statements []Statement) (sip.Description, []error)
}

// Statement is one row of a description.csv: one thing the folder states
// about the described entity, as a key, an optional language tag and a
// value, with the line it was read from. The reader checks the row's
// syntax only; what the key means is the vocabulary's.
type Statement struct {
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

// StatementError is a finding about one statement of a description.csv:
// which row, by line, and what is wrong with it. It is the CLI's
// counterpart of sip.TermError, which names a term by position.
type StatementError struct {
	// Line is the row's line in the file, counted from one.
	Line int
	// Err says what is wrong with the statement.
	Err error
}

func (e *StatementError) Error() string { return fmt.Sprintf("line %d: %v", e.Line, e.Err) }

func (e *StatementError) Unwrap() error { return e.Err }
