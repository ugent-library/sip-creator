package meemoo

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

func testTerms() Terms {
	return Terms{
		{Key: "identifier", Value: "example-0001"},
		{Key: "title", Lang: "nl", Value: "Fotoalbum 2026"},
		{Key: "created", Value: "1913"},
		{Key: "subject", Lang: "nl", Value: "R&D <scans>"},
		{Key: "artmedium", Lang: "nl", Value: "zilvergelatinedruk"},
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
		"<dcterms:identifier>example-0001</dcterms:identifier>",
		`<dcterms:title xml:lang="nl">Fotoalbum 2026</dcterms:title>`,
		// the meemoo document types its dates as EDTF, as dc+schema does
		`<dcterms:created xsi:type="edtf:EDTF-level1">1913</dcterms:created>`,
		// operator values are arbitrary text and must be escaped
		"<dcterms:subject xml:lang=\"nl\">R&amp;D &lt;scans&gt;</dcterms:subject>",
		// the key maps to the camel-cased schema.org element
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
	bad := Terms{{Key: "titel", Value: "x"}}
	var buf bytes.Buffer
	if err := (dcschema{}).Encode(&buf, bad, "../../schemas"); err == nil {
		t.Fatal("Encode accepted an unknown key")
	}
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite refusing", buf.Len())
	}
}

func TestValidateTerm(t *testing.T) {
	tests := []struct {
		name string
		term sip.Term
		want string // "" means valid; else substring of the error
	}{
		{"valid plain", sip.Term{Key: "title", Value: "x"}, ""},
		{"valid schema.org key with lang", sip.Term{Key: "artform", Lang: "nl-BE", Value: "x"}, ""},
		{"valid abstract", sip.Term{Key: "abstract", Value: "x"}, ""},
		{"typo", sip.Term{Key: "titel", Value: "x"}, "unknown key"},
		// a real DCMI term meemoo's profile excludes
		{"dcterms outside the profile", sip.Term{Key: "accrualpolicy", Value: "x"}, "unknown key"},
		// schema.org is not an open passthrough
		{"schema outside the profile", sip.Term{Key: "duration", Value: "x"}, "unknown key"},
		// a key is the plain word, never the element name it emits
		{"element name as key", sip.Term{Key: "dcterms:title", Value: "x"}, "unknown key"},
		// keys are lowercase; case folding is the rows file's convention
		{"capitalized", sip.Term{Key: "Title", Value: "x"}, "unknown key"},
		{"bad lang", sip.Term{Key: "title", Lang: "nl!", Value: "x"}, "not a language tag"},
		{"empty value", sip.Term{Key: "subject", Value: "  "}, "empty value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTerm(tt.term)
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
		{Key: "identifier", Value: "A"},
		{Key: "identifier", Value: "B"},
	}
	err := terms.Validate()
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("want the exactly-one identifier rule, got %v", err)
	}
}

// The identifier swap: read the local identifier first, then replace it
// with the object identifier. Swap must follow that order.
func TestTermsIdentifierSwap(t *testing.T) {
	terms := testTerms()

	if got := terms.localIdentifier(); got != "example-0001" {
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
// cardinality limits and a Dutch entry wherever a key is language-tagged,
// with every finding reported at once and named by key.
func TestTermsValidateAppliesMeemooRules(t *testing.T) {
	terms := append(testTerms(),
		sip.Term{Key: "created", Value: "1914"},
		sip.Term{Key: "abstract", Lang: "en", Value: "About"})
	err := terms.Validate()
	if err == nil {
		t.Fatal("want the repeated created and the abstract without Dutch refused")
	}
	for _, want := range []string{"created appears more than once", `abstract carries language-tagged values but none in "nl"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	if err := testTerms().Validate(); err != nil {
		t.Fatalf("conformant terms refused: %v", err)
	}
}
