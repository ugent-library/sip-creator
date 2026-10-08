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
// names ("title", "coverage"). dc.xml keeps the order the producer gave. A
// term's language tag is accepted so producers can state it, but dc.xml
// leaves it out, because the simpledc document carries no xml:lang.
// Validate holds the rules on what a term may say.
type Terms []sip.Term

func (t Terms) has(key string) bool {
	for _, term := range t {
		if term.Key == key {
			return true
		}
	}
	return false
}

// langRx matches the shape of a language tag: a primary subtag of two or
// three letters, then optional subtags. It does not check the full BCP 47
// grammar.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// validateTerm checks that the term's key is one of the fifteen elements,
// that its language tag is well-formed when it has one, and that its value
// is not empty and holds only text XML can carry. It returns an error
// naming the first rule the term breaks.
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

// Validate checks every term. The identifier may repeat, like every Simple
// Dublin Core element, so a producer can state several identifiers.
func (t Terms) Validate() error {
	var errs []error
	for i, term := range t {
		if err := validateTerm(term); err != nil {
			errs = append(errs, &sip.TermError{Index: i, Err: err})
		}
	}
	return errors.Join(errs...)
}

// ValidateRequired checks that the terms state every key in dcRequired. It
// returns an error naming each missing key.
func (t Terms) ValidateRequired() error {
	var errs []error
	for _, key := range dcRequired {
		if !t.has(key) {
			errs = append(errs, fmt.Errorf("%s is required but missing", key))
		}
	}
	return errors.Join(errs...)
}
