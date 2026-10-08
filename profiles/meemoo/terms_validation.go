package meemoo

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// langRx matches the shape of a language tag: a primary subtag of two or
// three letters, then optional subtags. It does not check the full BCP 47
// grammar.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// validateTerm checks that the term names an element of the profile, that
// its language tag is well-formed when it has one, and that its value is
// not empty and holds only text XML can carry. It returns an error naming
// the first rule the term breaks.
func validateTerm(t sip.Term) error {
	if _, ok := elementsByName[t.Key]; !ok {
		return fmt.Errorf("unknown element %q: not an element of Meemoo's basic content profile", t.Key)
	}
	if t.Lang != "" && !langRx.MatchString(t.Lang) {
		return fmt.Errorf("%q is not a language tag", t.Lang)
	}
	if strings.TrimSpace(t.Value) == "" {
		return fmt.Errorf("%s has an empty value", t.Key)
	}
	if err := build.ValidateXMLText(t.Value); err != nil {
		return fmt.Errorf("%s: %w", t.Key, err)
	}
	return nil
}

// RequiredLang is the language Meemoo requires a value in wherever a
// language-tagged element appears: Dutch (Meemoo SIP 1.2, basic content
// profile).
const RequiredLang = "nl"

// Validate checks every term, and Meemoo's own rules: the elements'
// cardinality limits and a Dutch entry wherever a language-tagged element
// appears. A second identifier is a cardinality finding, because the table
// allows the identifier once.
func (t Terms) Validate() error {
	var errs []error
	for i, term := range t {
		if err := validateTerm(term); err != nil {
			errs = append(errs, &sip.TermError{Index: i, Err: err})
		}
	}
	errs = append(errs, t.validateCardinality(), t.validateRequiredLang(RequiredLang))
	return errors.Join(errs...)
}

// validateCardinality checks the terms against Meemoo's 0..1 and 1..1
// restrictions, counted per language for language-tagged elements. It
// returns one finding for each element, or element and language, that
// occurs too often.
func (t Terms) validateCardinality() error {
	seen := map[string]int{}
	var errs []error
	for _, term := range t {
		switch elementsByName[term.Key].Repeat {
		case once:
			seen[term.Key]++
			if seen[term.Key] == 2 {
				errs = append(errs, fmt.Errorf("%s appears more than once; give exactly one value", term.Key))
			}
		case oncePerLanguage:
			// "\x00" cannot appear in an element name, so per-language
			// entries never collide with the plain entries above.
			entry := term.Key + "\x00" + term.Lang
			seen[entry]++
			if seen[entry] != 2 {
				continue // report each offending element/language pair once
			}
			if term.Lang == "" {
				errs = append(errs, fmt.Errorf("%s appears more than once; repeat it only with a distinct language on each value", term.Key))
				continue
			}
			errs = append(errs, fmt.Errorf("%s appears more than once in language %q; give one value per language", term.Key, term.Lang))
		}
	}
	return errors.Join(errs...)
}

// ValidateRequired checks that the terms state every element in required.
// It returns an error naming each missing element.
func (t Terms) ValidateRequired() error {
	var errs []error
	for _, element := range required {
		if !t.has(element) {
			errs = append(errs, fmt.Errorf("%s is required but missing", element))
		}
	}
	return errors.Join(errs...)
}

// validateRequiredLang checks that every element with language-tagged
// values has at least one value in lang. It returns an error naming each
// element that has none. An empty lang turns the check off.
func (t Terms) validateRequiredLang(lang string) error {
	if lang == "" {
		return nil
	}
	var tagged []string // elements with language-tagged values, in order of first appearance
	// missing doubles as the seen set: an element enters as true (missing) on
	// first sight and flips to false once a value in lang appears.
	missing := map[string]bool{}
	for _, term := range t {
		if term.Lang == "" {
			continue
		}
		if _, seen := missing[term.Key]; !seen {
			tagged = append(tagged, term.Key)
			missing[term.Key] = true
		}
		if term.Lang == lang {
			missing[term.Key] = false
		}
	}
	var errs []error
	for _, element := range tagged {
		if missing[element] {
			errs = append(errs, fmt.Errorf("%s carries language-tagged values but none in %q", element, lang))
		}
	}
	return errors.Join(errs...)
}
