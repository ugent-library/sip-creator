package earkmods

import (
	"strings"
	"testing"
)

// testRecord is a complete record: the identity with the title in two
// languages, and two copies, one with a barcode and an enumeration, one
// with a call number alone.
func testRecord() Record {
	return Record{
		Identifier: "example-0001",
		Titles: []Title{
			{Value: "Fotoalbum 2026", Lang: "nl"},
			{Value: "Photo album 2026", Lang: "en"},
		},
		Items: []Item{
			{CallNumber: "EX.0001", Barcode: "000000123", Enumeration: "vol. 3 (1913)"},
			{CallNumber: "EX.0002"},
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

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name  string
		title Title
		want  string // "" means valid; else substring of the error
	}{
		{"valid", Title{Value: "x"}, ""},
		{"valid with lang", Title{Value: "x", Lang: "nl-BE"}, ""},
		{"bad lang", Title{Value: "x", Lang: "nl!"}, "not a language tag"},
		{"empty value", Title{Value: "  "}, "empty value"},
		{"control character", Title{Value: "a\x01b"}, "U+0001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireError(t, validateTitle(tt.title), tt.want)
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
		// XML cannot carry it, in any of the three values
		{"control character in the call number", Item{CallNumber: "A\x01"}, "U+0001"},
		{"control character in the barcode", Item{CallNumber: "A", Barcode: "1\x1b"}, "U+001B"},
		{"control character in the enumeration", Item{CallNumber: "A", Enumeration: "vol.\x0b1"}, "U+000B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireError(t, validateItem(tt.item), tt.want)
		})
	}
}

// A blank identifier is a finding (an empty one means none); a barcode
// names one copy. Items without a barcode never collide: empty means none.
func TestRecordValidate(t *testing.T) {
	requireError(t, testRecord().Validate(), "")

	requireError(t, Record{Identifier: " "}.Validate(), "identifier is blank")
	requireError(t, Record{Identifier: "ID\x01"}.Validate(), `identifier: "ID\x01" holds the character U+0001`)
	requireError(t, Record{}.Validate(), "")

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
	requireError(t, Record{Identifier: "A"}.ValidateRequired(), "title is required")
	requireError(t, Record{Titles: []Title{{Value: "x"}}}.ValidateRequired(), "identifier is required")
	requireError(t, testRecord().ValidateRequired(), "")

	err := Record{Items: []Item{{CallNumber: "A"}}}.ValidateRequired()
	for _, want := range []string{"identifier is required", "title is required"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("items-only record: want %q, got %v", want, err)
		}
	}
}

// Validate joins every finding, per-title and per-item ones included, so a
// producer sees all of them in one round; each names its position in its
// text.
func TestRecordValidateReportsEveryFinding(t *testing.T) {
	r := Record{
		Identifier: "A",
		Titles: []Title{
			{Value: "x"},
			{Value: " "},
			{Value: "y", Lang: "nl!"},
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
	for _, want := range []string{"title 2: has an empty value", `title 3: "nl!" is not a language tag`, "item 2: has no call number", "item 3: barcode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	if got := len(err.(interface{ Unwrap() []error }).Unwrap()); got != 4 {
		t.Errorf("got %d findings, want 4: %v", got, err)
	}
}
