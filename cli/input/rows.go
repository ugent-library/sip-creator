package input

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/ugent-library/sip-creator/encoders/dc"
)

// decodeDescriptive decodes the one descriptive rows file at a level of the
// input folder. The package level needs one, a representation may have
// none, two at one level is a violation, and so is a second vocabulary
// anywhere in the folder: the profile chosen at create reads one.
func (r *reader) decodeDescriptive(dir string, files []rowsFile, packageLevel bool) dc.Terms {
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
	r.noteStandard(f)
	return r.decodeRows(f.src, packageLevel)
}

// noteStandard records the vocabulary of the first rows file met and
// reports any later file in another one.
func (r *reader) noteStandard(f rowsFile) {
	if r.standard == "" {
		r.standard, r.standardFile = f.standard, r.rel(f.src)
		return
	}
	if r.standard != f.standard {
		r.violate("%s is %s but %s is %s; every descriptive rows file in one input folder must be in the same vocabulary (input specification §3)", r.rel(f.src), f.standard, r.standardFile, r.standard)
	}
}

// decodeRows decodes one descriptive rows file (dcschema.csv or dc.csv)
// into ordered terms, collecting a violation per broken rule. The
// package-level file requires identifier and title; a representation-level
// one does not.
func (r *reader) decodeRows(src string, packageLevel bool) dc.Terms {
	rel := r.rel(src)

	cr, ok := r.openCSV(src)
	if !ok {
		return nil
	}

	var terms dc.Terms
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
		key, value := row[0], row[1]

		element, lang, ok := r.parseKey(rel, line, key)
		if !ok {
			continue
		}
		// What a term may say (vocabulary, language tag, non-empty value)
		// is the library's rule, the same one an embedding caller hits;
		// the decoder only adds the file/line context.
		term := dc.Term{Element: element, Lang: lang, Value: value}
		if err := term.Validate(); err != nil {
			r.violate("%s line %d: %v", rel, line, err)
			continue
		}
		terms = append(terms, term)
	}

	// The convention's own cardinality rule: single-valued and
	// per-language keys must not repeat, whatever the profile. The rule is
	// the same for every rows file, so check needs no configuration. It
	// is a cross-row rule checked on the finished list: each finding names
	// the element and language, which locates the rows in a keyed file.
	if err := terms.ValidateCardinality(); err != nil {
		if joined, ok := err.(interface{ Unwrap() []error }); ok {
			for _, finding := range joined.Unwrap() {
				r.violate("%s: %v", rel, finding)
			}
		} else {
			r.violate("%s: %v", rel, err)
		}
	}

	if packageLevel {
		if !terms.Has("dcterms:identifier") {
			r.violate("%s: identifier is missing; the local catalog or inventory number is required", rel)
		}
		if !terms.Has("dcterms:title") {
			r.violate("%s: title is missing; a title is required", rel)
		}
	}
	return terms
}

func isHeaderRow(row []string) bool {
	return len(row) == 2 &&
		strings.EqualFold(strings.TrimSpace(row[0]), "key") &&
		strings.EqualFold(strings.TrimSpace(row[1]), "value")
}

// parseKey handles the key *syntax* of the CSV convention (the optional
// [lang] bracket and the plain-key spellings of the descriptive
// vocabulary) and returns the element name the key maps onto. Whether the
// language tag inside the brackets is *valid* is dc.Term.Validate's
// rule; the decoder only adds the file/line context.
func (r *reader) parseKey(file string, line int, raw string) (element, lang string, ok bool) {
	key := raw
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
	element, known := dc.ResolveKey(key)
	if !known {
		r.violate("%s line %d: unknown key %q: a typo would silently drop metadata; see the supported keys in the input specification", file, line, raw)
		return "", "", false
	}
	return element, lang, true
}
