package vocabulary

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Every registered profile has a vocabulary, and each builds what its
// profile's encoder accepts: the reader hands rows to the one, the engine
// runs Check on the result. A profile without a vocabulary could not read
// a folder.
func TestEveryProfileHasAVocabularyItsEncoderAccepts(t *testing.T) {
	for _, name := range profiles.Names() {
		def, _ := profiles.Get(name)
		vocab, ok := For(name)
		if !ok {
			t.Errorf("profile %q has no vocabulary", name)
			continue
		}
		description, findings := vocab.Description(nil, nil)
		if len(findings) != 0 {
			t.Errorf("profile %q: no rows, yet findings %v", name, findings)
		}
		if err := def.Encoder.Check(description); err != nil {
			t.Errorf("profile %q: Check refuses what its vocabulary built: %v", name, err)
		}
	}
	if _, ok := For("nope"); ok {
		t.Error("an unregistered name has a vocabulary")
	}
}

var (
	rows = []input.Row{
		{Key: "identifier", Value: "ID-1", Line: 2},
		{Key: "title", Lang: "nl", Value: "Kat", Line: 3},
		{Key: "title", Lang: "en", Value: "Cat", Line: 4},
	}
	want = []sip.Term{
		{Key: "identifier", Value: "ID-1"},
		{Key: "title", Lang: "nl", Value: "Kat"},
		{Key: "title", Lang: "en", Value: "Cat"},
	}
)

// Rows become the profile's statements as stated, in row order, so a term
// error's index names the row.
func TestRowsBecomeStatementsInOrder(t *testing.T) {
	tests := []struct {
		name  string
		vocab input.Vocabulary
		terms func(sip.Description) ([]sip.Term, bool)
	}{
		{"basic", Meemoo{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(meemoo.Terms); return tt, ok }},
		{"eark", Eark{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(eark.Terms); return tt, ok }},
		{"eark-mods", EarkMods{}, func(d sip.Description) ([]sip.Term, bool) { r, ok := d.(earkmods.Record); return r.Terms, ok }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, findings := tt.vocab.Description(rows, nil)
			if len(findings) != 0 {
				t.Fatalf("findings %v for rows every vocabulary places", findings)
			}
			got, ok := tt.terms(description)
			if !ok {
				t.Fatalf("description is %T", description)
			}
			if len(got) != len(want) {
				t.Fatalf("got %d statements, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("statement %d = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

// A profile whose document has no place for a record's copies reports item
// rows as one finding about the file, never silently drops them.
func TestItemsWithoutAPlaceAreAFinding(t *testing.T) {
	items := []input.ItemRow{{CallNumber: "A", Line: 2}}
	for _, tt := range []struct {
		name  string
		vocab input.Vocabulary
	}{{"basic", Meemoo{}}, {"eark", Eark{}}, {"eark-mods", EarkMods{}}} {
		_, findings := tt.vocab.Description(nil, items)
		if len(findings) != 1 || findings[0].Line != 0 || !strings.Contains(findings[0].Err.Error(), "items.csv") {
			t.Errorf("%s: findings = %v, want one about items.csv", tt.name, findings)
		}
	}
}
