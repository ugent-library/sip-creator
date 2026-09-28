package dcschema

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// langRx is a pragmatic language-tag shape (primary subtag plus optional
// subtags), not full BCP 47 validation.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// Validate reports why the term cannot be emitted: an element outside the
// descriptive vocabulary, a malformed language tag, or an empty value.
// These rules apply to every producer, not just the CSV transport.
func (t Term) Validate() error {
	if _, ok := vocabularyByElement[t.Element]; !ok {
		return fmt.Errorf("%q is not in the descriptive vocabulary; see the supported keys in the input specification", t.Element)
	}
	if t.Lang != "" && !langRx.MatchString(t.Lang) {
		return fmt.Errorf("%q is not a language tag", t.Lang)
	}
	if strings.TrimSpace(t.Value) == "" {
		return fmt.Errorf("%s has an empty value", t.Element)
	}
	return nil
}

// RequiredLang is the language meemoo requires a value in wherever a
// language-tagged element appears: Dutch (meemoo SIP 1.2, basic content
// profile).
const RequiredLang = "nl"

// Validate checks every term, the one-identifier rule, and meemoo's own
// rules: the vocabulary's cardinality limits and a Dutch entry wherever a
// language-tagged element appears. Every finding is reported, joined into
// one error, so a producer corrects a document in one round. The
// identifier rule stands on its own because the local identifier is an
// identity, and two of them is an ambiguity no consumer can resolve; the
// vocabulary also lists identifier as `once`, and that overlap is
// deliberate.
func (t Terms) Validate() error {
	var errs []error
	identifiers := 0
	for i, term := range t {
		if err := term.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("term %d: %w", i+1, err))
		}
		if term.Element == "dcterms:identifier" {
			identifiers++
		}
	}
	if identifiers > 1 {
		errs = append(errs, fmt.Errorf("dcterms:identifier appears %d times; give exactly one", identifiers))
	}
	errs = append(errs, t.validateCardinality(), t.validateRequiredLang(RequiredLang))
	return errors.Join(errs...)
}

// validateCardinality reports every term that exceeds its element's
// cardinality (meemoo's 0..1/1..1 restrictions, counted per language for
// lang-tagged elements). Findings name the element (and language), which
// locates the offending rows in a keyed file. An element outside the table
// has the zero cardinality, many, so an unknown element never adds a false
// repeat finding to the one Term.Validate already gave.
func (t Terms) validateCardinality() error {
	seen := map[string]int{}
	var errs []error
	for _, term := range t {
		switch vocabularyByElement[term.Element].Repeat {
		case once:
			seen[term.Element]++
			if seen[term.Element] == 2 {
				errs = append(errs, fmt.Errorf("%s appears more than once; give exactly one value", term.Element))
			}
		case oncePerLanguage:
			// "\x00" cannot appear in an element name, so per-language
			// keys never collide with the plain element keys above.
			key := term.Element + "\x00" + term.Lang
			seen[key]++
			if seen[key] != 2 {
				continue // report each offending element/language pair once
			}
			if term.Lang == "" {
				errs = append(errs, fmt.Errorf("%s appears more than once; repeat it only with distinct language tags (title[nl], title[en])", term.Element))
				continue
			}
			errs = append(errs, fmt.Errorf("%s appears more than once in language %q; give one value per language", term.Element, term.Lang))
		}
	}
	return errors.Join(errs...)
}

// ValidateRequired reports each element a package-level description must
// state (required) that the terms do not. A finding names the CSV key and
// the element, so a rows file and a library caller's terms can both be
// corrected from it.
func (t Terms) ValidateRequired() error {
	var errs []error
	for _, element := range required {
		if !t.has(element) {
			errs = append(errs, fmt.Errorf("%s (%s) is required but missing", vocabularyByElement[element].Key, element))
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
	// missing doubles as the seen set: an element enters as true (missing)
	// on first sight and flips to false once a value in lang appears.
	missing := map[string]bool{}
	for _, term := range t {
		if term.Lang == "" {
			continue
		}
		if _, seen := missing[term.Element]; !seen {
			tagged = append(tagged, term.Element)
			missing[term.Element] = true
		}
		if term.Lang == lang {
			missing[term.Element] = false
		}
	}
	var errs []error
	for _, el := range tagged {
		if missing[el] {
			errs = append(errs, fmt.Errorf("%s carries language-tagged values but none in %q", el, lang))
		}
	}
	return errors.Join(errs...)
}
