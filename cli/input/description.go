package input

import (
	"errors"
	"io"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// description returns the level's description from the one source the
// folder supplies for it: the rows file, decoded under the vocabulary, or
// the profile's supplied document, read as it is. Both at one level is a
// violation: an entity has one description, and the tool does not pick.
// The package level needs one; a representation may have neither. rows and
// document are "" when the folder has no such file.
func (d *directory) description(rows, document string, packageLevel bool) sip.Description {
	switch {
	case rows != "" && document != "":
		described := "representation"
		if packageLevel {
			described = "package"
		}
		d.violate("%s and %s are both present; describe the %s with one of the two, not both (input specification §3)", d.rel(rows), d.rel(document), described)
		return nil
	case document != "":
		return d.readDocument(document)
	case rows != "":
		return d.decodeDescription(rows, packageLevel)
	case packageLevel:
		d.violateMissingDescription()
	}
	return nil
}

// violateMissingDescription records that the package level describes
// nothing, naming the file or files the profile accepts.
func (d *directory) violateMissingDescription() {
	if d.document == nil {
		d.violate("descriptive rows are missing: every package folder needs a description.csv describing the content (input specification §3)")
		return
	}
	d.violate("descriptive metadata is missing: every package folder needs a description.csv or a %s describing the content (input specification §3)", d.document.DocumentName())
}

// decodeDescription decodes the descriptive rows file at src into the
// profile's description, collecting a violation per broken rule. The row
// syntax (header, two columns, key[lang]) is the file's own and is checked
// here; what a key means is the vocabulary's; what the finished
// description may say and what a package-level one must state are the
// description's own rules, run once on the result. The library runs the
// same methods again as the contract before a build; these calls report,
// so check and create agree.
func (d *directory) decodeDescription(src string, packageLevel bool) sip.Description {
	rel := d.rel(src)

	cr, ok := d.openCSV(src)
	if !ok {
		return nil
	}

	var statements []Statement
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
		statements = append(statements, Statement{Key: key, Lang: lang, Value: row[1], Line: line})
	}

	// A finding about one statement is reported at the row's line: the
	// vocabulary names the line itself, and the description's rules name
	// a term's position, which the statements turn back into a line. A
	// cross-row finding names the key and language, which locates the
	// rows in a keyed file.
	description, errs := d.vocabulary.Description(statements)
	errs = append(errs, flatten(description.Validate())...)
	if packageLevel {
		errs = append(errs, flatten(description.ValidateRequired())...)
	}
	for _, err := range errs {
		var se *StatementError
		var te *sip.TermError
		switch {
		case errors.As(err, &se):
			d.violate("%s line %d: %v", rel, se.Line, se.Err)
		case errors.As(err, &te) && te.Index < len(statements):
			d.violate("%s line %d: %v", rel, statements[te.Index].Line, te.Err)
		default:
			d.violate("%s: %v", rel, err)
		}
	}
	if len(statements) == 0 {
		return nil
	}
	return description
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
