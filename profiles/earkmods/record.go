// Package earkmods is the eark-mods profile: a plain E-ARK SIP for
// RODA-class repositories whose descriptive world is MODS 3.7. A package
// describes one bibliographic record, the statements about the work plus
// the physical copies the library holds of it, and the vocabulary maps
// each plain key onto one complete MODS element (ADR-0015). Its Definition
// names the rest as data, and the registry in profiles/ hands that out as
// "eark-mods".
package earkmods

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/sip"
)

// Record is the descriptive metadata of one bibliographic record in MODS
// 3.7: the statements about the work, and the physical copies of it. It is
// what a package describes; a representation describes a version of the
// same content and states terms alone. Validate holds the rules on what a
// record may say.
type Record struct {
	// Terms are the statements about the work, each keyed by a plain key of
	// the MODS vocabulary ("identifier", "title"); the order the producer
	// stated them in is preserved through to the emitted XML.
	Terms []sip.Term
	// Items are the physical copies of the record, one per copy. Items
	// describe the package level only (ADR-0015): a copy is never a
	// representation, so a representation's record carries none.
	Items []Item
}

// Item is one physical copy of a bibliographic record: where it is shelved,
// how it is identified, and which part of a multi-part work it is.
type Item struct {
	// CallNumber is the copy's shelf mark, the one value every item states.
	CallNumber string
	// Barcode is the copy's item barcode; empty when the copy has none.
	// Unique across a record's items when set.
	Barcode string
	// Enumeration is the volume or issue designation of a copy of a
	// journal, periodical or newspaper; empty for a single-part work.
	Enumeration string
}

// Record is the profile's description; nothing in this package uses it as
// one yet, so the assertion makes a drift in the interface a build error
// here rather than in the profile's definition.
var _ sip.Description = Record{}

// has reports whether any term states the given key.
func (r Record) has(key string) bool {
	for _, term := range r.Terms {
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
// vocabulary, a malformed language tag, or an empty value.
func validateTerm(t sip.Term) error {
	if _, ok := vocabularyByKey[t.Key]; !ok {
		return fmt.Errorf("unknown key %q: not in the MODS vocabulary; see the supported keys in the input specification", t.Key)
	}
	if t.Lang != "" && !langRx.MatchString(t.Lang) {
		return fmt.Errorf("%q is not a language tag", t.Lang)
	}
	if strings.TrimSpace(t.Value) == "" {
		return fmt.Errorf("%s has an empty value", t.Key)
	}
	return nil
}

// validateItem reports why the item cannot be emitted: no call number, or
// an optional value that is blank rather than absent, which would emit an
// empty element. Barcode uniqueness is a cross-item rule, checked in
// Validate.
func validateItem(it Item) error {
	if strings.TrimSpace(it.CallNumber) == "" {
		return errors.New("has no call number; every item states one")
	}
	if it.Barcode != "" && strings.TrimSpace(it.Barcode) == "" {
		return errors.New("has a blank barcode; leave it empty instead")
	}
	if it.Enumeration != "" && strings.TrimSpace(it.Enumeration) == "" {
		return errors.New("has a blank enumeration; leave it empty instead")
	}
	return nil
}

// Validate checks every term and every item plus the two cross rules: at
// most one identifier, and no barcode twice. Every finding is reported,
// joined into one error; a finding about one term is a *sip.TermError
// naming the term's position, and a finding about one item names the
// item's position in its text ("item 2: ..."). The identifier is the
// record's identity, and two of them is an ambiguity no consumer can
// resolve; two items with one barcode name one physical copy twice. MODS
// itself limits nothing here: identifier and titleInfo are optional and
// repeatable.
func (r Record) Validate() error {
	var errs []error
	identifiers := 0
	for i, term := range r.Terms {
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

	barcodes := make(map[string]int, len(r.Items)) // barcode → position of the item that first carried it
	for i, item := range r.Items {
		if err := validateItem(item); err != nil {
			errs = append(errs, fmt.Errorf("item %d: %w", i+1, err))
		}
		if item.Barcode == "" {
			continue
		}
		if first, ok := barcodes[item.Barcode]; ok {
			errs = append(errs, fmt.Errorf("item %d: barcode %q is also item %d's; a barcode names one copy", i+1, item.Barcode, first+1))
			continue
		}
		barcodes[item.Barcode] = i
	}
	return errors.Join(errs...)
}

// ValidateRequired reports each key a package-level record must state
// (required) that the terms do not.
func (r Record) ValidateRequired() error {
	var errs []error
	for _, key := range required {
		if !r.has(key) {
			errs = append(errs, fmt.Errorf("%s is required but missing", key))
		}
	}
	return errors.Join(errs...)
}
