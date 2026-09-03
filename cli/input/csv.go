package input

import (
	"bytes"
	"encoding/csv"
	"os"
	"unicode/utf8"
)

// openCSV reads src and returns a CSV reader over its content, applying the
// encoding rules every CSV in the input convention shares: the file must be
// UTF-8, a leading BOM is accepted and dropped (spreadsheet tools produce
// one), and row width is left for the caller to check per row, for a better
// message. Returns ok=false when the file is unreadable or not UTF-8, with
// the violation recorded.
func (r *reader) openCSV(src string) (cr *csv.Reader, ok bool) {
	rel := r.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil, false
	}
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	if !utf8.Valid(data) {
		r.violate("%s: not valid UTF-8; re-export the file as UTF-8", rel)
		return nil, false
	}

	cr = csv.NewReader(bytes.NewReader(data))
	cr.FieldsPerRecord = -1
	return cr, true
}
