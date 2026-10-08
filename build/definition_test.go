package build_test

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/profiles/ugent"
	"github.com/ugent-library/sip-creator/sip"
)

func TestWithSubmitterMeemoo(t *testing.T) {
	def := basicDef(t)

	got, err := def.WithSubmitter("Example Organization", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	agents := got.Declaration.Agents
	if len(agents) != len(def.Declaration.Agents)+1 {
		t.Fatalf("agents = %d, want %d", len(agents), len(def.Declaration.Agents)+1)
	}
	sub := agents[len(agents)-1]
	if sub.Role != "CREATOR" || sub.Type != "ORGANIZATION" {
		t.Errorf("submitter agent role/type = %q/%q, want CREATOR/ORGANIZATION", sub.Role, sub.Type)
	}
	if sub.Name != "Example Organization" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "OR-a1b2c3d" || sub.NoteType != "IDENTIFICATIONCODE" {
		t.Errorf("submitter note = %q (%q), want the OR-id as IDENTIFICATIONCODE", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterMeemooRequiresORID(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("Example Organization", ""); err == nil {
		t.Fatal("WithSubmitter() with empty OR-id on a Meemoo profile: want error, got nil")
	}
}

// The submitter's name and OR-id are written into the package METS, so
// they are held to what XML can carry.
func TestWithSubmitterRefusesTextXMLCannotCarry(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("Example\x01Organization", "OR-a1b2c3d"); err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("WithSubmitter() with a control character in the name: error = %v", err)
	}
	if _, err := basicDef(t).WithSubmitter("Example Organization", "OR-\x1b"); err == nil || !strings.Contains(err.Error(), "OR-id") {
		t.Errorf("WithSubmitter() with a control character in the OR-id: error = %v", err)
	}
}

func TestWithSubmitterRequiresName(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("", "OR-a1b2c3d"); err == nil {
		t.Fatal("WithSubmitter() with empty name: want error, got nil")
	}
}

func TestWithSubmitterUGent(t *testing.T) {
	def, ok := profiles.Get("ugent/basic")
	if !ok {
		t.Fatal(`no "ugent/basic" definition registered`)
	}

	// The OR-id is a Meemoo concept. ugent/basic ignores a configured value.
	got, err := def.WithSubmitter("Example Organization", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	sub := got.Declaration.Agents[len(got.Declaration.Agents)-1]
	if sub.Name != "Example Organization" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "" || sub.NoteType != "" {
		t.Errorf("ugent/basic submitter note = %q (%q), want none", sub.Note, sub.NoteType)
	}
}

// WithSubmitter returns a copy whose agents never share storage with the
// definition it was called on. The definition here has spare capacity
// after its agents, as one a program builds with append often has. Without
// the copy, two calls would append into the same array, and the second
// submitter would overwrite the first.
func TestWithSubmitterCopiesTheAgents(t *testing.T) {
	def := basicDef(t)
	agents := make([]sip.Agent, len(def.Declaration.Agents), len(def.Declaration.Agents)+4)
	copy(agents, def.Declaration.Agents)
	def.Declaration.Agents = agents

	first, err := def.WithSubmitter("First Organization", "OR-a1b2c3d")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := def.WithSubmitter("Second Organization", "OR-e4f5g6h"); err != nil {
		t.Fatal(err)
	}

	if got := first.Declaration.Agents[len(first.Declaration.Agents)-1].Name; got != "First Organization" {
		t.Errorf("first definition's submitter = %q after a second call, want %q", got, "First Organization")
	}
	if len(def.Declaration.Agents) != len(agents) {
		t.Errorf("the definition WithSubmitter was called on has %d agents, want %d", len(def.Declaration.Agents), len(agents))
	}
}

// meemoo/basic allows one representation and no description below the
// package level. ugent/basic allows both. Build refuses what
// ValidateSource refuses, before anything is written.
func TestValidateSourceAppliesProfileRules(t *testing.T) {
	b, in, outDir := newTestBuilder(t, basicDef(t))
	access := in.Representations[0]
	access.Name = "access"
	access.Description = meemoo.Terms{{Key: "dcterms:title", Lang: "nl", Value: "Toegangskopie"}}
	in.Representations = append(in.Representations, access)

	err := basicDef(t).ValidateSource(in)
	for _, want := range []string{"at most 1 representation(s), the package has 2", `representation "access" has a description`} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ValidateSource = %v, want a finding mentioning %q", err, want)
		}
	}
	if _, err := b.Build(in); err == nil {
		t.Error("Build accepted what ValidateSource refused")
	}
	requireEmpty(t, outDir)

	in.Description = identityTerms()
	in.Representations[1].Description = ugent.Terms{{Key: "title", Value: "Access copy"}}
	if err := ugentBasicDef(t).ValidateSource(in); err != nil {
		t.Errorf("ugent/basic has no such rules, yet ValidateSource refused: %v", err)
	}
}

// Under a vocabulary of representation types a representation's name must
// be one of the set, and a type, when given, must equal the name. Without
// a vocabulary, as under meemoo/basic, neither rule applies.
func TestValidateSourceAppliesRepresentationTypes(t *testing.T) {
	cases := []struct {
		name, repName, repType string
		want                   string // "" when the source is accepted
	}{
		{"name in the set", "archival", "", ""},
		{"type equal to the name", "access", "access", ""},
		{"name outside the set", "master", "", `profile "ugent/basic" names its representations preservation, archival, access; "master" is not one of them`},
		{"type other than the name", "archival", "preservation", `under profile "ugent/basic" a representation's type is its name; "archival" has type "preservation"`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, in, _ := newTestBuilder(t, ugentBasicDef(t))
			in.Description = identityTerms()
			in.Representations[0].Name = tt.repName
			in.Representations[0].Type = tt.repType
			err := ugentBasicDef(t).ValidateSource(in)
			switch {
			case tt.want == "" && err != nil:
				t.Errorf("ValidateSource = %v, want the source accepted", err)
			case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
				t.Errorf("ValidateSource = %v, want %q", err, tt.want)
			}
		})
	}

	_, in, _ := newTestBuilder(t, basicDef(t))
	in.Representations[0].Name = "master"
	in.Representations[0].Type = "Master scan"
	if err := basicDef(t).ValidateSource(in); err != nil {
		t.Errorf("meemoo/basic has no vocabulary, yet ValidateSource refused: %v", err)
	}
}

// meemoo/basic needs one representation. The UGent profiles accept a
// package without any.
func TestValidateSourceAppliesMinRepresentations(t *testing.T) {
	_, in, _ := newTestBuilder(t, basicDef(t))
	in.Representations = nil
	want := `profile "meemoo/basic" needs at least 1 representation(s), the package has 0`
	if err := basicDef(t).ValidateSource(in); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("ValidateSource = %v, want %q", err, want)
	}

	in.Description = identityTerms()
	if err := ugentBasicDef(t).ValidateSource(in); err != nil {
		t.Errorf("ugent/basic refused a package without representations: %v", err)
	}
	in.Description = identityRecord()
	if err := bibliographicDef(t).ValidateSource(in); err != nil {
		t.Errorf("ugent/bibliographic refused a package without representations: %v", err)
	}
}
