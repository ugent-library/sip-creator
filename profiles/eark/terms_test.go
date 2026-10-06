package eark

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

func testTerms() Terms {
	return Terms{
		{Key: "identifier", Value: "uuid-x"},
		{Key: "title", Lang: "nl", Value: "Fotoalbum 2026"},
		{Key: "date", Value: "1913"},
		{Key: "format", Value: "48 foto's"},
		{Key: "subject", Value: "R&D <scans>"},
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := (simpledc{}).Encode(&buf, testTerms(), "../../schemas"); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"<simpledc",
		`xsi:noNamespaceSchemaLocation="../../schemas/simpledc.xsd"`,
		"<identifier>uuid-x</identifier>",
		// language tags are accepted but not emitted
		"<title>Fotoalbum 2026</title>",
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
	if err := (simpledc{}).Encode(&buf, testTerms(), "../../../../schemas"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `xsi:noNamespaceSchemaLocation="../../../../schemas/simpledc.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", buf.String())
	}
}

func TestEncodeRefusesInvalid(t *testing.T) {
	bad := Terms{{Key: "abstract", Value: "x"}} // a qualified term, not Simple DC
	var buf bytes.Buffer
	if err := (simpledc{}).Encode(&buf, bad, "../../schemas"); err == nil {
		t.Fatal("Encode accepted a key outside Simple Dublin Core")
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
		{"valid", sip.Term{Key: "title", Value: "x"}, ""},
		{"valid with lang", sip.Term{Key: "description", Lang: "nl-BE", Value: "x"}, ""},
		{"valid coverage", sip.Term{Key: "coverage", Value: "x"}, ""},
		// a qualified term Meemoo's vocabulary has; not Simple DC
		{"qualified term", sip.Term{Key: "abstract", Value: "x"}, "unknown key"},
		{"prefixed", sip.Term{Key: "dcterms:title", Value: "x"}, "unknown key"},
		{"typo", sip.Term{Key: "titel", Value: "x"}, "unknown key"},
		// keys are lowercase; case folding is the rows file's convention
		{"capitalized", sip.Term{Key: "Title", Value: "x"}, "unknown key"},
		{"bad lang", sip.Term{Key: "title", Lang: "nl!", Value: "x"}, "not a language tag"},
		{"empty value", sip.Term{Key: "subject", Value: "  "}, "empty value"},
		// XML cannot carry it, so escaping would change the value
		{"control character", sip.Term{Key: "title", Value: "Tab\x0bvertical"}, "title: \"Tab\\vvertical\" holds the character U+000B"},
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

// Simple DC has no cardinality rules of its own, so a repeated element is
// fine; only the identifier stays single.
func TestTermsValidate(t *testing.T) {
	repeated := Terms{
		{Key: "identifier", Value: "A"},
		{Key: "description", Value: "een"},
		{Key: "description", Value: "twee"},
	}
	if err := repeated.Validate(); err != nil {
		t.Fatalf("repeated description refused: %v", err)
	}

	twoIDs := Terms{
		{Key: "identifier", Value: "A"},
		{Key: "identifier", Value: "B"},
	}
	err := twoIDs.Validate()
	if want := "identifier appears 2 times; give exactly one"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("want the identifier rule %q, got %v", want, err)
	}
}

// A package-level description states an identifier and a title; plain
// E-ARK requires nothing more, and the list names only real elements.
func TestValidateRequired(t *testing.T) {
	err := (Terms{{Key: "identifier", Value: "A"}}).ValidateRequired()
	if err == nil || !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("want the missing title named, got %v", err)
	}
	err = (Terms{{Key: "title", Value: "x"}}).ValidateRequired()
	if err == nil || !strings.Contains(err.Error(), "identifier is required") {
		t.Fatalf("want the missing identifier named, got %v", err)
	}
	if err := testTerms().ValidateRequired(); err != nil {
		t.Fatalf("complete terms refused: %v", err)
	}
	for _, key := range required {
		if !elementSet[key] {
			t.Errorf("required key %q is not a Simple Dublin Core element", key)
		}
	}
}

// Validate joins every finding, per-term ones included, so a producer sees
// all of them in one round rather than the first bad term alone. A
// per-term finding carries the term's position, so a caller who decoded
// the terms from rows can point at the row.
func TestTermsValidateReportsEveryTerm(t *testing.T) {
	terms := Terms{
		{Key: "identifier", Value: "A"},
		{Key: "abstract", Value: "x"},
		{Key: "subject", Value: " "},
	}
	err := terms.Validate()
	if err == nil {
		t.Fatal("want two findings, got none")
	}
	for _, want := range []string{"term 2:", "term 3:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	var positions []int
	for _, finding := range err.(interface{ Unwrap() []error }).Unwrap() {
		var te *sip.TermError
		if errors.As(finding, &te) {
			positions = append(positions, te.Index)
		}
	}
	if len(positions) != 2 || positions[0] != 1 || positions[1] != 2 {
		t.Errorf("per-term findings at positions %v, want [1 2]", positions)
	}
}
