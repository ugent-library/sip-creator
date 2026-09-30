package vocabulary

import (
	"fmt"
	"strings"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/sip"
)

// EarkMods is the eark-mods profile's vocabulary: the MODS keys the input
// specification lists and the place in the record each one fills. The
// record is typed by field (ADR-0021), so the key table lives here, on the
// CLI side, not in the profile package. The rules a statement can break
// on its own (an unknown key, a language on a key that takes none, an
// empty value, a second row for a key that occurs once, a repeated
// language) are errors at the row's line and the statement is not placed;
// the reader runs the record's own rules on the result. The rows carry
// flat statements only: a record's copies, and any other structure, reach
// the package through the library's record or a supplied mods.xml.
type EarkMods struct{}

// placement is what the vocabulary knows about one key: where its value
// goes in the record, whether the key takes a language tag, and how often
// it may occur.
type placement struct {
	fill   func(*earkmods.Record, input.Statement)
	lang   bool
	repeat cardinality
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
		repeat: once,
	},
	"title": {
		fill: func(r *earkmods.Record, s input.Statement) {
			r.Titles = append(r.Titles, earkmods.Title{Value: s.Value, Lang: s.Lang})
		},
		lang:   true,
		repeat: oncePerLanguage,
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
		if s.Lang != "" && !key.lang {
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
		if key.repeat == oncePerLanguage {
			entry += "\x00" + s.Lang
		}
		if line, seen := first[entry]; seen {
			var err error
			switch {
			case key.repeat == once:
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
