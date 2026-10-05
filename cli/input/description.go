package input

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// parseTerms parses the content of a description.csv: a "key,value"
// header, then one term per row of two columns, with lines[i] the line
// terms[i] was read from, counted from one. The errs are findings in the
// rows: a row that breaks the syntax is reported as a *rowError and left
// out, and the rows after it are still read; a CSV syntax error ends the
// parse, because the reader may not find its place again. err means the
// content cannot be read as CSV at all.
func parseTerms(data []byte) (terms []sip.Term, lines []int, errs []error, err error) {
	cr, err := newCSVReader(data)
	if err != nil {
		return nil, nil, nil, err
	}

	headerSeen := false
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

		if !headerSeen {
			headerSeen = true
			if isHeaderRow(row) {
				continue
			}
			// The first row may be data that lacks a header above it;
			// keep it, so its own findings are reported too.
			errs = append(errs, errors.New(`the first row must be the header "key,value"`))
		}

		if len(row) != 2 {
			errs = append(errs, &rowError{line: line, err: fmt.Errorf("expected exactly two columns (key,value), got %d", len(row))})
			continue
		}

		key, lang, err := parseKey(row[0])
		if err != nil {
			errs = append(errs, &rowError{line: line, err: err})
			continue
		}
		terms = append(terms, sip.Term{Key: key, Lang: lang, Value: row[1]})
		lines = append(lines, line)
	}
	return terms, lines, errs, nil
}

func isHeaderRow(row []string) bool {
	return len(row) == 2 &&
		strings.EqualFold(strings.TrimSpace(row[0]), "key") &&
		strings.EqualFold(strings.TrimSpace(row[1]), "value")
}

// parseKey splits a key cell into the plain key, lowercased, and the
// language tag in its optional brackets. Whether the tag is a valid
// language tag is a rule on the term, which Validate checks.
func parseKey(raw string) (key, lang string, err error) {
	key = raw
	if i := strings.IndexByte(key, '['); i >= 0 {
		if !strings.HasSuffix(key, "]") {
			return "", "", fmt.Errorf("malformed language tag in %q; write it like title[nl]", raw)
		}
		lang = key[i+1 : len(key)-1]
		key = key[:i]
		if lang == "" {
			return "", "", fmt.Errorf("malformed language tag in %q; write it like title[nl]", raw)
		}
	}

	// Every supported element has a plain key, so a prefixed key points
	// the operator at the key table instead of a generic unknown-key
	// message.
	if strings.Contains(key, ":") {
		return "", "", fmt.Errorf("prefixed keys like %q are not supported: every element has a plain key; see the supported keys in the input specification", raw)
	}
	return strings.ToLower(key), lang, nil
}

// flatten splits a joined error into its parts, so each finding is
// reported as its own violation.
func flatten(err error) []error {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}
	var out []error
	for _, e := range joined.Unwrap() {
		out = append(out, flatten(e)...)
	}
	return out
}
