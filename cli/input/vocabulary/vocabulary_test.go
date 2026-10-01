package vocabulary

import (
	"encoding/xml"
	"errors"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Every registered profile has a vocabulary, and each builds what its
// profile's encoder accepts: the reader hands statements to the one, the
// engine runs Check on the result. A profile without a vocabulary could
// not read a folder.
func TestEveryProfileHasAVocabularyItsEncoderAccepts(t *testing.T) {
	for _, name := range profiles.Names() {
		def, _ := profiles.Get(name)
		vocab, ok := For(name)
		if !ok {
			t.Errorf("profile %q has no vocabulary", name)
			continue
		}
		description, errs := vocab.Description(nil)
		if len(errs) != 0 {
			t.Errorf("profile %q: no statements, yet errors %v", name, errs)
		}
		if err := def.Encoder.Check(description); err != nil {
			t.Errorf("profile %q: Check refuses what its vocabulary built: %v", name, err)
		}
	}
	if _, ok := For("nope"); ok {
		t.Error("an unregistered name has a vocabulary")
	}
}

// A vocabulary takes a supplied document exactly when its profile's
// encoder judges one, and names the file the package gives the document:
// the two eark profiles do, basic does not. A vocabulary on one side only
// would reserve a name the build refuses, or refuse a name the build takes.
func TestDocumentVocabulariesMatchTheirEncoders(t *testing.T) {
	for _, name := range profiles.Names() {
		def, _ := profiles.Get(name)
		vocab, _ := For(name)
		_, encoderTakes := def.Encoder.(build.DescriptiveDocumentChecker)
		docVocab, vocabTakes := vocab.(input.DocumentVocabulary)
		if encoderTakes != vocabTakes {
			t.Errorf("profile %q: its encoder takes a supplied document: %v; its vocabulary: %v", name, encoderTakes, vocabTakes)
			continue
		}
		if vocabTakes && docVocab.DocumentName() != def.DescriptiveName {
			t.Errorf("profile %q: document name %q, want the package's %q", name, docVocab.DocumentName(), def.DescriptiveName)
		}
	}
}

var (
	simpledcRoot = xml.StartElement{Name: xml.Name{Local: "simpledc"}}
	modsRoot     = xml.StartElement{
		Name: xml.Name{Space: "http://www.loc.gov/mods/v3", Local: "mods"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "version"}, Value: "3.7"}},
	}
)

// Each document vocabulary accepts its own standard's root and refuses the
// other's, as its encoder does.
func TestDocumentVocabulariesJudgeTheRoot(t *testing.T) {
	tests := []struct {
		name             string
		vocab            input.DocumentVocabulary
		accepts, refuses xml.StartElement
	}{
		{"eark", Eark{}, simpledcRoot, modsRoot},
		{"eark-mods", EarkMods{}, modsRoot, simpledcRoot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.vocab.CheckDocument(tt.accepts); err != nil {
				t.Errorf("refused its own standard's root: %v", err)
			}
			if err := tt.vocab.CheckDocument(tt.refuses); err == nil {
				t.Error("accepted another standard's root")
			}
		})
	}
}

var (
	statements = []input.Statement{
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

// In the flat worlds the statements become the profile's terms as stated,
// in statement order, so a term error's index names the row.
func TestStatementsBecomeTermsInOrder(t *testing.T) {
	tests := []struct {
		name  string
		vocab input.Vocabulary
		terms func(sip.Description) ([]sip.Term, bool)
	}{
		{"basic", Meemoo{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(meemoo.Terms); return tt, ok }},
		{"eark", Eark{}, func(d sip.Description) ([]sip.Term, bool) { tt, ok := d.(eark.Terms); return tt, ok }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, errs := tt.vocab.Description(statements)
			if len(errs) != 0 {
				t.Fatalf("errors %v for statements every vocabulary places", errs)
			}
			got, ok := tt.terms(description)
			if !ok {
				t.Fatalf("description is %T", description)
			}
			if len(got) != len(want) {
				t.Fatalf("got %d terms, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("term %d = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

// Under eark-mods the statements fill the record's fields: the identifier
// once and the titles in statement order with their language. The rows
// carry no items; those reach a record through the library or a supplied
// document.
func TestEarkModsStatementsFillTheRecord(t *testing.T) {
	description, errs := EarkMods{}.Description(statements)
	if len(errs) != 0 {
		t.Fatalf("errors %v for statements the vocabulary places", errs)
	}
	record, ok := description.(earkmods.Record)
	if !ok {
		t.Fatalf("description is %T, want earkmods.Record", description)
	}
	if record.Identifier != "ID-1" {
		t.Errorf("Identifier = %q", record.Identifier)
	}
	wantTitles := []earkmods.Title{{Value: "Kat", Lang: "nl"}, {Value: "Cat", Lang: "en"}}
	if len(record.Titles) != 2 || record.Titles[0] != wantTitles[0] || record.Titles[1] != wantTitles[1] {
		t.Errorf("Titles = %+v, want %+v", record.Titles, wantTitles)
	}
	if record.Items != nil {
		t.Errorf("Items = %+v, want none from rows", record.Items)
	}
}

// A statement the MODS vocabulary cannot place is a StatementError at its
// line, and the statement is not placed; the statements around it still
// are.
func TestEarkModsErrorsNameTheLine(t *testing.T) {
	tests := []struct {
		name       string
		statements []input.Statement
		want       string // substring of the one error
		line       int
	}{
		{"unknown key", []input.Statement{{Key: "abstract", Value: "x", Line: 5}}, `unknown key "abstract"`, 5},
		{"dc element as key", []input.Statement{{Key: "description", Value: "x", Line: 5}}, "unknown key", 5},
		{"mods element as key", []input.Statement{{Key: "titleinfo", Value: "x", Line: 5}}, "unknown key", 5},
		{"language on the identifier", []input.Statement{{Key: "identifier", Lang: "nl", Value: "x", Line: 5}}, "identifier takes no language tag", 5},
		{"empty value", []input.Statement{{Key: "title", Value: " ", Line: 5}}, "title has an empty value", 5},
		{"second identifier", append(statements, input.Statement{Key: "identifier", Value: "ID-2", Line: 5}), "identifier appears more than once (first on line 2)", 5},
		{"title repeated in one language", append(statements, input.Statement{Key: "title", Lang: "nl", Value: "Poes", Line: 5}), `title appears more than once in language "nl" (first on line 3)`, 5},
		{"title repeated untagged", []input.Statement{{Key: "title", Value: "a", Line: 2}, {Key: "title", Value: "b", Line: 3}}, "distinct language tags", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description, errs := EarkMods{}.Description(tt.statements)
			if len(errs) != 1 {
				t.Fatalf("errors = %v, want exactly one", errs)
			}
			var se *input.StatementError
			if !errors.As(errs[0], &se) {
				t.Fatalf("error is %T, want a *input.StatementError", errs[0])
			}
			if se.Line != tt.line || !strings.Contains(se.Err.Error(), tt.want) {
				t.Errorf("error = line %d %q, want line %d mentioning %q", se.Line, se.Err, tt.line, tt.want)
			}
			// The refused statement left no trace: the record holds the
			// statements before it and nothing else.
			record := description.(earkmods.Record)
			placed := len(record.Titles)
			if record.Identifier != "" {
				placed++
			}
			if placed != len(tt.statements)-1 {
				t.Errorf("record holds %d values, want the %d statements that were placed: %+v", placed, len(tt.statements)-1, record)
			}
		})
	}
}
