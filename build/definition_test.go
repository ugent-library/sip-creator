package build_test

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/earkmods"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// earkDef returns the registered "eark" definition the tests build with.
func earkDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}
	return def
}

// earkmodsDef returns the registered "eark-mods" definition the tests
// build with.
func earkmodsDef(t *testing.T) build.Definition {
	t.Helper()
	def, ok := profiles.Get("eark-mods")
	if !ok {
		t.Fatal(`no "eark-mods" definition registered`)
	}
	return def
}

// identityTerms is the input convention's own MUSTs and nothing more, in
// the eark profile's standard, Simple Dublin Core.
func identityTerms() eark.Terms {
	return eark.Terms{
		{Key: "identifier", Value: "local-id-001"},
		{Key: "title", Value: "Catus Testus"},
	}
}

// identityRecord is the same identity in the eark-mods profile's standard,
// a MODS record without items.
func identityRecord() earkmods.Record {
	return earkmods.Record{
		Identifier: "local-id-001",
		Titles:     []earkmods.Title{{Value: "Catus Testus"}},
	}
}

// meemooIdentityTerms is the same identity in Meemoo's standard: short of
// the four keys the basic profile requires.
func meemooIdentityTerms() meemoo.Terms {
	return meemoo.Terms{
		{Key: "identifier", Value: "local-id-001"},
		{Key: "title", Lang: "nl", Value: "Catus Testus"},
	}
}

// Cardinality and the Dutch-language rule belong to Meemoo's standard, so
// build.SourcePackage.Validate applies them to Meemoo terms at both levels whatever
// the profile, and never to Simple Dublin Core terms.
func TestSourcePackageValidateAppliesStandardRules(t *testing.T) {
	_, in, _ := newTestBuilder(t, basicDef(t))
	in.Description = append(testDescription(),
		sip.Term{Key: "abstract", Lang: "nl", Value: "een"},
		sip.Term{Key: "abstract", Lang: "nl", Value: "twee"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Errorf("repeated abstract accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t, basicDef(t))
	in.Description = append(testDescription(), sip.Term{Key: "subject", Lang: "en", Value: "cats"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), `"nl"`) {
		t.Errorf("subject without a Dutch entry accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t, basicDef(t))
	in.Representations[0].Description = meemoo.Terms{{Key: "title", Lang: "en", Value: "Cats"}}
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), `representation "master"`) {
		t.Errorf("representation title without a Dutch entry accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t, basicDef(t))
	in.Description = append(identityTerms(),
		sip.Term{Key: "description", Lang: "en", Value: "one"},
		sip.Term{Key: "description", Lang: "en", Value: "two"})
	if err := in.Validate(); err != nil {
		t.Errorf("Simple DC has no such rules, yet Validate refused: %v", err)
	}
}

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

func TestWithSubmitterRequiresName(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("", "OR-a1b2c3d"); err == nil {
		t.Fatal("WithSubmitter() with empty name: want error, got nil")
	}
}

func TestWithSubmitterEARK(t *testing.T) {
	def, ok := profiles.Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}

	// The OR-id is a Meemoo concept; a configured value is ignored here.
	got, err := def.WithSubmitter("Example Organization", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	sub := got.Declaration.Agents[len(got.Declaration.Agents)-1]
	if sub.Name != "Example Organization" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "" || sub.NoteType != "" {
		t.Errorf("eark submitter note = %q (%q), want none", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterLeavesRegistryUntouched(t *testing.T) {
	before := len(basicDef(t).Declaration.Agents)

	if _, err := basicDef(t).WithSubmitter("Example Organization", "OR-a1b2c3d"); err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	after := basicDef(t)
	if len(after.Declaration.Agents) != before {
		t.Fatalf("registry agents = %d after WithSubmitter, want %d", len(after.Declaration.Agents), before)
	}
	for _, a := range after.Declaration.Agents {
		if a.Type == "ORGANIZATION" {
			t.Errorf("registry gained an ORGANIZATION agent: %+v", a)
		}
	}
}
