package profiles

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/dc"
	"github.com/ugent-library/sip-creator/encoders/dcschema"
	"github.com/ugent-library/sip-creator/schemas"
	"github.com/ugent-library/sip-creator/sip"
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

// identityTerms is the input convention's own MUSTs and nothing more, in
// the eark profile's standard, Simple Dublin Core.
func identityTerms() dc.Terms {
	return dc.Terms{
		{Element: "identifier", Value: "local-id-001"},
		{Element: "title", Value: "Catus Testus"},
	}
}

// meemooIdentityTerms is the same identity in meemoo's standard: short of
// the four elements the basic profile requires.
func meemooIdentityTerms() dcschema.Terms {
	return dcschema.Terms{
		{Element: "dcterms:identifier", Value: "local-id-001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Catus Testus"},
	}
}

// What each profile's spec requires on top of identity: identity-only
// terms satisfy eark and are refused under basic, which names every
// missing key at once, each with the element the meemoo table emits for
// it.
func TestValidateDescriptiveRequiredPerProfile(t *testing.T) {
	if err := earkDef(t).validateDescriptive(&Input{Descriptive: identityTerms()}); err != nil {
		t.Fatalf("eark refused identity-only terms: %v", err)
	}
	err := basicDef(t).validateDescriptive(&Input{Descriptive: meemooIdentityTerms()})
	if err == nil {
		t.Fatal("basic accepted terms without description and created")
	}
	for _, want := range []string{"description (dcterms:description)", "created (dcterms:created)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %s: %v", want, err)
		}
	}
}

// Identifier and title are every package's identity whatever the profile,
// checked once through the keys both worlds resolve: the meemoo world
// names the key and its element, Simple DC the key alone.
func TestValidateIdentity(t *testing.T) {
	for _, d := range []sip.Description{identityTerms(), meemooIdentityTerms()} {
		if err := ValidateIdentity(d); err != nil {
			t.Errorf("identity terms refused: %v", err)
		}
	}
	err := ValidateIdentity(dcschema.Terms{{Element: "dcterms:identifier", Value: "x"}})
	if err == nil || !strings.Contains(err.Error(), "title (dcterms:title) is required") {
		t.Errorf("meemoo terms without a title accepted: %v", err)
	}
	err = ValidateIdentity(dc.Terms{{Element: "title", Value: "x"}})
	if err == nil || !strings.Contains(err.Error(), "identifier is required") {
		t.Errorf("Simple DC terms without an identifier accepted: %v", err)
	}
}

// Every registry entry's required keys resolve in its own descriptive
// standard, so a typo in a definition fails here rather than at the first
// build. The sets themselves are pinned too: meemoo's basic content profile
// requires description and created on top of identity (meemoo SIP 1.2),
// plain E-ARK nothing. A new profile must be added to both maps.
func TestRegistryRequiredKeys(t *testing.T) {
	resolve := map[string]func(string) (string, bool){
		"basic": dcschema.ResolveKey,
		"eark":  dc.ResolveKey,
	}
	want := map[string][]string{
		"basic": {"description", "created"},
		"eark":  nil,
	}
	for _, name := range Names() {
		def, _ := Get(name)
		r, ok := resolve[name]
		if !ok {
			t.Fatalf("profile %q is not covered here; add its standard's ResolveKey and its required keys", name)
		}
		for _, key := range def.RequiredKeys {
			if _, ok := r(key); !ok {
				t.Errorf("profile %q requires key %q, which its descriptive standard does not know", name, key)
			}
		}
		if !slices.Equal(def.RequiredKeys, want[name]) {
			t.Errorf("profile %q RequiredKeys = %v, want %v", name, def.RequiredKeys, want[name])
		}
	}
}

// Cardinality and the Dutch-language rule are meemoo's standard's own, so
// Input.Validate applies them to dcschema terms at both levels whatever
// the profile, and never to Simple Dublin Core terms.
func TestInputValidateAppliesStandardRules(t *testing.T) {
	_, in, _ := newTestBuilder(t)
	in.Descriptive = append(testDescriptive(),
		dcschema.Term{Element: "dcterms:abstract", Lang: "nl", Value: "een"},
		dcschema.Term{Element: "dcterms:abstract", Lang: "nl", Value: "twee"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Errorf("repeated abstract accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t)
	in.Descriptive = append(testDescriptive(), dcschema.Term{Element: "dcterms:subject", Lang: "en", Value: "cats"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), `"nl"`) {
		t.Errorf("subject without a Dutch entry accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t)
	in.Representations[0].Descriptive = dcschema.Terms{{Element: "dcterms:title", Lang: "en", Value: "Cats"}}
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), `representation "master"`) {
		t.Errorf("representation title without a Dutch entry accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t)
	in.Descriptive = append(identityTerms(),
		dc.Term{Element: "description", Lang: "en", Value: "one"},
		dc.Term{Element: "description", Lang: "en", Value: "two"})
	if err := in.Validate(); err != nil {
		t.Errorf("Simple DC has no such rules, yet Validate refused: %v", err)
	}
}

// Every registry entry's schema list names bundled files, so a typo fails
// here rather than at the first build (a name may repeat across the encoder
// lists a profile concatenates; the assembler ships it once). Both profiles
// ship the whole bundle today; a profile shipping a subset gets its own
// line.
func TestRegistrySchemas(t *testing.T) {
	bundle := schemas.Get()
	for _, name := range Names() {
		def, _ := Get(name)
		for _, xsd := range def.Schemas {
			if _, ok := bundle[xsd]; !ok {
				t.Errorf("profile %q ships %q, which is not bundled", name, xsd)
			}
		}
	}

	all := slices.Sorted(maps.Keys(bundle))
	for _, name := range []string{"basic", "eark"} {
		def, _ := Get(name)
		if got := slices.Compact(slices.Sorted(slices.Values(def.Schemas))); !slices.Equal(got, all) {
			t.Errorf("profile %q Schemas = %v, want the whole bundle %v", name, got, all)
		}
	}
}

func TestWithSubmitterMeemoo(t *testing.T) {
	def := basicDef(t)

	got, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d")
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

	sub := got.Declaration.Agents[len(got.Declaration.Agents)-1]
	if sub.Name != "Universiteitsbibliotheek Gent" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "" || sub.NoteType != "" {
		t.Errorf("eark submitter note = %q (%q), want none", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterLeavesRegistryUntouched(t *testing.T) {
	before := len(basicDef(t).Declaration.Agents)

	if _, err := basicDef(t).WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d"); err != nil {
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
