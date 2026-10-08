package meemoo

import (
	"errors"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

func TestValidateCardinality(t *testing.T) {
	tests := []struct {
		name  string
		terms Terms
		want  string // empty when conformant, otherwise a substring of the error
	}{
		{"single-valued repeated", Terms{
			{Key: "dcterms:identifier", Value: "A"},
			{Key: "dcterms:identifier", Value: "B"},
		}, "exactly one"},
		{"per-language same language", Terms{
			{Key: "dcterms:abstract", Lang: "nl", Value: "een"},
			{Key: "dcterms:abstract", Lang: "nl", Value: "twee"},
		}, `language "nl"`},
		{"per-language distinct languages", Terms{
			{Key: "dcterms:title", Lang: "nl", Value: "Kat"},
			{Key: "dcterms:title", Lang: "en", Value: "Cat"},
		}, ""},
		{"per-language untagged repeat", Terms{
			{Key: "dcterms:abstract", Value: "een"},
			{Key: "dcterms:abstract", Value: "twee"},
		}, "a distinct language on each value"},
		{"repeatable repeated", Terms{
			{Key: "dcterms:subject", Lang: "nl", Value: "katten"},
			{Key: "dcterms:subject", Lang: "nl", Value: "testdata"},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.terms.validateCardinality()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want conformant, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}

// Every violation surfaces at once, not just the first, each naming its
// element.
func TestValidateCardinalityJoinsFindings(t *testing.T) {
	terms := Terms{
		{Key: "dcterms:created", Value: "1913"},
		{Key: "dcterms:created", Value: "1914"},
		{Key: "dcterms:rights", Lang: "nl", Value: "a"},
		{Key: "dcterms:rights", Lang: "nl", Value: "b"},
	}
	err := terms.validateCardinality()
	if err == nil {
		t.Fatal("want two findings, got none")
	}
	for _, want := range []string{"created appears", "rights appears"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
}

// Validate joins every finding, per-term ones included, so a producer sees
// all of them in one round rather than the first bad term alone. A
// per-term finding carries the term's position, so a caller who decoded
// the terms from rows can point at the row.
func TestTermsValidateReportsEveryTerm(t *testing.T) {
	terms := Terms{
		{Key: "dcterms:identifier", Value: "A"},
		{Key: "titel", Value: "x"},
		{Key: "dcterms:subject", Value: " "},
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

// A package-level description states Meemoo's four required elements.
// Each missing one is a finding naming the element.
func TestValidateRequired(t *testing.T) {
	err := (Terms{
		{Key: "dcterms:identifier", Value: "A"},
		{Key: "dcterms:title", Lang: "nl", Value: "Kat"},
	}).ValidateRequired()
	if err == nil {
		t.Fatal("want the missing keys reported, got none")
	}
	for _, want := range []string{"description is required", "created is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not include %q: %v", want, err)
		}
	}
	err = (Terms{{Key: "dcterms:title", Lang: "nl", Value: "Kat"}}).ValidateRequired()
	if err == nil || !strings.Contains(err.Error(), "identifier is required") {
		t.Errorf("missing identifier not reported: %v", err)
	}
	complete := append(testTerms(), sip.Term{Key: "dcterms:description", Lang: "nl", Value: "Een album"})
	if err := complete.ValidateRequired(); err != nil {
		t.Fatalf("complete terms refused: %v", err)
	}
}

// Every required element is one the table lists, so a typo in the list
// fails here rather than at the first build.
func TestRequiredElementsAreInTheTable(t *testing.T) {
	for _, element := range required {
		if _, ok := elementsByName[element]; !ok {
			t.Errorf("required element %q is not in the table", element)
		}
	}
}

func TestValidateRequiredLang(t *testing.T) {
	tests := []struct {
		name  string
		terms Terms
		lang  string
		want  string // empty when conformant, otherwise a substring of the error
	}{
		{"no rule", Terms{{Key: "dcterms:title", Lang: "fr", Value: "x"}}, "", ""},
		{"tagged without required language", Terms{
			{Key: "dcterms:title", Lang: "fr", Value: "Chat"},
		}, "nl", "title carries"},
		{"required language among others", Terms{
			{Key: "dcterms:title", Lang: "fr", Value: "Chat"},
			{Key: "dcterms:title", Lang: "nl", Value: "Kat"},
		}, "nl", ""},
		{"untagged values carry no rule", Terms{
			{Key: "dcterms:creator", Value: "Jane Doe"},
		}, "nl", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.terms.validateRequiredLang(tt.lang)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want conformant, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}
