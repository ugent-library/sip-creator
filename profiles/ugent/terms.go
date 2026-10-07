package ugent

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Terms is the descriptive metadata of ugent/basic: an ordered list of
// terms in Simple Dublin Core, each keyed by one of the fifteen element
// names ("title", "coverage"); the order the producer gave them in is
// preserved through to the emitted XML. A term's language tag is accepted
// so producers can state it, but not emitted: the simpledc document
// carries no xml:lang. Validate holds the rules on what a term may say.
type Terms []sip.Term

// has reports whether any term states the given key.
func (t Terms) has(key string) bool {
	for _, term := range t {
		if term.Key == key {
			return true
		}
	}
	return false
}

// langRx is a pragmatic language-tag shape (primary subtag plus optional
// subtags), not full BCP 47 validation.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// validateTerm reports why the term cannot be emitted: a key outside the
// fifteen, a malformed language tag, an empty value, or a value XML cannot
// carry.
func validateTerm(t sip.Term) error {
	if !dcElementSet[t.Key] {
		return fmt.Errorf("unknown key %q: not a Simple Dublin Core element; see the supported keys in the input specification", t.Key)
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

// Validate checks every term plus the one cross-term rule: at most one
// identifier. Every finding is reported, joined into one error; a finding
// about one term is a *sip.TermError naming the term's position. The
// local identifier is an identity, and two of them is an ambiguity no
// consumer can resolve. Simple Dublin Core itself limits nothing: every
// element is optional and repeatable.
func (t Terms) Validate() error {
	var errs []error
	identifiers := 0
	for i, term := range t {
		if err := validateTerm(term); err != nil {
			errs = append(errs, &sip.TermError{Index: i, Err: err})
		}
		if term.Key == "identifier" {
			identifiers++
		}
	}
	if identifiers > 1 {
		errs = append(errs, fmt.Errorf("identifier appears %d times; give exactly one", identifiers))
	}
	return errors.Join(errs...)
}

// ValidateRequired reports each key a package-level description must
// state (required) that the terms do not.
func (t Terms) ValidateRequired() error {
	var errs []error
	for _, key := range dcRequired {
		if !t.has(key) {
			errs = append(errs, fmt.Errorf("%s is required but missing", key))
		}
	}
	return errors.Join(errs...)
}
