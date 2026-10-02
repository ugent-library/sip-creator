package input

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"unicode/utf8"
)

// newCSVReader returns a CSV reader over data under the rules every CSV in
// the input folder shares: the content must be UTF-8, and a leading BOM is
// dropped (spreadsheet tools write one). Rows may have any width, so the
// parser can name the line and the expected columns of a row that is off.
func newCSVReader(data []byte) (*csv.Reader, error) {
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	if !utf8.Valid(data) {
		return nil, errors.New("not valid UTF-8; re-export the file as UTF-8")
	}
	cr := csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1
	return cr, nil
}

// rowError is a finding about one row of a CSV in the input folder: the
// row's line, counted from one, and what is wrong with it.
type rowError struct {
	line int
	err  error
}

func (e *rowError) Error() string { return fmt.Sprintf("line %d: %v", e.line, e.err) }

func (e *rowError) Unwrap() error { return e.err }
