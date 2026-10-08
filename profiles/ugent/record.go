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
// how it is identified, and which part of a multi-part work it is.
type Item struct {
	// CallNumber is the copy's call number, the one value every item states.
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

// validateItem checks that the item states a call number, that its barcode
// and enumeration are not blank when set, and that every value holds only
// text XML can carry. It returns an error naming the first rule the item
// breaks. A blank value would be written as an empty element, so an item
// without a barcode or an enumeration leaves that field empty. Validate
// checks that barcodes are unique, because that rule spans items.
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

// Validate checks the identifier, every title and every item, and one rule
// across items: no barcode twice, because two items with one barcode name
// one physical copy twice. A finding about one title or one item names its
// position in its text, such as "title 2: ..." or "item 2: ...".
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
