package mapping

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Every registered profile has a mapping, and each builds what its
// profile's metadata model accepts: Read hands terms to the one, the
// engine runs ValidateType on the result. A profile without a mapping
// could not read a folder.
func TestEveryProfileHasAMappingItsModelAccepts(t *testing.T) {
	for _, name := range profiles.Names() {
		def, _ := profiles.Get(name)
		mapper, ok := For(name)
		if !ok {
			t.Errorf("profile %q has no mapping", name)
			continue
		}
		description, errs := mapper.Map(nil)
		if len(errs) != 0 {
			t.Errorf("profile %q: no terms, yet errors %v", name, errs)
		}
		if err := def.Model.ValidateType(description); err != nil {
			t.Errorf("profile %q: ValidateType refuses what its mapping built: %v", name, err)
		}
	}
	if _, ok := For("nope"); ok {
		t.Error("an unregistered name has a mapping")
	}
}

var terms = []sip.Term{
	{Key: "identifier", Value: "ID-1"},
	{Key: "title", Lang: "nl", Value: "Kat"},
	{Key: "title", Lang: "en", Value: "Cat"},
}

// Under basic and eark the terms become the profile's terms unchanged, in
// order, so a term error's index names the row.
func TestTermsKeepTheirOrder(t *testing.T) {
	tests := []struct {
		name   string
		mapper input.Mapper
		terms  func(sip.Description) ([]sip.Term, bool)
	}{
		{"basic", Meemoo{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(meemoo.Terms); return tt, ok }},
		{"eark", Eark{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(eark.Terms); return tt, ok }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, errs := tt.mapper.Map(terms)
			if len(errs) != 0 {
				t.Fatalf("errors %v for terms every mapping places", errs)
			}
			got, ok := tt.terms(description)
			if !ok {
				t.Fatalf("description is %T", description)
			}
			if !slices.Equal(got, terms) {
				t.Errorf("terms = %+v, want %+v", got, terms)
			}
		})
	}
}

// Under eark-mods the terms fill the record's fields: the identifier once
// and the titles in order with their language. The rows carry no items;
// those reach a record through the library or a supplied document.
func TestEarkModsTermsFillTheRecord(t *testing.T) {
	description, errs := EarkMods{}.Map(terms)
	if len(errs) != 0 {
		t.Fatalf("errors %v for terms the mapping places", errs)
	}
	record, ok := description.(earkmods.Record)
	if !ok {
		t.Fatalf("description is %T, want earkmods.Record", description)
	}
	if record.Identifier != "ID-1" {
		t.Errorf("Identifier = %q", record.Identifier)
	}
	wantTitles := []earkmods.Title{{Value: "Kat", Lang: "nl"}, {Value: "Cat", Lang: "en"}}
	if !slices.Equal(record.Titles, wantTitles) {
		t.Errorf("Titles = %+v, want %+v", record.Titles, wantTitles)
	}
	if record.Items != nil {
		t.Errorf("Items = %+v, want none from rows", record.Items)
	}
}

// A term the MODS mapping cannot place is a TermError at its index, and
// the term is not placed; the terms around it still are. A repeat is
// reported at the repeated term.
func TestEarkModsErrorsNameTheTerm(t *testing.T) {
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
		{"second identifier", slices.Concat(terms, []sip.Term{{Key: "identifier", Value: "ID-2"}}), "identifier appears more than once; give exactly one value", 3},
		{"title repeated in one language", slices.Concat(terms, []sip.Term{{Key: "title", Lang: "nl", Value: "Poes"}}), `title appears more than once in language "nl"`, 3},
		{"title repeated untagged", []sip.Term{{Key: "title", Value: "a"}, {Key: "title", Value: "b"}}, "distinct language tags", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, errs := EarkMods{}.Map(tt.terms)
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
			record := description.(earkmods.Record)
			placed := len(record.Titles)
			if record.Identifier != "" {
				placed++
			}
			if placed != len(tt.terms)-1 {
				t.Errorf("record holds %d values, want the %d terms that were placed: %+v", placed, len(tt.terms)-1, record)
			}
		})
	}
}
