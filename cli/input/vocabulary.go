package input

import (
	"encoding/xml"
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

// DocumentVocabulary is the optional part of a Vocabulary whose profile
// takes a finished document of its standard in place of the rows, as the
// two eark profiles do. The reader reserves the file name it gives, at the
// input root and inside each representation directory, reads that file's
// root element, and asks whether the root is the profile's. A vocabulary
// without it takes rows only, and no file name is reserved for a document:
// a dc.xml under such a profile is content like any other file.
type DocumentVocabulary interface {
	// DocumentName is the file name of the supplied document, dc.xml or
	// mods.xml: the name the package gives the document too.
	DocumentName() string
	// CheckDocument returns why root, the document's root element, is not
	// the profile's standard: another element or namespace, or a version
	// other than the one the package declares. It is the judgement the
	// engine makes before a build, so check refuses what create would.
	CheckDocument(root xml.StartElement) error
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
