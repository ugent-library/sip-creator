package build_test

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// otherDescription stands in for a description of a standard no registered
// profile writes.
type otherDescription struct{}

func (otherDescription) Validate() error         { return nil }
func (otherDescription) ValidateRequired() error { return nil }

// A description of another standard is refused by the profile's
// descriptive-standard check before validation and before any side effect,
// at package and representation level alike: a type no profile writes,
// Meemoo terms handed to eark, Simple DC terms handed to basic or to
// eark-mods, a MODS record handed to either DC profile.
func TestBuildRejectsDescriptionOfAnotherStandard(t *testing.T) {
	cases := []struct {
		name string
		def  build.Definition
		desc sip.Description
		want string
	}{
		{"unknown type to eark", earkDef(t), otherDescription{}, "eark.Terms"},
		{"Meemoo terms to eark", earkDef(t), testDescription(), "meemoo.Terms, not Simple Dublin Core"},
		{"simple dc terms to basic", basicDef(t), identityTerms(), "eark.Terms, not Meemoo dc+schema"},
		{"simple dc terms to eark-mods", earkmodsDef(t), identityTerms(), "eark.Terms, not a MODS record"},
		{"record to eark", earkDef(t), identityRecord(), "earkmods.Record, not Simple Dublin Core"},
		{"record to basic", basicDef(t), identityRecord(), "earkmods.Record, not Meemoo dc+schema"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, c.def)
			in.Description = c.desc
			_, err := b.Build(in)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Build error = %v, want the mismatch mentioning %q", err, c.want)
			}
			requireEmpty(t, outDir)
		})
	}

	b, in, outDir := newTestBuilder(t, earkDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = otherDescription{}
	_, err := b.Build(in)
	if err == nil || !strings.Contains(err.Error(), `representation "master"`) {
		t.Fatalf("Build error = %v, want the mismatch naming the representation", err)
	}
	requireEmpty(t, outDir)
}
