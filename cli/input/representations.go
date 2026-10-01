package input

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ugent-library/sip-creator/build"
)

// repRow is one decoded representations.csv data row.
type repRow struct {
	line                int
	folder, label, kind string
}

// applyRepresentations decodes representations.csv and applies it to the
// representations read from representations/: each row names a
// representation folder and supplies its label and type. The file is strict when present
// (input-spec.md): every row must match a folder, every folder must be
// covered by a row, and the row order becomes the packaging order. Empty
// cells stay empty: build.SourceRepresentation resolves the defaults.
func (w *folderWalker) applyRepresentations(src string, reps []build.SourceRepresentation) []build.SourceRepresentation {
	rel := w.rel(src)
	rows, decoded := w.decodeRepresentations(src)
	if !decoded {
		return reps
	}
	if len(rows) == 0 {
		w.violate("%s: the file has no rows; list every representation folder, or delete the file", rel)
		return reps
	}

	byName := make(map[string]int, len(reps))
	for i, rep := range reps {
		byName[rep.Name] = i
	}

	covered := make(map[string]int, len(rows)) // folder → line of its row
	var ordered []build.SourceRepresentation
	for _, row := range rows {
		if prev, ok := covered[row.folder]; ok {
			w.violate("%s line %d: folder %q already has a row (line %d)", rel, row.line, row.folder, prev)
			continue
		}
		covered[row.folder] = row.line
		i, ok := byName[row.folder]
		if !ok {
			w.violate("%s line %d: there is no folder representations/%s; every row must name an existing representation folder", rel, row.line, row.folder)
			continue
		}
		rep := reps[i]
		rep.Label = row.label
		rep.Type = row.kind
		ordered = append(ordered, rep)
	}

	// A folder the file does not cover must fail loudly: skipping it
	// would silently drop content from the package.
	for _, rep := range reps {
		if _, ok := covered[rep.Name]; !ok {
			w.violate("representations/%s is not listed in %s; add a row for it, or remove the folder", rep.Name, rel)
			ordered = append(ordered, rep)
		}
	}
	return ordered
}

// decodeRepresentations reads the representations.csv at src into rows and
// records a violation per broken rule. Returns decoded=false when the file
// cannot be used at all; a usable file with no data rows returns an empty
// slice.
func (w *folderWalker) decodeRepresentations(src string) (rows []repRow, decoded bool) {
	rel := w.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		w.violate("%s: %v", rel, err)
		return nil, false
	}

	rows, errs, err := parseRepresentationRows(data)
	if err != nil {
		for _, e := range flatten(err) {
			w.violate("%s: %v", rel, e)
		}
		return nil, false
	}
	for _, e := range errs {
		if re, ok := errors.AsType[*rowError](e); ok {
			w.violate("%s line %d: %v", rel, re.line, re.err)
			continue
		}
		w.violate("%s: %v", rel, e)
	}
	return rows, true
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

// rowError is a finding about one row of a representations.csv.
type rowError struct {
	line int
	err  error
}

func (e *rowError) Error() string { return fmt.Sprintf("line %d: %v", e.line, e.err) }

func (e *rowError) Unwrap() error { return e.err }
