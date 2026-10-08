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

// Definition.ValidateSource refuses a description of another standard,
// before the description is validated and before anything is written. It
// does so at package and representation level alike. The cases are a type
// no profile writes, Meemoo terms handed to ugent/basic, Simple DC terms
// handed to meemoo/basic or to ugent/bibliographic, and a MODS record
// handed to either DC profile.
func TestBuildRejectsDescriptionOfAnotherStandard(t *testing.T) {
	cases := []struct {
		name string
		def  build.Definition
		desc sip.Description
		want string
	}{
		{"unknown type to ugent/basic", ugentBasicDef(t), otherDescription{}, "ugent.Terms"},
		{"Meemoo terms to ugent/basic", ugentBasicDef(t), testDescription(), "meemoo.Terms, not Simple Dublin Core"},
		{"simple dc terms to basic", basicDef(t), identityTerms(), "ugent.Terms, not Meemoo dc+schema"},
		{"simple dc terms to ugent/bibliographic", bibliographicDef(t), identityTerms(), "ugent.Terms, not a MODS record"},
		{"record to ugent/basic", ugentBasicDef(t), identityRecord(), "ugent.Record, not Simple Dublin Core"},
		{"record to basic", basicDef(t), identityRecord(), "ugent.Record, not Meemoo dc+schema"},
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

	b, in, outDir := newTestBuilder(t, ugentBasicDef(t))
	in.Description = identityTerms()
	in.Representations[0].Description = otherDescription{}
	_, err := b.Build(in)
	if err == nil || !strings.Contains(err.Error(), `representation "archival"`) {
		t.Fatalf("Build error = %v, want the mismatch naming the representation", err)
	}
	requireEmpty(t, outDir)
}
