package input

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ugent-library/sip-creator/build"
)

// repRow is one decoded representations.csv data row.
type repRow struct {
	line                int
	folder, label, kind string
}

// parseRepresentationRows parses the content of a representations.csv: a
// header naming the columns, then one row per representation folder.
// The errs are findings in the rows: a *rowError for a row that breaks a
// rule, or a CSV syntax error, which ends the parse because the reader may
// not find its place again. A row with a bad label or type is still
// returned, so matching rows to folders can report on it too. err
// means the file cannot be used: not UTF-8, empty, or a wrong header, whose
// problems are joined into it.
func parseRepresentationRows(data []byte) (rows []repRow, errs []error, err error) {
	cr, err := newCSVReader(data)
	if err != nil {
		return nil, nil, err
	}

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New(`the file is empty; the first row must be the header "folder,label,type"`)
	}
	if err != nil {
		return nil, nil, err
	}
	cols, err := parseRepresentationsHeader(header)
	if err != nil {
		return nil, nil, err
	}

	cell := func(row []string, col int) string {
		if col < 0 {
			return ""
		}
		return row[col]
	}

	rows = []repRow{}
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			errs = append(errs, err) // the csv error names the line
			break
		}
		line, _ := cr.FieldPos(0)

		if len(row) != len(header) {
			errs = append(errs, &rowError{line, fmt.Errorf("expected %d columns per the header, got %d", len(header), len(row))})
			continue
		}
		// A trailing space in a folder name is invisible in the file and
		// can never match a portable-charset folder name, so trim it away.
		folder := strings.TrimSpace(cell(row, cols.folder))
		if folder == "" {
			errs = append(errs, &rowError{line, errors.New("the folder cell is empty; every row must name a representation folder")})
			continue
		}
		label, kind := cell(row, cols.label), cell(row, cols.kind)
		// Whether a value may be emitted is the library's rule, so a label
		// is refused here the same way as in a SourcePackage built in Go.
		if err := build.ValidateAttributeText(label); err != nil {
			errs = append(errs, &rowError{line, fmt.Errorf("label: %w", err)})
		}
		if err := build.ValidateAttributeText(kind); err != nil {
			errs = append(errs, &rowError{line, fmt.Errorf("type: %w", err)})
		}
		rows = append(rows, repRow{line: line, folder: folder, label: label, kind: kind})
	}
	return rows, errs, nil
}

// repColumns holds the position of each column in a representations.csv
// row; -1 for a column the header leaves out.
type repColumns struct {
	folder, label, kind int
}

// parseRepresentationsHeader finds the columns by name, not position. An
// unknown name is an error, because a typo would silently drop a column.
// All of the header's problems are joined into the error.
func parseRepresentationsHeader(header []string) (repColumns, error) {
	cols := repColumns{folder: -1, label: -1, kind: -1}
	var errs []error
	for i, h := range header {
		var col *int
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "folder":
			col = &cols.folder
		case "label":
			col = &cols.label
		case "type":
			col = &cols.kind
		default:
			errs = append(errs, fmt.Errorf("unknown column %q in the header; the columns are folder, label, type", h))
			continue
		}
		if *col >= 0 {
			errs = append(errs, fmt.Errorf("the header names column %q twice", strings.TrimSpace(h)))
			continue
		}
		*col = i
	}
	if cols.folder < 0 && len(errs) == 0 {
		errs = append(errs, errors.New(`the header has no folder column; the first row must be a header like "folder,label,type"`))
	}
	return cols, errors.Join(errs...)
}
