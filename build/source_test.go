package build_test

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// build.SourcePackage.Validate holds the rules for a source package however
// it was made: the rules the CLI's input reader reports as violations,
// checked again for a SourcePackage built directly in Go.
func TestSourcePackageValidate(t *testing.T) {
	valid := func(t *testing.T) *build.SourcePackage {
		_, in, _ := newTestBuilder(t, basicDef(t))
		return in
	}

	tests := []struct {
		name   string
		break_ func(*build.SourcePackage)
		want   string
	}{
		{"no descriptive", func(c *build.SourcePackage) { c.Description = nil }, "no descriptive metadata"},
		{"invalid term", func(c *build.SourcePackage) {
			c.Description = append(c.Description.(meemoo.Terms), sip.Term{Key: "dcterms:titel", Value: "x"})
		}, "unknown element"},
		{"no identifier", func(c *build.SourcePackage) {
			c.Description = meemoo.Terms{{Key: "dcterms:title", Value: "x"}}
		}, "identifier is required"},
		{"no title", func(c *build.SourcePackage) {
			c.Description = meemoo.Terms{{Key: "dcterms:identifier", Value: "x"}}
		}, "title is required"},
		{"no representations", func(c *build.SourcePackage) { c.Representations = nil }, "at least one version"},
		{"bad name", func(c *build.SourcePackage) { c.Representations[0].Name = "master copy" }, "may only contain"},
		{"dot name", func(c *build.SourcePackage) { c.Representations[0].Name = "." }, "outside representations/"},
		{"dot-dot name", func(c *build.SourcePackage) { c.Representations[0].Name = ".." }, "outside representations/"},
		{"label XML cannot carry", func(c *build.SourcePackage) { c.Representations[0].Label = "Master\x01" }, `label: "Master\x01" holds the character U+0001`},
		{"type XML cannot carry", func(c *build.SourcePackage) { c.Representations[0].Type = "a\x0bb" }, "type:"},
		{"content category XML cannot carry", func(c *build.SourcePackage) { c.ContentCategory = "\x1b" }, "content category:"},
		{"file path XML cannot carry", func(c *build.SourcePackage) { c.Representations[0].Files[0].Path = "a\x01b.jpg" }, "which XML cannot carry"},
		{"file path not UTF-8", func(c *build.SourcePackage) { c.Representations[0].Files[0].Path = "caf\xe9.jpg" }, "not valid UTF-8"},
		{"duplicate label", func(c *build.SourcePackage) {
			c.Representations = append(c.Representations, c.Representations[0])
		}, "supplied twice"},
		{"empty representation", func(c *build.SourcePackage) { c.Representations[0].Files = nil }, "no content files"},
		{"duplicate logical path", func(c *build.SourcePackage) {
			c.Representations[0].Files = append(c.Representations[0].Files, c.Representations[0].Files[0])
		}, "share the logical path"},
		{"file without source", func(c *build.SourcePackage) {
			c.Representations[0].Files[0].Source = ""
		}, "needs both a Source and a Path"},
		{"malformed package identifier", func(c *build.SourcePackage) {
			c.PackageIdentifier = "not-a-uuid"
		}, "uuid-<uuid> form"},
		{"record status outside the vocabulary", func(c *build.SourcePackage) {
			c.RecordStatus = "supplement"
		}, "SIP3 vocabulary"},
		{"update status without the updated package's identifier", func(c *build.SourcePackage) {
			c.RecordStatus = "REPLACEMENT"
		}, "PackageIdentifier must carry"},
		{"invalid representation descriptive", func(c *build.SourcePackage) {
			c.Representations[0].Description = meemoo.Terms{{Key: "dcterms:titel", Value: "x"}}
		}, "unknown element"},
		{"received premis claims the generated name", func(c *build.SourcePackage) {
			c.Premis = []build.SourceFile{{Source: "/x/premis.xml", Path: "premis.xml"}}
		}, "reserved for the generated"},
		{"rep received premis claims the generated name", func(c *build.SourcePackage) {
			c.Representations[0].Premis = []build.SourceFile{{Source: "/x/premis.xml", Path: "sub/premis.xml"}}
		}, "reserved for the generated"},
		{"received premis without a path", func(c *build.SourcePackage) {
			c.Premis = []build.SourceFile{{Source: "/x/events.xml"}}
		}, "package premis: a file needs both a Source and a Path"},
		{"duplicate documentation path", func(c *build.SourcePackage) {
			c.Documentation = []build.SourceFile{{Source: "/x/a.txt", Path: "notes.txt"}, {Source: "/y/b.txt", Path: "notes.txt"}}
		}, `documentation: two files share the logical path "notes.txt"`},
		{"rep documentation without a source", func(c *build.SourcePackage) {
			c.Representations[0].Documentation = []build.SourceFile{{Path: "notes.txt"}}
		}, `representation "master" documentation: a file needs both a Source and a Path`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid(t)
			tt.break_(in)
			err := in.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}

	if err := valid(t).Validate(); err != nil {
		t.Fatalf("valid source package rejected: %v", err)
	}
}

// Cardinality and the Dutch-language rule belong to Meemoo's standard, so
// build.SourcePackage.Validate applies them to Meemoo terms at both levels whatever
// the profile, and never to Simple Dublin Core terms.
func TestSourcePackageValidateAppliesStandardRules(t *testing.T) {
	_, in, _ := newTestBuilder(t, basicDef(t))
	in.Description = append(testDescription(),
		sip.Term{Key: "dcterms:abstract", Lang: "nl", Value: "een"},
		sip.Term{Key: "dcterms:abstract", Lang: "nl", Value: "twee"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Errorf("repeated abstract accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t, basicDef(t))
	in.Description = append(testDescription(), sip.Term{Key: "dcterms:subject", Lang: "en", Value: "cats"})
	if err := in.Validate(); err == nil || !strings.Contains(err.Error(), `"nl"`) {
		t.Errorf("subject without a Dutch entry accepted: %v", err)
	}

	_, in, _ = newTestBuilder(t, basicDef(t))
	in.Representations[0].Description = meemoo.Terms{{Key: "dcterms:title", Lang: "en", Value: "Cats"}}
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

// XML 1.0 carries every character except most control characters,
// U+FFFE and U+FFFF; a value must also be UTF-8. Escaping cannot carry the
// rest, so they are refused.
func TestValidateXMLText(t *testing.T) {
	tests := []struct {
		value string
		want  string // "" means accepted; else substring of the error
	}{
		{"", ""},
		{`R&D <a> "b" 'c'`, ""},
		{"two\tcolumns\nand lines\r\n", ""},
		{"caf\u00e9 \U0001F408 \uE000", ""},
		{"a\x00b", "U+0000"},
		{"a\x01b", "U+0001"},
		{"vertical\x0btab", "U+000B"},
		{"escape\x1b", "U+001B"},
		{"not a character \uFFFE", "U+FFFE"},
		{"caf\xe9", "not valid UTF-8"},
	}
	for _, tt := range tests {
		err := build.ValidateXMLText(tt.value)
		if tt.want == "" {
			if err != nil {
				t.Errorf("ValidateXMLText(%q) = %v, want accepted", tt.value, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("ValidateXMLText(%q) = %v, want an error mentioning %q", tt.value, err, tt.want)
		}
	}
}
