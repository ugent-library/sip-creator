package input

import (
	"errors"
	"io"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// decodeDescription decodes the descriptive rows file at src into the profile's
// description, collecting a violation per broken rule. src is "" when the
// level has no rows file: the package level needs one, a representation
// may have none. The row syntax (header, two columns, key[lang]) is the
// file's own; what a term may say, the cross-row rules and what a
// package-level description must state are the profile's, run once on the
// finished list. The library runs the same methods again as the contract
// before a build; these calls report, so check and create agree.
func (d *directory) decodeDescription(src string, packageLevel bool) sip.Description {
	if src == "" {
		if packageLevel {
			d.violate("descriptive rows are missing: every package folder needs a description.csv describing the content (input specification §3)")
		}
		return nil
	}
	rel := d.rel(src)

	cr, ok := d.openCSV(src)
	if !ok {
		return nil
	}

	var terms []sip.Term
	var lines []int // lines[i] is the line terms[i] was read from
	headerSeen := false
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// The reader may not recover its position after a syntax
			// error; report it (the csv error names the line) and stop.
			d.violate("%s: %v", rel, err)
			break
		}
		line, _ := cr.FieldPos(0)

		if !headerSeen {
			headerSeen = true
			if isHeaderRow(row) {
				continue
			}
			// A missing header is a violation, but the row itself may be
			// data; keep decoding so its findings surface too.
			d.violate(`%s: the first row must be the header "key,value"`, rel)
		}

		if len(row) != 2 {
			d.violate("%s line %d: expected exactly two columns (key,value), got %d", rel, line, len(row))
			continue
		}

		key, lang, ok := d.parseKey(rel, line, row[0])
		if !ok {
			continue
		}
		terms = append(terms, sip.Term{Key: key, Lang: lang, Value: row[1]})
		lines = append(lines, line)
	}

	// A finding about one term names it by position, which the lines
	// gathered above turn back into the row's line; a cross-row finding
	// names the key and language, which locates the rows in a keyed file.
	description := d.profile.Encoder.NewDescription(terms)
	errs := findings(description.Validate())
	if packageLevel {
		errs = append(errs, findings(description.ValidateRequired())...)
	}
	for _, err := range errs {
		var te *sip.TermError
		if errors.As(err, &te) {
			d.violate("%s line %d: %v", rel, lines[te.Index], te.Err)
			continue
		}
		d.violate("%s: %v", rel, err)
	}
	if len(terms) == 0 {
		return nil
	}
	return description
}

// findings flattens a joined error into its parts, so each finding is
// reported as its own violation.
func findings(err error) []error {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}
	var out []error
	for _, e := range joined.Unwrap() {
		out = append(out, findings(e)...)
	}
	return out
}

func isHeaderRow(row []string) bool {
	return len(row) == 2 &&
		strings.EqualFold(strings.TrimSpace(row[0]), "key") &&
		strings.EqualFold(strings.TrimSpace(row[1]), "value")
}

// parseKey handles the key *syntax* of the CSV convention (the optional
// [lang] bracket, no prefixes, case-insensitive spelling) and returns the
// plain key for the vocabulary to check. Whether the language tag inside
// the brackets is *valid* is the term's own rule; the decoder only adds
// the file/line context.
func (d *directory) parseKey(file string, line int, raw string) (key, lang string, ok bool) {
	key = raw
	if i := strings.IndexByte(key, '['); i >= 0 {
		if !strings.HasSuffix(key, "]") {
			d.violate("%s line %d: malformed language tag in %q; write it like title[nl]", file, line, raw)
			return "", "", false
		}
		lang = key[i+1 : len(key)-1]
		key = key[:i]
		if lang == "" {
			d.violate("%s line %d: malformed language tag in %q; write it like title[nl]", file, line, raw)
			return "", "", false
		}
	}

	// Prefixed keys are not supported: every supported element has
	// a plain key, so point the operator at the spelling table instead of
	// a generic unknown-key message.
	if strings.Contains(key, ":") {
		d.violate("%s line %d: prefixed keys like %q are not supported: every element has a plain key; see the supported keys in the input specification", file, line, raw)
		return "", "", false
	}
	return strings.ToLower(key), lang, true
}
