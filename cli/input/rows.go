package input

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// decodeDescriptive decodes the one descriptive rows file at a level of the
// input folder. The package level needs one, a representation may have
// none, two at one level is a violation, and so is a second vocabulary
// anywhere in the folder: the profile chosen at create reads one.
func (r *reader) decodeDescriptive(dir string, files []rowsFile, packageLevel bool) sip.Description {
	switch {
	case len(files) == 0:
		if packageLevel {
			r.violate("descriptive rows are missing: every package folder needs a dcschema.csv (meemoo profiles) or a dc.csv (plain E-ARK) describing the content (input specification §3)")
		}
		return nil
	case len(files) > 1:
		r.violate("%s: %s and %s are both present; a folder holds one descriptive rows file per level (input specification §3)", r.rel(dir), filepath.Base(files[0].src), filepath.Base(files[1].src))
		return nil
	}
	f := files[0]
	r.noteVocabulary(f)
	return r.decodeRows(f, packageLevel)
}

// noteVocabulary records the vocabulary of the first rows file met and
// reports any later file in another one.
func (r *reader) noteVocabulary(f rowsFile) {
	if r.vocabulary == nil {
		r.vocabulary, r.vocabularyFile = f.vocabulary, r.rel(f.src)
		return
	}
	if r.vocabulary != f.vocabulary {
		r.violate("%s is %s but %s is %s; every descriptive rows file in one input folder must be in the same vocabulary (input specification §3)", r.rel(f.src), f.vocabulary.name, r.vocabularyFile, r.vocabulary.name)
	}
}

// decodeRows decodes one descriptive rows file into the description its
// vocabulary produces, collecting a violation per broken rule. The row
// syntax (header, two columns, key[lang]) is the file's own; what a term
// may say, the cross-row rules and what a package-level description must
// state are the vocabulary's, run once on the finished list. The library
// runs the same methods again as the contract before a build; these calls
// report, so check and create agree.
func (r *reader) decodeRows(f rowsFile, packageLevel bool) sip.Description {
	rel := r.rel(f.src)

	cr, ok := r.openCSV(f.src)
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
			r.violate("%s: %v", rel, err)
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
			r.violate(`%s: the first row must be the header "key,value"`, rel)
		}

		if len(row) != 2 {
			r.violate("%s line %d: expected exactly two columns (key,value), got %d", rel, line, len(row))
			continue
		}

		key, lang, ok := r.parseKey(rel, line, row[0])
		if !ok {
			continue
		}
		terms = append(terms, sip.Term{Key: key, Lang: lang, Value: row[1]})
		lines = append(lines, line)
	}

	// A finding about one term names it by position, which the lines
	// gathered above turn back into the row's line; a cross-row finding
	// names the key and language, which locates the rows in a keyed file.
	d := f.vocabulary.terms(terms)
	errs := findings(d.Validate())
	if packageLevel {
		errs = append(errs, findings(d.ValidateRequired())...)
	}
	for _, err := range errs {
		var te *sip.TermError
		if errors.As(err, &te) {
			r.violate("%s line %d: %v", rel, lines[te.Index], te.Err)
			continue
		}
		r.violate("%s: %v", rel, err)
	}
	if len(terms) == 0 {
		return nil
	}
	return d
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
func (r *reader) parseKey(file string, line int, raw string) (key, lang string, ok bool) {
	key = raw
	if i := strings.IndexByte(key, '['); i >= 0 {
		if !strings.HasSuffix(key, "]") {
			r.violate("%s line %d: malformed language tag in %q; write it like title[nl]", file, line, raw)
			return "", "", false
		}
		lang = key[i+1 : len(key)-1]
		key = key[:i]
		if lang == "" {
			r.violate("%s line %d: malformed language tag in %q; write it like title[nl]", file, line, raw)
			return "", "", false
		}
	}

	// Prefixed keys are not supported: every supported element has
	// a plain key, so point the operator at the spelling table instead of
	// a generic unknown-key message.
	if strings.Contains(key, ":") {
		r.violate("%s line %d: prefixed keys like %q are not supported: every element has a plain key; see the supported keys in the input specification", file, line, raw)
		return "", "", false
	}
	return strings.ToLower(key), lang, true
}
