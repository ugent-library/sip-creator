package metadata

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
// These rules bind every producer, not just the CSV transport.
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

// Validate checks every term plus the one cross-term rule: at most one
// dcterms:identifier. The local identifier is an identity, and two of
// them is an ambiguity no consumer can resolve. The vocabulary also marks
// identifier `once`, so ValidateCardinality states this rule again. That
// is deliberate, not a candidate for dedup: cardinality binds only where a
// profile enforces it (the meemoo family), while the identifier rule is
// unconditional and must hold for eark too.
func (t Terms) Validate() error {
	identifiers := 0
	for i, term := range t {
		if err := term.Validate(); err != nil {
			return fmt.Errorf("term %d: %w", i+1, err)
		}
		if term.Element == "dcterms:identifier" {
			identifiers++
		}
	}
	if identifiers > 1 {
		return fmt.Errorf("dcterms:identifier appears %d times; give exactly one", identifiers)
	}
	return nil
}

// ValidateCardinality reports every term that exceeds its element's
// cardinality mark (meemoo's 0..1/1..1 restrictions, counted per language
// for lang-tagged elements). Whether the marks bind is the profile family's
// call: the meemoo family enforces them, plain E-ARK does not. Findings
// name the element (and language), which locates the offending rows in a
// keyed file; one joined error carries them all.
func (t Terms) ValidateCardinality() error {
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

// ValidateRequired reports each required element the terms do not state.
// Which elements are required is profile data (meemoo's basic profile
// mandates four; plain E-ARK only the input convention's identity MUSTs),
// so the set arrives as an argument.
func (t Terms) ValidateRequired(elements ...string) error {
	var errs []error
	for _, el := range elements {
		if !t.Has(el) {
			errs = append(errs, fmt.Errorf("%s is required but missing", el))
		}
	}
	return errors.Join(errs...)
}

// ValidateRequiredLang reports each element that carries language-tagged
// values without one in the required language. meemoo requires an entry in
// Dutch wherever a lang-tagged element appears; which language (if any) the
// rule demands is profile data, so it arrives as an argument ("" disables).
func (t Terms) ValidateRequiredLang(lang string) error {
	if lang == "" {
		return nil
	}
	var tagged []string // elements with lang-tagged values, in first-appearance order
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
