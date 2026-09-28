package input

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
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
	r.noteStandard(f)
	return r.decodeRows(f, packageLevel)
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

// rowsBuilder is the vocabulary side of decoding one rows file. Each
// descriptive standard resolves keys and validates terms against its own
// table; the row syntax (header, two columns, key[lang]) is shared.
type rowsBuilder interface {
	// add resolves one key and appends the term, or returns why the row is
	// refused.
	add(key, lang, value string) error
	// finish validates the finished list against the standard's own rules
	// and, at package level, what the standard requires a package
	// description to state. It returns the description (nil when there are
	// no terms) and every finding.
	finish(packageLevel bool) (sip.Description, []error)
}

// dcschemaRows builds meemoo dc+schema terms from a dcschema.csv.
type dcschemaRows struct{ terms meemoo.Terms }

func (b *dcschemaRows) add(key, lang, value string) error {
	element, ok := meemoo.ResolveKey(key)
	if !ok {
		return unknownKey(key)
	}
	// What a term may say (vocabulary, language tag, non-empty value) is
	// the library's rule, the same one an embedding caller hits; the
	// decoder runs it per row only to add the file/line context, and drops
	// a refused row so the finished list is not reported twice.
	term := meemoo.Term{Element: element, Lang: lang, Value: value}
	if err := term.Validate(); err != nil {
		return err
	}
	b.terms = append(b.terms, term)
	return nil
}

func (b *dcschemaRows) finish(packageLevel bool) (sip.Description, []error) {
	errs := findings(b.terms.Validate())
	if packageLevel {
		errs = append(errs, findings(b.terms.ValidateRequired())...)
	}
	if len(b.terms) == 0 {
		return nil, errs
	}
	return b.terms, errs
}

// dcRows builds Simple Dublin Core terms from a dc.csv.
type dcRows struct{ terms eark.Terms }

func (b *dcRows) add(key, lang, value string) error {
	element, ok := eark.ResolveKey(key)
	if !ok {
		return unknownKey(key)
	}
	term := eark.Term{Element: element, Lang: lang, Value: value}
	if err := term.Validate(); err != nil {
		return err
	}
	b.terms = append(b.terms, term)
	return nil
}

func (b *dcRows) finish(packageLevel bool) (sip.Description, []error) {
	errs := findings(b.terms.Validate())
	if packageLevel {
		errs = append(errs, findings(b.terms.ValidateRequired())...)
	}
	if len(b.terms) == 0 {
		return nil, errs
	}
	return b.terms, errs
}

// unknownKey says why a key no table lists is refused: a typo must not
// silently drop metadata.
func unknownKey(key string) error {
	return fmt.Errorf("unknown key %q: a typo would silently drop metadata; see the supported keys in the input specification", key)
}

// findings flattens a joined error into its parts, so each cross-row
// finding (a cardinality limit, a missing Dutch entry, a missing identity
// key) is reported as its own violation.
func findings(err error) []error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()
	}
	return []error{err}
}

// decodeRows decodes one descriptive rows file into the description its
// vocabulary produces, collecting a violation per broken rule. The
// package-level file requires identifier and title; a representation-level
// one does not.
func (r *reader) decodeRows(f rowsFile, packageLevel bool) sip.Description {
	rel := r.rel(f.src)

	cr, ok := r.openCSV(f.src)
	if !ok {
		return nil
	}

	var b rowsBuilder
	switch f.standard {
	case dcStandard:
		b = &dcRows{}
	default:
		b = &dcschemaRows{}
	}

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
		if err := b.add(key, lang, row[1]); err != nil {
			r.violate("%s line %d: %v", rel, line, err)
		}
	}

	// The standard's cross-row rules (cardinality, required language, one
	// identifier) and the identity every package states are checked on the
	// finished list: each finding names the key or element and language,
	// which locates the rows in a keyed file. These calls report; the
	// library runs the same methods again as the contract before a build,
	// so check and create agree.
	d, errs := b.finish(packageLevel)
	for _, err := range errs {
		r.violate("%s: %v", rel, err)
	}
	return d
}

func isHeaderRow(row []string) bool {
	return len(row) == 2 &&
		strings.EqualFold(strings.TrimSpace(row[0]), "key") &&
		strings.EqualFold(strings.TrimSpace(row[1]), "value")
}

// parseKey handles the key *syntax* of the CSV convention (the optional
// [lang] bracket, no prefixes) and returns the plain key for the
// vocabulary to resolve. Whether the language tag inside the brackets is
// *valid* is the term's own rule; the decoder only adds the file/line
// context.
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
	return key, lang, true
}
