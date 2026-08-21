package profiles

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/metadata"
)

// earkDef returns the registered "eark" definition the tests build with.
func earkDef(t *testing.T) Definition {
	t.Helper()
	def, ok := Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}
	return def
}

// identityTerms is the input convention's own MUSTs and nothing more:
// enough for eark, short of meemoo's four.
func identityTerms() metadata.Terms {
	return metadata.Terms{
		{Element: "dcterms:identifier", Value: "local-id-001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Catus Testus"},
	}
}

// The per-family required sets: identity-only terms satisfy eark and are
// refused under basic, which names every missing element at once.
func TestValidateDescriptiveRequiredPerFamily(t *testing.T) {
	in := &Input{Descriptive: identityTerms()}

	if err := earkDef(t).validateDescriptive(in); err != nil {
		t.Fatalf("eark refused identity-only terms: %v", err)
	}
	err := basicDef(t).validateDescriptive(in)
	if err == nil {
		t.Fatal("basic accepted terms without description and created")
	}
	for _, want := range []string{"dcterms:description", "dcterms:created"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %s: %v", want, err)
		}
	}

	// A title is required by both families' sets; since Input.Validate no
	// longer checks it, the Definition is the only guard.
	in = &Input{Descriptive: metadata.Terms{{Element: "dcterms:identifier", Value: "x"}}}
	for _, def := range []Definition{basicDef(t), earkDef(t)} {
		if err := def.validateDescriptive(in); err == nil || !strings.Contains(err.Error(), "dcterms:title") {
			t.Errorf("%s accepted terms without a title: %v", def.Name, err)
		}
	}
}

// The cardinality marks bind the meemoo family only: a repeated abstract
// (same language) fails basic, at package and representation level, and
// passes eark.
func TestValidateDescriptiveCardinalityPerFamily(t *testing.T) {
	repeated := metadata.Terms{
		{Element: "dcterms:abstract", Lang: "nl", Value: "een"},
		{Element: "dcterms:abstract", Lang: "nl", Value: "twee"},
	}
	in := &Input{Descriptive: append(testDescriptive(), repeated...)}

	if err := earkDef(t).validateDescriptive(in); err != nil {
		t.Fatalf("eark enforced a meemoo cardinality mark: %v", err)
	}
	err := basicDef(t).validateDescriptive(in)
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("basic did not refuse the repeated abstract: %v", err)
	}

	in = &Input{
		Descriptive:     testDescriptive(),
		Representations: []SourceRepresentation{{Label: "master", Descriptive: repeated}},
	}
	err = basicDef(t).validateDescriptive(in)
	if err == nil || !strings.Contains(err.Error(), `representation "master"`) {
		t.Fatalf("basic did not refuse the representation-level repeat: %v", err)
	}
}

// The required language is meemoo family data: lang-tagged elements
// without a Dutch entry fail basic and pass eark.
func TestValidateDescriptiveRequiredLangPerFamily(t *testing.T) {
	terms := testDescriptive()
	terms = append(terms, metadata.Term{Element: "dcterms:subject", Lang: "en", Value: "cats"})
	in := &Input{Descriptive: terms}

	if err := earkDef(t).validateDescriptive(in); err != nil {
		t.Fatalf("eark enforced a required language: %v", err)
	}
	err := basicDef(t).validateDescriptive(in)
	if err == nil || !strings.Contains(err.Error(), `"nl"`) {
		t.Fatalf("basic did not demand the Dutch entry: %v", err)
	}
}

func TestWithSubmitterMeemoo(t *testing.T) {
	def := basicDef(t)

	got, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	agents := got.Mets.Agents
	if len(agents) != len(def.Mets.Agents)+1 {
		t.Fatalf("agents = %d, want %d", len(agents), len(def.Mets.Agents)+1)
	}
	sub := agents[len(agents)-1]
	if sub.Role != "CREATOR" || sub.Type != "ORGANIZATION" {
		t.Errorf("submitter agent role/type = %q/%q, want CREATOR/ORGANIZATION", sub.Role, sub.Type)
	}
	if sub.Name != "Universiteitsbibliotheek Gent" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "OR-a1b2c3d" || sub.NoteType != "IDENTIFICATIONCODE" {
		t.Errorf("submitter note = %q (%q), want the OR-id as IDENTIFICATIONCODE", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterMeemooRequiresORID(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("Universiteitsbibliotheek Gent", ""); err == nil {
		t.Fatal("WithSubmitter() with empty OR-id on a meemoo profile: want error, got nil")
	}
}

func TestWithSubmitterRequiresName(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("", "OR-a1b2c3d"); err == nil {
		t.Fatal("WithSubmitter() with empty name: want error, got nil")
	}
}

func TestWithSubmitterEARK(t *testing.T) {
	def, ok := Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}

	// The OR-id is a meemoo concept; a configured value is ignored here.
	got, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	sub := got.Mets.Agents[len(got.Mets.Agents)-1]
	if sub.Name != "Universiteitsbibliotheek Gent" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "" || sub.NoteType != "" {
		t.Errorf("eark submitter note = %q (%q), want none", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterLeavesRegistryUntouched(t *testing.T) {
	before := len(basicDef(t).Mets.Agents)

	if _, err := basicDef(t).WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d"); err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	after := basicDef(t)
	if len(after.Mets.Agents) != before {
		t.Fatalf("registry agents = %d after WithSubmitter, want %d", len(after.Mets.Agents), before)
	}
	for _, a := range after.Mets.Agents {
		if a.Type == "ORGANIZATION" {
			t.Errorf("registry gained an ORGANIZATION agent: %+v", a)
		}
	}
}
