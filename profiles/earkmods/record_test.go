package earkmods

import (
	"errors"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

// testRecord is a complete record: the identity in two languages and two
// copies, one with a barcode and an enumeration, one with a call number
// alone.
func testRecord() Record {
	return Record{
		Terms: []sip.Term{
			{Key: "identifier", Value: "990001234560471"},
			{Key: "title", Lang: "nl", Value: "Fotoalbum Gent 1913"},
			{Key: "title", Lang: "en", Value: "Photo album Ghent 1913"},
		},
		Items: []Item{
			{CallNumber: "BIB.FA.001", Barcode: "000000123", Enumeration: "vol. 3 (1913)"},
			{CallNumber: "BIB.FA.002"},
		},
	}
}

// requireError fails unless err mentions want; want "" means err must be nil.
func requireError(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("want valid, got %v", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("want error mentioning %q, got %v", want, err)
	}
}

func TestValidateTerm(t *testing.T) {
	tests := []struct {
		name string
		term sip.Term
		want string // "" means valid; else substring of the error
	}{
		{"valid", sip.Term{Key: "title", Value: "x"}, ""},
		{"valid with lang", sip.Term{Key: "title", Lang: "nl-BE", Value: "x"}, ""},
		{"valid identifier", sip.Term{Key: "identifier", Value: "x"}, ""},
		// a Simple DC element; the MODS vocabulary has its own keys
		{"dc key", sip.Term{Key: "description", Value: "x"}, "unknown key"},
		// the key is title; titleInfo is the element it emits
		{"element as key", sip.Term{Key: "titleinfo", Value: "x"}, "unknown key"},
		{"prefixed", sip.Term{Key: "mods:title", Value: "x"}, "unknown key"},
		// keys are lowercase; case folding is the rows file's convention
		{"capitalized", sip.Term{Key: "Title", Value: "x"}, "unknown key"},
		{"bad lang", sip.Term{Key: "title", Lang: "nl!", Value: "x"}, "not a language tag"},
		{"empty value", sip.Term{Key: "title", Value: "  "}, "empty value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireError(t, validateTerm(tt.term), tt.want)
		})
	}
}

func TestValidateItem(t *testing.T) {
	tests := []struct {
		name string
		item Item
		want string
	}{
		{"call number alone", Item{CallNumber: "A"}, ""},
		{"complete", Item{CallNumber: "A", Barcode: "1", Enumeration: "vol. 1"}, ""},
		{"no call number", Item{Barcode: "1"}, "no call number"},
		{"blank call number", Item{CallNumber: " "}, "no call number"},
		// empty means none; blank would emit an empty element
		{"blank barcode", Item{CallNumber: "A", Barcode: " "}, "blank barcode"},
		{"blank enumeration", Item{CallNumber: "A", Enumeration: "\t"}, "blank enumeration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireError(t, validateItem(tt.item), tt.want)
		})
	}
}

// MODS repeats titleInfo freely, so a second title is fine; the identifier
// stays single, and a barcode names one copy. Items without a barcode never
// collide: empty means none.
func TestRecordValidate(t *testing.T) {
	requireError(t, testRecord().Validate(), "")

	twoIDs := Record{Terms: []sip.Term{
		{Key: "identifier", Value: "A"},
		{Key: "identifier", Value: "B"},
	}}
	requireError(t, twoIDs.Validate(), "exactly one")

	twoBarcodes := Record{Items: []Item{
		{CallNumber: "A", Barcode: "1"},
		{CallNumber: "B", Barcode: "1"},
	}}
	requireError(t, twoBarcodes.Validate(), `item 2: barcode "1" is also item 1's`)

	noBarcodes := Record{Items: []Item{{CallNumber: "A"}, {CallNumber: "B"}}}
	requireError(t, noBarcodes.Validate(), "")
}

// A package-level record states an identifier and a title; items alone do
// not describe anything.
func TestValidateRequired(t *testing.T) {
	requireError(t, Record{Terms: []sip.Term{{Key: "identifier", Value: "A"}}}.ValidateRequired(), "title is required")
	requireError(t, Record{Terms: []sip.Term{{Key: "title", Value: "x"}}}.ValidateRequired(), "identifier is required")
	requireError(t, testRecord().ValidateRequired(), "")

	err := Record{Items: []Item{{CallNumber: "A"}}}.ValidateRequired()
	for _, want := range []string{"identifier is required", "title is required"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("items-only record: want %q, got %v", want, err)
		}
	}
}

// Validate joins every finding, per-term and per-item ones included, so a
// producer sees all of them in one round. A per-term finding carries the
// term's position, so a caller who decoded the terms from rows can point
// at the row; an item finding names the item's position in its text.
func TestRecordValidateReportsEveryFinding(t *testing.T) {
	r := Record{
		Terms: []sip.Term{
			{Key: "identifier", Value: "A"},
			{Key: "abstract", Value: "x"},
			{Key: "title", Value: " "},
		},
		Items: []Item{
			{CallNumber: "A"},
			{Barcode: "1"},
			{CallNumber: "C", Barcode: "1"},
		},
	}
	err := r.Validate()
	if err == nil {
		t.Fatal("want four findings, got none")
	}
	for _, want := range []string{"term 2:", "term 3:", "item 2: has no call number", "item 3: barcode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	var positions []int
	for _, finding := range err.(interface{ Unwrap() []error }).Unwrap() {
		var te *sip.TermError
		if errors.As(finding, &te) {
			positions = append(positions, te.Index)
		}
	}
	if len(positions) != 2 || positions[0] != 1 || positions[1] != 2 {
		t.Errorf("per-term findings at positions %v, want [1 2]", positions)
	}
}
