package ugent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ugent-library/sip-creator/build"
)

// Record is the descriptive metadata of ugent/bibliographic: one
// bibliographic record in MODS 3.7, typed by field where MODS is a tree
// (ADR-0021). It is UGent's application profile of MODS, named in the
// library's catalogue words: what the record states about the work, and the
// physical copies of it. A package-level record describes the intellectual
// entity. A representation's record describes one version of the same
// content. Validate holds the rules on what a record may say.
type Record struct {
	// Identifier is the record's local identifier: the catalogue number the
	// describing institution finds it by, such as the record number in a
	// library catalogue. It is written as a mods:identifier of type local.
	// A package-level record always states one. A representation's record
	// may leave it empty.
	Identifier string
	// OtherIdentifiers are the record's identifiers besides Identifier, in
	// the order given: standard numbers such as an ISBN or ISSN, and the
	// numbers other systems know the record by. Each is written as a
	// mods:identifier without a type attribute, because the catalogue does
	// not say which kind of identifier each is.
	OtherIdentifiers []string
	// Titles are the record's titles, one titleInfo/title each, in the
	// order given. A title's language is written as xml:lang.
	Titles []Title
	// Items are the physical copies of the record, one per copy. Items
	// belong on the package-level record (ADR-0015), because a copy is
	// never a representation. Nothing refuses items on a representation's
	// record. They are written to that representation's mods.xml.
	Items []Item
}

// Title is one title of a record.
type Title struct {
	// Value is the title text.
	Value string
	// Lang is the title's language tag, or empty when the title has none.
	Lang string
}

// Item is one physical copy of a bibliographic record: where it is shelved,
// how it is identified, and which part of a multi-part work it is. An item
// states a call number, a barcode, or both.
type Item struct {
	// CallNumber is the copy's call number, or empty when the catalogue has
	// none for it.
	CallNumber string
	// Barcode is the copy's item barcode, or empty when the copy has none.
	// No two items of a record share a barcode.
	Barcode string
	// Enumeration is the volume or issue designation of a copy of a
	// journal, periodical or newspaper, or empty for a single-part work.
	Enumeration string
}

// validateTitle checks that the title's text is not empty and holds only
// text XML can carry, and that its language tag is well-formed when it has
// one. It returns an error naming the first rule the title breaks.
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

// validateItem checks that the item states a call number or a barcode, that
// none of its values is blank when set, and that every value holds only text
// XML can carry. It returns an error naming the first rule the item breaks.
// An item with neither a call number nor a barcode would be written as a
// copyInformation that names no copy. A blank value would be written as an
// empty element, so an item without a value leaves that field empty.
// Validate checks that barcodes are unique, because that rule spans items.
func validateItem(it Item) error {
	if it.CallNumber == "" && it.Barcode == "" {
		return errors.New("has neither a call number nor a barcode; an item states at least one")
	}
	if it.CallNumber != "" && strings.TrimSpace(it.CallNumber) == "" {
		return errors.New("has a blank call number; leave it empty instead")
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

// validateOtherIdentifier checks that the identifier is not empty and holds
// only text XML can carry. It returns an error naming the first rule the
// identifier breaks. An empty entry would be written as an empty
// mods:identifier, so a record without other identifiers leaves the list
// empty instead.
func validateOtherIdentifier(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("is empty")
	}
	return build.ValidateXMLText(id)
}

// Validate checks the identifier, every other identifier, every title and
// every item, and one rule across items: no barcode twice, because two
// items with one barcode name one physical copy twice. A finding about one
// entry of a list names its position in its text, such as "title 2: ..."
// or "item 2: ...".
func (r Record) Validate() error {
	var errs []error
	if r.Identifier != "" && strings.TrimSpace(r.Identifier) == "" {
		errs = append(errs, errors.New("identifier is blank; leave it empty instead"))
	}
	if err := build.ValidateXMLText(r.Identifier); err != nil {
		errs = append(errs, fmt.Errorf("identifier: %w", err))
	}
	for i, id := range r.OtherIdentifiers {
		if err := validateOtherIdentifier(id); err != nil {
			errs = append(errs, fmt.Errorf("other identifier %d: %w", i+1, err))
		}
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

// ValidateRequired checks that the record states an identifier and at
// least one title, as docs/profiles/ugent-bibliographic.md §3 requires
// ("The description MUST state the catalogue record's identifier and at
// least one title."). It returns an error naming each missing field.
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
