package input

import (
	"errors"
	"io"
	"strings"

	"github.com/ugent-library/sip-creator/profiles"
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
// covered by a row, and the row order becomes the packaging order. Defaults
// are not applied here: empty cells stay empty, and the library resolves the
// name → label → type cascade, so the CLI and an embedding caller get
// identical behavior.
func (r *reader) applyRepresentations(src string, reps []Representation) []Representation {
	rel := r.rel(src)
	rows, decoded := r.decodeRepresentations(src)
	if !decoded {
		return reps
	}
	if len(rows) == 0 {
		r.violate("%s: the file has no rows; list every representation directory, or delete the file", rel)
		return reps
	}

	byName := make(map[string]int, len(reps))
	for i, rep := range reps {
		byName[rep.Name] = i
	}

	covered := make(map[string]int, len(rows)) // directory → line of its row
	var ordered []Representation
	for _, row := range rows {
		if prev, ok := covered[row.dir]; ok {
			r.violate("%s line %d: directory %q already has a row (line %d)", rel, row.line, row.dir, prev)
			continue
		}
		covered[row.dir] = row.line
		i, ok := byName[row.dir]
		if !ok {
			r.violate("%s line %d: there is no directory representations/%s; every row must name an existing representation directory", rel, row.line, row.dir)
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
			r.violate("representations/%s is not listed in %s; add a row for it, or remove the directory", rep.Name, rel)
			ordered = append(ordered, rep)
		}
	}
	return ordered
}

// decodeRepresentations reads the file into rows, checking the file-local
// rules: a header with a directory column (label and type optional, nothing
// else), consistent row width, a directory in every row, and label/type
// values that are safe to emit as METS attributes. Returns decoded=false
// when the file itself could not be decoded (violations recorded); a
// decoded file with no data rows returns an empty slice.
func (r *reader) decodeRepresentations(src string) (rows []repRow, decoded bool) {
	rel := r.rel(src)

	cr, ok := r.openCSV(src)
	if !ok {
		return nil, false
	}

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		r.violate(`%s: the file is empty; the first row must be the header "directory,label,type"`, rel)
		return nil, false
	}
	if err != nil {
		r.violate("%s: %v", rel, err)
		return nil, false
	}

	// Columns are matched by header name, not position; an unknown header
	// is a violation because a typo would silently drop a column.
	dirCol, labelCol, typeCol := -1, -1, -1
	headerOK := true
	for i, h := range header {
		var col *int
		switch strings.ToLower(strings.TrimSpace(h)) {
		case "directory":
			col = &dirCol
		case "label":
			col = &labelCol
		case "type":
			col = &typeCol
		default:
			r.violate("%s: unknown column %q in the header; the columns are directory, label, type", rel, h)
			headerOK = false
			continue
		}
		if *col >= 0 {
			r.violate("%s: the header names column %q twice", rel, strings.TrimSpace(h))
			headerOK = false
			continue
		}
		*col = i
	}
	if dirCol < 0 && headerOK {
		r.violate(`%s: the header has no directory column; the first row must be a header like "directory,label,type"`, rel)
		headerOK = false
	}
	if !headerOK {
		return nil, false
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
			// The reader may not recover its position after a syntax
			// error; report it (the csv error names the line) and stop.
			r.violate("%s: %v", rel, err)
			break
		}
		line, _ := cr.FieldPos(0)

		if len(row) != len(header) {
			r.violate("%s line %d: expected %d columns per the header, got %d", rel, line, len(header), len(row))
			continue
		}
		// A trailing space in a directory name is invisible in the file and
		// can never match a portable-charset directory, so trim it away.
		dir := strings.TrimSpace(cell(row, dirCol))
		if dir == "" {
			r.violate("%s line %d: the directory cell is empty; every row must name a representation directory", rel, line)
			continue
		}
		label, kind := cell(row, labelCol), cell(row, typeCol)
		// Whether a value may be emitted is the library's rule, the same
		// one an embedding caller hits; the decoder adds file/line context.
		if err := profiles.ValidateAttributeText(label); err != nil {
			r.violate("%s line %d: label: %v", rel, line, err)
		}
		if err := profiles.ValidateAttributeText(kind); err != nil {
			r.violate("%s line %d: type: %v", rel, line, err)
		}
		// Keep the row even when a value is bad: matching and coverage
		// findings should still surface (collect-all).
		rows = append(rows, repRow{line: line, dir: dir, label: label, kind: kind})
	}
	return rows, true
}
