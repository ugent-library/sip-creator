package input

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// description returns the level's description from the one source the
// folder supplies for it: the rows file, mapped by the profile's mapper, or
// the profile's supplied document, read as it is. Both at one level is a
// violation: an entity has one description, and the tool does not pick.
// The package level needs one; a representation may have neither.
// rowsPath and documentPath are "" when the folder has no such file.
func (w *folderWalker) description(rowsPath, documentPath string, packageLevel bool) sip.Description {
	switch {
	case rowsPath != "" && documentPath != "":
		described := "representation"
		if packageLevel {
			described = "package"
		}
		w.violate("%s and %s are both present; describe the %s with one of the two, not both (input specification §3)", w.rel(rowsPath), w.rel(documentPath), described)
		return nil
	case documentPath != "":
		return w.readDocument(documentPath)
	case rowsPath != "":
		return w.decodeDescription(rowsPath, packageLevel)
	case packageLevel:
		w.violateMissingDescription()
	}
	return nil
}

// violateMissingDescription records that the package level describes
// nothing, naming the file or files the profile accepts.
func (w *folderWalker) violateMissingDescription() {
	if w.documentFormat == nil {
		w.violate("descriptive rows are missing: every package folder needs a description.csv describing the content (input specification §3)")
		return
	}
	w.violate("descriptive metadata is missing: every package folder needs a description.csv or a %s describing the content (input specification §3)", w.documentName)
}

// decodeDescription decodes the description.csv at src into the profile's
// description and records a violation per broken rule: the row syntax,
// the mapper's placement of each term, and the description's own
// rules, with ValidateRequired at the package level only.
func (w *folderWalker) decodeDescription(src string, packageLevel bool) sip.Description {
	rel := w.rel(src)

	data, err := os.ReadFile(src)
	if err != nil {
		w.violate("%s: %v", rel, err)
		return nil
	}

	terms, lines, errs, err := parseTerms(data)
	if err != nil {
		w.violate("%s: %v", rel, err)
		return nil
	}
	description, mapErrs := w.mapper.Map(terms)
	errs = append(errs, mapErrs...)
	errs = append(errs, flatten(description.Validate())...)
	if packageLevel {
		errs = append(errs, flatten(description.ValidateRequired())...)
	}

	// A finding about one row is reported at the row's line: the parser
	// names the line of a row that did not become a term, and the
	// mapper and the description's rules name a term by its index,
	// which lines turns back into a line. A cross-row finding names the
	// key and language, which locates the rows in a keyed file.
	for _, err := range errs {
		if re, ok := errors.AsType[*rowError](err); ok {
			w.violate("%s line %d: %v", rel, re.line, re.err)
			continue
		}
		if te, ok := errors.AsType[*sip.TermError](err); ok && te.Index < len(lines) {
			w.violate("%s line %d: %v", rel, lines[te.Index], te.Err)
			continue
		}
		w.violate("%s: %v", rel, err)
	}
	if len(terms) == 0 {
		return nil
	}
	return description
}

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
