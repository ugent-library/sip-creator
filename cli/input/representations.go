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
// header naming the columns, then one row per representation folder. errs
// holds the findings in the rows. A row that breaks a rule is reported as
// a *rowError and left out. A row whose label or type XML cannot carry is
// the exception: it is reported and still returned, so matching rows to
// folders can report on it too. A CSV syntax error ends the parse, because
// the CSV reader may not find its place again. err is set only when the
// file cannot be used: it is not UTF-8, it is empty, or its header is
// wrong. All the header's problems are joined into err.
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
		// Spaces around a folder name are invisible in the file, and a
		// representation name may not contain a space, so they are trimmed.
		folder := strings.TrimSpace(cell(row, cols.folder))
		if folder == "" {
			errs = append(errs, &rowError{line, errors.New("the folder cell is empty; every row must name a representation folder")})
			continue
		}
		label, kind := cell(row, cols.label), cell(row, cols.kind)
		// The label and type are held to the library's rule for XML text,
		// so they are refused here as in a SourcePackage built in Go.
		if err := build.ValidateXMLText(label); err != nil {
			errs = append(errs, &rowError{line, fmt.Errorf("label: %w", err)})
		}
		if err := build.ValidateXMLText(kind); err != nil {
			errs = append(errs, &rowError{line, fmt.Errorf("type: %w", err)})
		}
		rows = append(rows, repRow{line: line, folder: folder, label: label, kind: kind})
	}
	return rows, errs, nil
}

// repColumns holds the position of each column in a representations.csv
// row. A column the header leaves out has position -1.
type repColumns struct {
	folder, label, kind int
}

// parseRepresentationsHeader finds the columns by name, not position. An
// unknown name is an error, because a typo would silently drop a column.
// It returns the column positions and an error that joins all the
// header's problems.
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
