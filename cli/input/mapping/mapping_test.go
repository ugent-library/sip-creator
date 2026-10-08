package mapping

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

var terms = []sip.Term{
	{Key: "identifier", Value: "ID-1"},
	{Key: "title", Lang: "nl", Value: "Kat"},
	{Key: "title", Lang: "en", Value: "Cat"},
}

// Under ugent/basic the terms become Simple Dublin Core terms unchanged,
// in order: the keys are the elements' own names.
func TestSimpleDCKeepsTheTerms(t *testing.T) {
	description, errs := SimpleDC{}.Map(terms)
	if len(errs) != 0 {
		t.Fatalf("errors %v for terms the mapping places", errs)
	}
	got, ok := description.(ugent.Terms)
	if !ok {
		t.Fatalf("description is %T, want ugent.Terms", description)
	}
	if !slices.Equal([]sip.Term(got), terms) {
		t.Errorf("terms = %+v, want %+v", got, terms)
	}
}

// Under meemoo/basic each key becomes the element Meemoo's specification
// names, in order, with the language and value unchanged. An unknown key
// is a TermError at its index, and the term stays as written, so the terms
// keep their indexes.
func TestMeemooMapsKeysToElements(t *testing.T) {
	in := []sip.Term{
		{Key: "identifier", Value: "ID-1"},
		{Key: "coverage", Value: "Gent"},
		{Key: "ispartof", Lang: "nl", Value: "Reeks"},
		{Key: "artmedium", Value: "olieverf"},
	}
	description, errs := Meemoo{}.Map(in)
	want := meemoo.Terms{
		{Key: "dcterms:identifier", Value: "ID-1"},
		{Key: "coverage", Value: "Gent"},
		{Key: "dcterms:isPartOf", Lang: "nl", Value: "Reeks"},
		{Key: "schema:artMedium", Value: "olieverf"},
	}
	got, ok := description.(meemoo.Terms)
	if !ok {
		t.Fatalf("description is %T, want meemoo.Terms", description)
	}
	if !slices.Equal(got, want) {
		t.Errorf("terms = %+v, want %+v", got, want)
	}
	if len(errs) != 1 {
		t.Fatalf("errors = %v, want one for the unknown key", errs)
	}
	te, ok := errors.AsType[*sip.TermError](errs[0])
	if !ok || te.Index != 1 || !strings.Contains(te.Err.Error(), `unknown key "coverage"`) {
		t.Errorf("error = %v, want the unknown key at index 1", errs[0])
	}
}

// Every element the Meemoo mapping names is one meemoo.Terms accepts, so the key table cannot point at an element Meemoo's profile
// does not have.
func TestMeemooElementsAreAccepted(t *testing.T) {
	for key, element := range meemooElements {
		term := meemoo.Terms{{Key: element, Lang: "nl", Value: "x"}}
		if err := term.Validate(); err != nil {
			t.Errorf("key %q maps to %q, which the terms refuse: %v", key, element, err)
		}
	}
}

// Under ugent/bibliographic the terms fill the record's fields: the
// identifier once and the titles in order with their language. The rows
// carry no items. Items reach a record through the library or a supplied
// document.
func TestMODSTermsFillTheRecord(t *testing.T) {
	description, errs := MODS{}.Map(terms)
	if len(errs) != 0 {
		t.Fatalf("errors %v for terms the mapping places", errs)
	}
	record, ok := description.(ugent.Record)
	if !ok {
		t.Fatalf("description is %T, want ugent.Record", description)
	}
	if record.Identifier != "ID-1" {
		t.Errorf("Identifier = %q", record.Identifier)
	}
	wantTitles := []ugent.Title{{Value: "Kat", Lang: "nl"}, {Value: "Cat", Lang: "en"}}
	if !slices.Equal(record.Titles, wantTitles) {
		t.Errorf("Titles = %+v, want %+v", record.Titles, wantTitles)
	}
	if record.Items != nil {
		t.Errorf("Items = %+v, want none from rows", record.Items)
	}
}

// otheridentifier repeats freely, and each row adds one identifier in the
// order of the rows.
func TestMODSOtherIdentifiersRepeat(t *testing.T) {
	rows := slices.Concat(terms, []sip.Term{
		{Key: "otheridentifier", Value: "9789000000000"},
		{Key: "otheridentifier", Value: "(RUG01)000000001"},
	})
	description, errs := MODS{}.Map(rows)
	if len(errs) != 0 {
		t.Fatalf("errors %v for terms the mapping places", errs)
	}
	want := []string{"9789000000000", "(RUG01)000000001"}
	if got := description.(ugent.Record).OtherIdentifiers; !slices.Equal(got, want) {
		t.Errorf("OtherIdentifiers = %q, want %q", got, want)
	}
}

// A term the MODS mapping cannot place is a TermError at its index, and
// the term is not placed. The other terms still are. A repeat is reported
// at the later term.
func TestMODSErrorsNameTheTerm(t *testing.T) {
	tests := []struct {
		name  string
		terms []sip.Term
		want  string // substring of the one error
		index int
	}{
		{"unknown key", []sip.Term{{Key: "abstract", Value: "x"}}, `unknown key "abstract"`, 0},
		{"dc element as key", []sip.Term{{Key: "description", Value: "x"}}, "unknown key", 0},
		{"mods element as key", []sip.Term{{Key: "titleinfo", Value: "x"}}, "unknown key", 0},
		{"language on the identifier", []sip.Term{{Key: "identifier", Lang: "nl", Value: "x"}}, "identifier takes no language tag", 0},
		{"empty value", []sip.Term{{Key: "title", Value: " "}}, "title has an empty value", 0},
		{"language on another identifier", []sip.Term{{Key: "otheridentifier", Lang: "nl", Value: "x"}}, "otheridentifier takes no language tag", 0},
		{"empty other identifier", []sip.Term{{Key: "otheridentifier", Value: ""}}, "otheridentifier has an empty value", 0},
		{"second identifier", slices.Concat(terms, []sip.Term{{Key: "identifier", Value: "ID-2"}}), "identifier appears more than once; give exactly one value", 3},
		{"title repeated in one language", slices.Concat(terms, []sip.Term{{Key: "title", Lang: "nl", Value: "Poes"}}), `title appears more than once in language "nl"`, 3},
		{"title repeated untagged", []sip.Term{{Key: "title", Value: "a"}, {Key: "title", Value: "b"}}, "distinct language tags", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, errs := MODS{}.Map(tt.terms)
			if len(errs) != 1 {
				t.Fatalf("errors = %v, want exactly one", errs)
			}
			te, ok := errors.AsType[*sip.TermError](errs[0])
			if !ok {
				t.Fatalf("error is %T, want a *sip.TermError", errs[0])
			}
			if te.Index != tt.index || !strings.Contains(te.Err.Error(), tt.want) {
				t.Errorf("error = term %d %q, want term %d mentioning %q", te.Index, te.Err, tt.index, tt.want)
			}
			// The refused term left no trace: the record holds the terms
			// before it and nothing else.
			record := description.(ugent.Record)
			placed := len(record.Titles) + len(record.OtherIdentifiers)
			if record.Identifier != "" {
				placed++
			}
			if placed != len(tt.terms)-1 {
				t.Errorf("record holds %d values, want the %d terms that were placed: %+v", placed, len(tt.terms)-1, record)
			}
		})
	}
}
