package dc

import (
	"bytes"
	"strings"
	"testing"
)

func testTerms() Terms {
	return Terms{
		{Element: "identifier", Value: "uuid-x"},
		{Element: "title", Lang: "nl", Value: "Fotoalbum Gent 1913"},
		{Element: "date", Value: "1913"},
		{Element: "format", Value: "48 foto's"},
		{Element: "subject", Value: "R&D <scans>"},
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, testTerms(), "../../schemas"); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"<simpledc",
		`xsi:noNamespaceSchemaLocation="../../schemas/dc.xsd"`,
		"<identifier>uuid-x</identifier>",
		// language tags are accepted but not emitted
		"<title>Fotoalbum Gent 1913</title>",
		"<date>1913</date>",
		// operator values are arbitrary text and must be escaped
		"<format>48 foto&#39;s</format>",
		"<subject>R&amp;D &lt;scans&gt;</subject>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %s\n%s", want, out)
		}
	}
	if strings.Contains(out, "xml:lang") {
		t.Errorf("simpledc must not carry xml:lang\n%s", out)
	}
	// Term order is the producer's order.
	if strings.Index(out, "<title>") > strings.Index(out, "<date>") {
		t.Error("term order not preserved")
	}
}

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestEncodeSchemaLocation(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, testTerms(), "../../../../schemas"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `xsi:noNamespaceSchemaLocation="../../../../schemas/dc.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", buf.String())
	}
}

func TestEncodeRefusesInvalid(t *testing.T) {
	bad := Terms{{Element: "abstract", Value: "x"}} // a qualified term, not Simple DC
	var buf bytes.Buffer
	if err := Encode(&buf, bad, "../../schemas"); err == nil {
		t.Fatal("Encode accepted an element outside Simple Dublin Core")
	}
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite refusing", buf.Len())
	}
}

func TestResolveKey(t *testing.T) {
	tests := []struct {
		key     string
		element string // "" means the key must be unknown
	}{
		{"identifier", "identifier"},
		{"Title", "title"}, // keys are case-insensitive
		{"coverage", "coverage"},
		{"abstract", ""},      // a qualified term meemoo's vocabulary has; not Simple DC
		{"dcterms:title", ""}, // prefixed keys are not supported
		{"titel", ""},         // typo
	}
	for _, tt := range tests {
		element, ok := ResolveKey(tt.key)
		if tt.element == "" {
			if ok {
				t.Errorf("ResolveKey(%q) resolved to %q, want unknown", tt.key, element)
			}
			continue
		}
		if !ok || element != tt.element {
			t.Errorf("ResolveKey(%q) = %q, %v; want %q", tt.key, element, ok, tt.element)
		}
	}
}

func TestTermValidate(t *testing.T) {
	tests := []struct {
		name string
		term Term
		want string // "" means valid; else substring of the error
	}{
		{"valid", Term{Element: "title", Value: "x"}, ""},
		{"valid with lang", Term{Element: "description", Lang: "nl-BE", Value: "x"}, ""},
		{"qualified term", Term{Element: "abstract", Value: "x"}, "not a Simple Dublin Core element"},
		{"prefixed", Term{Element: "dcterms:title", Value: "x"}, "not a Simple Dublin Core element"},
		{"bad lang", Term{Element: "title", Lang: "nl!", Value: "x"}, "not a language tag"},
		{"empty value", Term{Element: "subject", Value: "  "}, "empty value"},
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

// Simple DC has no cardinality rules of its own, so a repeated element is
// fine; only the identifier stays single.
func TestTermsValidate(t *testing.T) {
	repeated := Terms{
		{Element: "identifier", Value: "A"},
		{Element: "description", Value: "een"},
		{Element: "description", Value: "twee"},
	}
	if err := repeated.Validate(); err != nil {
		t.Fatalf("repeated description refused: %v", err)
	}

	twoIDs := Terms{
		{Element: "identifier", Value: "A"},
		{Element: "identifier", Value: "B"},
	}
	err := twoIDs.Validate()
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("want the exactly-one identifier rule, got %v", err)
	}
}

func TestValidateRequired(t *testing.T) {
	terms := Terms{{Element: "identifier", Value: "A"}}
	err := terms.ValidateRequired("identifier", "title")
	if err == nil || !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("want the missing title named, got %v", err)
	}
	if err := testTerms().ValidateRequired("identifier", "title"); err != nil {
		t.Fatalf("complete terms refused: %v", err)
	}
	// A key outside the fifteen can only be a mistake in a profile
	// definition and is reported, never silently satisfied.
	if err := terms.ValidateRequired("abstract"); err == nil || !strings.Contains(err.Error(), `"abstract"`) {
		t.Errorf("unknown required key not reported: %v", err)
	}
}

func TestLocalIdentifier(t *testing.T) {
	if got := testTerms().LocalIdentifier(); got != "uuid-x" {
		t.Errorf("LocalIdentifier = %q, want uuid-x", got)
	}
	if got := (Terms{{Element: "title", Value: "x"}}).LocalIdentifier(); got != "" {
		t.Errorf("LocalIdentifier without identifier = %q, want empty", got)
	}
}
