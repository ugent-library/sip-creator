// Package dc is the Simple Dublin Core descriptive world: the fifteen
// elements of the Dublin Core Metadata Element Set, unqualified, written as
// the simpledc document RODA renders and indexes natively. The eark profile
// writes it.
package dc

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Term is one descriptive statement: one of the fifteen Simple Dublin Core
// elements, an optional language tag, and the value.
type Term struct {
	// Element is the element name, e.g. "title"; ResolveKey knows the
	// fifteen.
	Element string
	// Lang is the value's language tag. It is accepted so producers can
	// state it, but not emitted: the simpledc document carries no xml:lang.
	Lang string
	// Value is the term's text.
	Value string
}

// Terms is an ordered list of descriptive statements; the order the
// producer stated them in is preserved through to the emitted XML.
// Validate holds the rules on what a term may say, and Encode refuses
// invalid terms.
type Terms []Term

// Has reports whether any term states the given element.
func (t Terms) Has(element string) bool {
	for _, term := range t {
		if term.Element == element {
			return true
		}
	}
	return false
}

// LocalIdentifier returns the value of the identifier element: the
// producer's local catalog or inventory number ("" when absent).
func (t Terms) LocalIdentifier() string {
	for _, term := range t {
		if term.Element == "identifier" {
			return term.Value
		}
	}
	return ""
}

// langRx is a pragmatic language-tag shape (primary subtag plus optional
// subtags), not full BCP 47 validation.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// Validate reports why the term cannot be emitted: an element outside the
// fifteen, a malformed language tag, or an empty value.
func (t Term) Validate() error {
	if !elementSet[t.Element] {
		return fmt.Errorf("%q is not a Simple Dublin Core element; see the supported keys in the input specification", t.Element)
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
// identifier. The local identifier is an identity, and two of them is an
// ambiguity no consumer can resolve. Simple Dublin Core itself limits
// nothing: every element is optional and repeatable.
func (t Terms) Validate() error {
	identifiers := 0
	for i, term := range t {
		if err := term.Validate(); err != nil {
			return fmt.Errorf("term %d: %w", i+1, err)
		}
		if term.Element == "identifier" {
			identifiers++
		}
	}
	if identifiers > 1 {
		return fmt.Errorf("identifier appears %d times; give exactly one", identifiers)
	}
	return nil
}

// ValidateRequired reports each required element the terms do not state.
// Which elements are required is profile data (plain E-ARK asks for the
// input convention's identity MUSTs), so the set arrives as an argument.
func (t Terms) ValidateRequired(elements ...string) error {
	var errs []error
	for _, el := range elements {
		if !t.Has(el) {
			errs = append(errs, fmt.Errorf("%s is required but missing", el))
		}
	}
	return errors.Join(errs...)
}
