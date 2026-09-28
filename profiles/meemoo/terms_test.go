package meemoo

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

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := (dcschema{}).Encode(&buf, testTerms(), "../../schemas"); err != nil {
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

	// "../../schemas" resolves from metadata/descriptive/.
	if !strings.Contains(out, "../../schemas/descriptive_basic.xsd") {
		t.Error("package-level schema location hint missing")
	}
}

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestEncodeSchemaLocation(t *testing.T) {
	var buf bytes.Buffer
	if err := (dcschema{}).Encode(&buf, testTerms(), "../../../../schemas"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `xsi:schemaLocation="https://data.hetarchief.be/id/sip/1.2/basic ../../../../schemas/descriptive_basic.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", buf.String())
	}
}

func TestEncodeRefusesInvalid(t *testing.T) {
	bad := Terms{{Element: "dcterms:titel", Value: "x"}}
	var buf bytes.Buffer
	if err := (dcschema{}).Encode(&buf, bad, "../../schemas"); err == nil {
		t.Fatal("Encode accepted an invalid element")
	}
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite refusing", buf.Len())
	}
}

func TestTermValidate(t *testing.T) {
	tests := []struct {
		name string
		term Term
		want string // "" means valid; else substring of the error
	}{
		{"valid plain", Term{Element: "dcterms:title", Value: "x"}, ""},
		{"valid schema with lang", Term{Element: "schema:artform", Lang: "nl-BE", Value: "x"}, ""},
		{"valid new key element", Term{Element: "dcterms:abstract", Value: "x"}, ""},
		{"unprefixed", Term{Element: "title", Value: "x"}, "not in the descriptive vocabulary"},
		{"misspelled dcterms", Term{Element: "dcterms:titel", Value: "x"}, "not in the descriptive vocabulary"},
		// a real DCMI term meemoo's profile excludes; the old DCMI-55
		// membership check accepted it
		{"dcterms outside the profile", Term{Element: "dcterms:accrualPolicy", Value: "x"}, "not in the descriptive vocabulary"},
		// schema.org is no longer an open passthrough
		{"schema outside the profile", Term{Element: "schema:duration", Value: "x"}, "not in the descriptive vocabulary"},
		{"unknown prefix", Term{Element: "foo:bar", Value: "x"}, "not in the descriptive vocabulary"},
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
// with the object identifier. Assemble must follow that order.
func TestTermsIdentifierSwap(t *testing.T) {
	terms := testTerms()

	if got := terms.localIdentifier(); got != "BIB.FA.2026.001" {
		t.Fatalf("localIdentifier = %q", got)
	}

	terms.setObjectIdentifier("uuid-entity-1")
	if got := terms.localIdentifier(); got != "uuid-entity-1" {
		t.Fatalf("identifier not swapped in place, got %q", got)
	}

	var buf bytes.Buffer
	if err := (dcschema{}).Encode(&buf, terms, "../../schemas"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "<dcterms:identifier>uuid-entity-1</dcterms:identifier>") {
		t.Error("encoded document does not carry the object identifier")
	}
}

// Validate applies meemoo's own rules, not only term validity: the table's
// cardinality limits and a Dutch entry wherever an element is
// language-tagged, with every finding reported at once.
func TestTermsValidateAppliesMeemooRules(t *testing.T) {
	terms := append(testTerms(),
		Term{Element: "dcterms:created", Value: "1914"},
		Term{Element: "dcterms:abstract", Lang: "en", Value: "About"})
	err := terms.Validate()
	if err == nil {
		t.Fatal("want the repeated created and the abstract without Dutch refused")
	}
	for _, want := range []string{"dcterms:created appears more than once", `dcterms:abstract carries language-tagged values but none in "nl"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	if err := testTerms().Validate(); err != nil {
		t.Fatalf("conformant terms refused: %v", err)
	}
}
