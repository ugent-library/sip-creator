package meemoo

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// langRx is a pragmatic language-tag shape (primary subtag plus optional
// subtags), not full BCP 47 validation.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// validateTerm reports why the term cannot be emitted: an element outside
// the profile's elements, a malformed language tag, an empty value, or a
// value XML cannot carry.
// These rules apply to every producer, not just the CSV transport.
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

// Validate checks every term, the one-identifier rule, and Meemoo's own
// rules: the elements' cardinality limits and a Dutch entry wherever a
// language-tagged element appears. Every finding is reported, joined into
// one error, so a producer corrects a document in one round; a finding
// about one term is a *sip.TermError naming the term's position. The
// identifier rule stands on its own because the local identifier is an
// identity, and two of them is an ambiguity no consumer can resolve; the
// table also lists the identifier as `once`, and that overlap is
// deliberate.
func (t Terms) Validate() error {
	var errs []error
	identifiers := 0
	for i, term := range t {
		if err := validateTerm(term); err != nil {
			errs = append(errs, &sip.TermError{Index: i, Err: err})
		}
		if term.Key == identifierElement {
			identifiers++
		}
	}
	if identifiers > 1 {
		errs = append(errs, fmt.Errorf("%s appears %d times; give exactly one", identifierElement, identifiers))
	}
	errs = append(errs, t.validateCardinality(), t.validateRequiredLang(RequiredLang))
	return errors.Join(errs...)
}

// validateCardinality reports every term that exceeds its element's
// cardinality (Meemoo's 0..1/1..1 restrictions, counted per language for
// lang-tagged elements). Findings name the element (and language). An
// element outside the table has the zero cardinality, many, so an unknown
// element never adds a false repeat finding to the one validateTerm
// already gave.
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

// ValidateRequired reports each element a package-level description must
// state (required) that the terms do not.
func (t Terms) ValidateRequired() error {
	var errs []error
	for _, element := range required {
		if !t.has(element) {
			errs = append(errs, fmt.Errorf("%s is required but missing", element))
		}
	}
	return errors.Join(errs...)
}

// validateRequiredLang reports each element that carries language-tagged
// values without one in lang ("" disables the rule).
func (t Terms) validateRequiredLang(lang string) error {
	if lang == "" {
		return nil
	}
	var tagged []string // elements with lang-tagged values, in first-appearance order
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
