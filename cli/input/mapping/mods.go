package mapping

import (
	"fmt"
	"strings"

	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

// MODS is the ugent/bibliographic profile's mapping: the MODS keys the input
// specification lists and where each one goes in the record. The record is
// typed by field (ADR-0021), so the key table lives here, not in
// profiles/ugent. A term that breaks a rule on its own (an unknown key,
// a language tag where none is taken, an empty value, a repeat) is an
// error at its index and is not placed. The rows carry flat terms only: a
// record's copies reach the package through the library's record or a
// supplied mods.xml.
type MODS struct{}

// placement is what the mapping knows about one key: where its value
// goes in the record, whether the key takes a language tag, and how often
// it may occur.
type placement struct {
	fill      func(*ugent.Record, sip.Term)
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
		fill:   func(r *ugent.Record, t sip.Term) { r.Identifier = t.Value },
		occurs: once,
	},
	"title": {
		fill: func(r *ugent.Record, t sip.Term) {
			r.Titles = append(r.Titles, ugent.Title{Value: t.Value, Lang: t.Lang})
		},
		takesLang: true,
		occurs:    oncePerLanguage,
	},
}

// Map fills a record from the terms in order, reporting each term
// it cannot place as a *sip.TermError at its index. A repeat is reported
// at the repeated term; the first one is placed.
func (MODS) Map(terms []sip.Term) (sip.Description, []error) {
	var record ugent.Record
	var errs []error
	placed := map[string]bool{} // key, or key and language
	for i, t := range terms {
		key, ok := modsKeys[t.Key]
		if !ok {
			errs = append(errs, &sip.TermError{Index: i, Err: fmt.Errorf("unknown key %q: not a key of the %s profile; see the supported keys in the input specification", t.Key, ugent.Bibliographic.Name)})
			continue
		}
		if t.Lang != "" && !key.takesLang {
			errs = append(errs, &sip.TermError{Index: i, Err: fmt.Errorf("%s takes no language tag", t.Key)})
			continue
		}
		if strings.TrimSpace(t.Value) == "" {
			errs = append(errs, &sip.TermError{Index: i, Err: fmt.Errorf("%s has an empty value", t.Key)})
			continue
		}
		// "\x00" cannot appear in a key, so per-language entries never
		// collide with the plain key entries.
		entry := t.Key
		if key.occurs == oncePerLanguage {
			entry += "\x00" + t.Lang
		}
		if placed[entry] {
			var err error
			switch {
			case key.occurs == once:
				err = fmt.Errorf("%s appears more than once; give exactly one value", t.Key)
			case t.Lang == "":
				err = fmt.Errorf("%s appears more than once; repeat it only with distinct language tags (%s[nl], %s[en])", t.Key, t.Key, t.Key)
			default:
				err = fmt.Errorf("%s appears more than once in language %q; give one value per language", t.Key, t.Lang)
			}
			errs = append(errs, &sip.TermError{Index: i, Err: err})
			continue
		}
		placed[entry] = true
		key.fill(&record, t)
	}
	return record, errs
}
