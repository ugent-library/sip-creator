package vocabulary

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/sip"
)

// EarkMods is the eark-mods profile's vocabulary: the MODS keys the input
// specification lists and where each one goes in the record. The record is
// typed by field (ADR-0021), so the key table lives here, not in the
// profile package. A statement that breaks a rule on its own (an unknown
// key, a language tag where none is taken, an empty value, a repeat) is an
// error at its line and is not placed. The rows carry flat statements
// only: a record's copies reach the package through the library's record
// or a supplied mods.xml.
type EarkMods struct{}

var _ input.DocumentFormat = EarkMods{}

// placement is what the vocabulary knows about one key: where its value
// goes in the record, whether the key takes a language tag, and how often
// it may occur.
type placement struct {
	fill      func(*earkmods.Record, input.Statement)
	takesLang bool
	occurs    cardinality
}

// cardinality says how often a key may occur in one description.csv.
type cardinality int

const (
	once            cardinality = iota // one row
	oncePerLanguage                    // one row per language tag
)

var modsKeys = map[string]placement{
	"identifier": {
		fill:   func(r *earkmods.Record, s input.Statement) { r.Identifier = s.Value },
		occurs: once,
	},
	"title": {
		fill: func(r *earkmods.Record, s input.Statement) {
			r.Titles = append(r.Titles, earkmods.Title{Value: s.Value, Lang: s.Lang})
		},
		takesLang: true,
		occurs:    oncePerLanguage,
	},
}

// Description fills a record from the statements in order, reporting each
// statement it cannot place at its line.
func (EarkMods) Description(statements []input.Statement) (sip.Description, []error) {
	var record earkmods.Record
	var errs []error
	first := map[string]int{} // key, or key and language, → line of the first statement placed
	for _, s := range statements {
		key, ok := modsKeys[s.Key]
		if !ok {
			errs = append(errs, &input.StatementError{Line: s.Line, Err: fmt.Errorf("unknown key %q: not in the MODS vocabulary; see the supported keys in the input specification", s.Key)})
			continue
		}
		if s.Lang != "" && !key.takesLang {
			errs = append(errs, &input.StatementError{Line: s.Line, Err: fmt.Errorf("%s takes no language tag", s.Key)})
			continue
		}
		if strings.TrimSpace(s.Value) == "" {
			errs = append(errs, &input.StatementError{Line: s.Line, Err: fmt.Errorf("%s has an empty value", s.Key)})
			continue
		}
		// "\x00" cannot appear in a key, so per-language entries never
		// collide with the plain key entries.
		entry := s.Key
		if key.occurs == oncePerLanguage {
			entry += "\x00" + s.Lang
		}
		if line, seen := first[entry]; seen {
			var err error
			switch {
			case key.occurs == once:
				err = fmt.Errorf("%s appears more than once (first on line %d); give exactly one value", s.Key, line)
			case s.Lang == "":
				err = fmt.Errorf("%s appears more than once (first on line %d); repeat it only with distinct language tags (%s[nl], %s[en])", s.Key, line, s.Key, s.Key)
			default:
				err = fmt.Errorf("%s appears more than once in language %q (first on line %d); give one value per language", s.Key, s.Lang, line)
			}
			errs = append(errs, &input.StatementError{Line: s.Line, Err: err})
			continue
		}
		first[entry] = s.Line
		key.fill(&record, s)
	}
	return record, errs
}

// DocumentName is the file name of a supplied MODS document: mods.xml, the
// name the package gives the document.
func (EarkMods) DocumentName() string {
	return earkmods.Definition.DocumentName
}

// ValidateDocumentRoot returns why root is not a mods:mods document
// declaring MODS 3.7, as the eark-mods profile's metadata model judges it.
func (EarkMods) ValidateDocumentRoot(root xml.StartElement) error {
	return validateDocumentRoot(earkmods.Definition, root)
}
