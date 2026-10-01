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
	line             int
	dir, label, kind string
}

// applyRepresentations decodes representations.csv and applies it to the
// representations read from the folder: each row names a directory and
// supplies its label and type. The file is strict when present
// (input-spec.md): every row must match a directory, every directory must be
// covered by a row, and the row order becomes the packaging order. Empty
// cells stay empty: build.SourceRepresentation resolves the defaults.
func (d *directory) applyRepresentations(src string, reps []build.SourceRepresentation) []build.SourceRepresentation {
	rel := d.rel(src)
	rows, decoded := d.decodeRepresentations(src)
	if !decoded {
		return reps
	}
	if len(rows) == 0 {
		d.violate("%s: the file has no rows; list every representation directory, or delete the file", rel)
		return reps
	}

	byName := make(map[string]int, len(reps))
	for i, rep := range reps {
		byName[rep.Name] = i
	}

	covered := make(map[string]int, len(rows)) // directory → line of its row
	var ordered []build.SourceRepresentation
	for _, row := range rows {
		if prev, ok := covered[row.dir]; ok {
			d.violate("%s line %d: directory %q already has a row (line %d)", rel, row.line, row.dir, prev)
			continue
		}
		covered[row.dir] = row.line
		i, ok := byName[row.dir]
		if !ok {
			d.violate("%s line %d: there is no directory representations/%s; every row must name an existing representation directory", rel, row.line, row.dir)
			continue
		}
		rep := reps[i]
		rep.Label = row.label
		rep.Type = row.kind
		ordered = append(ordered, rep)
	}

	// A directory the file does not cover must fail loudly: skipping it
	// would silently drop content from the package.
	for _, rep := range reps {
		if _, ok := covered[rep.Name]; !ok {
			d.violate("representations/%s is not listed in %s; add a row for it, or remove the directory", rep.Name, rel)
			ordered = append(ordered, rep)
		}
	}
	return ordered
}

// decodeRepresentations reads the representations.csv at src into rows and
// records a violation per broken rule. Returns decoded=false when the file
// cannot be used at all; a usable file with no data rows returns an empty
// slice.
func (d *directory) decodeRepresentations(src string) (rows []repRow, decoded bool) {
	rel := d.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		d.violate("%s: %v", rel, err)
		return nil, false
	}

	rows, errs, err := parseRepresentationRows(data)
	if err != nil {
		for _, e := range flatten(err) {
			d.violate("%s: %v", rel, e)
		}
		return nil, false
	}
	for _, e := range errs {
		if re, ok := errors.AsType[*rowError](e); ok {
			d.violate("%s line %d: %v", rel, re.line, re.err)
			continue
		}
		d.violate("%s: %v", rel, e)
	}
	return rows, true
}

// parseRepresentationRows parses the content of a representations.csv: a
// header naming the columns, then one row per representation directory.
// The errs are findings in the rows: a *rowError for a row that breaks a
// rule, or a CSV syntax error, which ends the parse because the reader may
// not find its place again. A row with a bad label or type is still
// returned, so matching rows to directories can report on it too. err
// means the file cannot be used: not UTF-8, empty, or a wrong header, whose
// problems are joined into it.
func parseRepresentationRows(data []byte) (rows []repRow, errs []error, err error) {
	cr, err := newCSVReader(data)
	if err != nil {
		return nil, nil, err
	}

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New(`the file is empty; the first row must be the header "directory,label,type"`)
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
		// A trailing space in a directory name is invisible in the file and
		// can never match a portable-charset directory, so trim it away.
		dir := strings.TrimSpace(cell(row, cols.dir))
		if dir == "" {
			errs = append(errs, &rowError{line, errors.New("the directory cell is empty; every row must name a representation directory")})
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
		rows = append(rows, repRow{line: line, dir: dir, label: label, kind: kind})
	}
	return rows, errs, nil
}

// repColumns holds the position of each column in a representations.csv
// row; -1 for a column the header leaves out.
type repColumns struct {
	dir, label, kind int
}

// parseRepresentationsHeader finds the columns by name, not position. An
// unknown name is an error, because a typo would silently drop a column.
// All of the header's problems are joined into the error.
func parseRepresentationsHeader(header []string) (repColumns, error) {
	cols := repColumns{dir: -1, label: -1, kind: -1}
	var errs []error
	for i, h := range header {
		var col *int
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "directory":
			col = &cols.dir
		case "label":
			col = &cols.label
		case "type":
			col = &cols.kind
		default:
			errs = append(errs, fmt.Errorf("unknown column %q in the header; the columns are directory, label, type", h))
			continue
		}
		if *col >= 0 {
			errs = append(errs, fmt.Errorf("the header names column %q twice", strings.TrimSpace(h)))
			continue
		}
		*col = i
	}
	if cols.dir < 0 && len(errs) == 0 {
		errs = append(errs, errors.New(`the header has no directory column; the first row must be a header like "directory,label,type"`))
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
