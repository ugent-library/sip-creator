package metadata

import (
	"bytes"
	"strings"
	"testing"
)

func testTerms() Terms {
	return Terms{
		{Element: "dcterms:identifier", Value: "BIB.FA.2026.001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Fotoalbum Gent 1913"},
		{Element: "dcterms:created", Value: "1913"},
		{Element: "dcterms:subject", Lang: "nl", Value: "R&D <scans>"},
		{Element: "schema:artMedium", Lang: "nl", Value: "zilvergelatinedruk"},
	}
}

func TestTermsEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := testTerms().Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`<metadata xmlns="https://data.hetarchief.be/id/sip/1.2/basic"`,
		"<dcterms:identifier>BIB.FA.2026.001</dcterms:identifier>",
		`<dcterms:title xml:lang="nl">Fotoalbum Gent 1913</dcterms:title>`,
		// the meemoo document types its dates as EDTF, as dc+schema does
		`<dcterms:created xsi:type="edtf:EDTF-level1">1913</dcterms:created>`,
		// operator values are arbitrary text and must be escaped
		"<dcterms:subject xml:lang=\"nl\">R&amp;D &lt;scans&gt;</dcterms:subject>",
		`<schema:artMedium xml:lang="nl">zilvergelatinedruk</schema:artMedium>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %s\n%s", want, out)
		}
	}

	// Term order is the producer's order.
	if strings.Index(out, "dcterms:title") > strings.Index(out, "dcterms:created") {
		t.Error("term order not preserved")
	}

	// Encode is the package-level default; the schema-location hint must
	// resolve from metadata/descriptive/.
	if !strings.Contains(out, "../../schemas/descriptive_basic.xsd") {
		t.Error("package-level schema location hint missing")
	}
}

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestEncodeTermsSchemaLocation(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeTerms(&buf, testTerms(), "../../../../schemas"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic ../../../../schemas/descriptive_basic.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", buf.String())
	}
}

func TestTermsEncodeRefusesInvalid(t *testing.T) {
	bad := Terms{{Element: "dcterms:titel", Value: "x"}}
	var buf bytes.Buffer
	if err := bad.Encode(&buf); err == nil {
		t.Fatal("Encode accepted an invalid element")
	}
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite refusing", buf.Len())
	}
}

func TestEncodeDCTerms(t *testing.T) {
	terms := Terms{
		{Element: "dcterms:identifier", Value: "uuid-x"},
		{Element: "dcterms:alternative", Value: "Alt"},
		{Element: "dcterms:created", Value: "1913"},
		{Element: "dcterms:abstract", Lang: "en", Value: "About"},
		{Element: "dcterms:isPartOf", Value: "Collectie Sacré"},
		{Element: "dcterms:spatial", Value: "Gent"},
		{Element: "dcterms:license", Value: "publiek domein"},
		{Element: "dcterms:extent", Value: "48 foto's"},
		{Element: "dcterms:rightsHolder", Value: "UGent"},      // no Simple DC home
		{Element: "schema:artMedium", Value: "zilvergelatine"}, // no Simple DC home
	}

	var buf bytes.Buffer
	if err := EncodeDCTerms(&buf, terms, PackageSchemas); err != nil {
		t.Fatalf("EncodeDCTerms: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"<simpledc",
		`xsi:noNamespaceSchemaLocation="../../schemas/dc.xsd"`,
		"<identifier>uuid-x</identifier>",
		"<title>Alt</title>",                   // alternative → title
		"<date>1913</date>",                    // created → date
		"<description>About</description>",     // abstract → description
		"<relation>Collectie Sacré</relation>", // isPartOf → relation
		"<coverage>Gent</coverage>",            // spatial → coverage
		"<rights>publiek domein</rights>",      // license → rights
		"<format>48 foto&#39;s</format>",       // extent → format, escaped
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %s\n%s", want, out)
		}
	}
	for _, banned := range []string{"UGent", "zilvergelatine", "dcterms:", "hetarchief"} {
		if strings.Contains(out, banned) {
			t.Errorf("output must not carry %s (no Simple DC home)\n%s", banned, out)
		}
	}
}

func TestTermValidate(t *testing.T) {
	tests := []struct {
		name string
		term Term
		want string // "" means valid; else substring of the error
	}{
		{"valid plain", Term{Element: "dcterms:title", Value: "x"}, ""},
		{"valid schema with lang", Term{Element: "schema:artForm", Lang: "nl-BE", Value: "x"}, ""},
		{"unprefixed", Term{Element: "title", Value: "x"}, "not a prefixed element"},
		{"misspelled dcterms", Term{Element: "dcterms:titel", Value: "x"}, "not a Dublin Core term"},
		{"bad schema shape", Term{Element: "schema:9bad", Value: "x"}, "not a schema.org property"},
		{"unknown prefix", Term{Element: "foo:bar", Value: "x"}, "unknown vocabulary prefix"},
		{"bad lang", Term{Element: "dcterms:title", Lang: "nl!", Value: "x"}, "not a language tag"},
		{"empty value", Term{Element: "dcterms:subject", Value: "  "}, "empty value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.term.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}

func TestTermsValidateDuplicateIdentifier(t *testing.T) {
	terms := Terms{
		{Element: "dcterms:identifier", Value: "A"},
		{Element: "dcterms:identifier", Value: "B"},
	}
	err := terms.Validate()
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("want the exactly-one identifier rule, got %v", err)
	}
}

// The identifier swap: read the local identifier first, then replace it
// with the object identifier — the order assemble must follow.
func TestTermsIdentifierSeams(t *testing.T) {
	terms := testTerms()

	if got := terms.LocalIdentifier("dcterms"); got != "BIB.FA.2026.001" {
		t.Fatalf("LocalIdentifier = %q", got)
	}
	if got := terms.LocalIdentifier("mods"); got != "" {
		t.Fatalf("unknown scheme must return empty, got %q", got)
	}

	terms.SetObjectIdentifier("uuid-entity-1")
	if got := terms.LocalIdentifier("dcterms"); got != "uuid-entity-1" {
		t.Fatalf("identifier not swapped in place, got %q", got)
	}

	var buf bytes.Buffer
	if err := terms.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "<dcterms:identifier>uuid-entity-1</dcterms:identifier>") {
		t.Error("encoded document does not carry the object identifier")
	}
}
