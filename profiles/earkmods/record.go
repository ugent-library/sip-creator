// Package earkmods is the eark-mods profile: a plain E-ARK SIP whose
// descriptive standard is MODS 3.7. A package
// describes one bibliographic record, typed by field where MODS is a tree
// (ADR-0021): the record's identifier and titles, plus the physical copies
// the library holds of it. Its Definition names the rest as data, and the
// registry in profiles/ hands that out as "eark-mods".
package earkmods

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Record is the descriptive metadata of one bibliographic record in MODS
// 3.7: what the record states about the work, field by field, and the
// physical copies of it. It is what a package describes; a representation
// describes a version of the same content. Validate holds the rules on
// what a record may say.
type Record struct {
	// Identifier is the record's local identifier, the catalogue number
	// the describing institution finds it by (in a library, the record
	// number in its catalogue), emitted as a mods:identifier of type local. Empty when the
	// record states none, which a representation's record may.
	Identifier string
	// Titles are the record's titles, one titleInfo/title each, in the
	// order given. A title's language is emitted as xml:lang.
	Titles []Title
	// Items are the physical copies of the record, one per copy. Items
	// belong on the package-level record (ADR-0015): a copy is never a
	// representation. Nothing refuses items on a representation's record;
	// they are written to that representation's mods.xml.
	Items []Item
}

// Title is one title of a record.
type Title struct {
	// Value is the title text.
	Value string
	// Lang is the title's language tag; empty when unspecified.
	Lang string
}

// Item is one physical copy of a bibliographic record: where it is shelved,
// how it is identified, and which part of a multi-part work it is.
type Item struct {
	// CallNumber is the copy's call number, the one value every item states.
	CallNumber string
	// Barcode is the copy's item barcode; empty when the copy has none.
	// Unique across a record's items when set.
	Barcode string
	// Enumeration is the volume or issue designation of a copy of a
	// journal, periodical or newspaper; empty for a single-part work.
	Enumeration string
}

// Record is the profile's description; nothing in this package uses it as
// one, so the assertion makes a drift in the interface a build error here
// rather than in the profile's definition.
var _ sip.Description = Record{}

// langRx is a pragmatic language-tag shape (primary subtag plus optional
// subtags), not full BCP 47 validation.
var langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)

// validateTitle reports why the title cannot be emitted: an empty text, a
// text XML cannot carry, or a malformed language tag.
func validateTitle(t Title) error {
	if strings.TrimSpace(t.Value) == "" {
		return errors.New("has an empty value")
	}
	if err := build.ValidateXMLText(t.Value); err != nil {
		return err
	}
	if t.Lang != "" && !langRx.MatchString(t.Lang) {
		return fmt.Errorf("%q is not a language tag", t.Lang)
	}
	return nil
}

// validateItem reports why the item cannot be emitted: no call number, an
// optional value that is blank rather than absent, which would emit an
// empty element, or a value XML cannot carry. Barcode uniqueness is a cross-item rule, checked in
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
	for _, value := range []string{it.CallNumber, it.Barcode, it.Enumeration} {
		if err := build.ValidateXMLText(value); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks the identifier, every title and every item plus the one
// cross rule: no barcode twice, since two items with one barcode name one
// physical copy twice. Every finding is reported, joined into one error; a
// finding about one title or one item names its position in its text
// ("title 2: ...", "item 2: ..."). The identifier is single by type, so
// the record cannot state two. MODS itself limits nothing here: identifier
// and titleInfo are optional and repeatable.
func (r Record) Validate() error {
	var errs []error
	if r.Identifier != "" && strings.TrimSpace(r.Identifier) == "" {
		errs = append(errs, errors.New("identifier is blank; leave it empty instead"))
	}
	if err := build.ValidateXMLText(r.Identifier); err != nil {
		errs = append(errs, fmt.Errorf("identifier: %w", err))
	}
	for i, title := range r.Titles {
		if err := validateTitle(title); err != nil {
			errs = append(errs, fmt.Errorf("title %d: %w", i+1, err))
		}
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

// ValidateRequired reports what a package-level record must state and
// this one does not: an identifier and a title, as every profile requires
// (input specification §3). A representation's record need not state
// them.
func (r Record) ValidateRequired() error {
	var errs []error
	if r.Identifier == "" {
		errs = append(errs, errors.New("identifier is required but missing"))
	}
	if len(r.Titles) == 0 {
		errs = append(errs, errors.New("title is required but missing"))
	}
	return errors.Join(errs...)
}
