package dcschema

import (
	"strings"
	"testing"
)

func TestValidateCardinality(t *testing.T) {
	tests := []struct {
		name  string
		terms Terms
		want  string // "" means conformant; else substring of the error
	}{
		{"single-valued repeated", Terms{
			{Element: "dcterms:identifier", Value: "A"},
			{Element: "dcterms:identifier", Value: "B"},
		}, "exactly one"},
		{"per-language same language", Terms{
			{Element: "dcterms:abstract", Lang: "nl", Value: "een"},
			{Element: "dcterms:abstract", Lang: "nl", Value: "twee"},
		}, `language "nl"`},
		{"per-language distinct languages", Terms{
			{Element: "dcterms:title", Lang: "nl", Value: "Kat"},
			{Element: "dcterms:title", Lang: "en", Value: "Cat"},
		}, ""},
		{"per-language untagged repeat", Terms{
			{Element: "dcterms:abstract", Value: "een"},
			{Element: "dcterms:abstract", Value: "twee"},
		}, "distinct language tags"},
		{"repeatable repeated", Terms{
			{Element: "dcterms:subject", Lang: "nl", Value: "katten"},
			{Element: "dcterms:subject", Lang: "nl", Value: "testdata"},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.terms.ValidateCardinality()
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

// Every violation surfaces at once, not just the first.
func TestValidateCardinalityJoinsFindings(t *testing.T) {
	terms := Terms{
		{Element: "dcterms:created", Value: "1913"},
		{Element: "dcterms:created", Value: "1914"},
		{Element: "dcterms:rights", Lang: "nl", Value: "a"},
		{Element: "dcterms:rights", Lang: "nl", Value: "b"},
	}
	err := terms.ValidateCardinality()
	if err == nil {
		t.Fatal("want two findings, got none")
	}
	for _, want := range []string{"dcterms:created", "dcterms:rights"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not name %s: %v", want, err)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	terms := Terms{
		{Element: "dcterms:identifier", Value: "A"},
		{Element: "dcterms:title", Lang: "nl", Value: "Kat"},
	}
	if err := terms.ValidateRequired("identifier", "title"); err != nil {
		t.Fatalf("want conformant, got %v", err)
	}
	err := terms.ValidateRequired("identifier", "title", "description", "created")
	if err == nil {
		t.Fatal("want the missing keys reported, got none")
	}
	// A finding names the key and the element it emits.
	for _, want := range []string{"description (dcterms:description)", "created (dcterms:created)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("findings do not name %s: %v", want, err)
		}
	}
	// A key outside the table can only be a mistake in a profile definition
	// and is reported, never silently satisfied.
	if err := terms.ValidateRequired("titel"); err == nil || !strings.Contains(err.Error(), `"titel"`) {
		t.Errorf("unknown required key not reported: %v", err)
	}
}

func TestValidateRequiredLang(t *testing.T) {
	tests := []struct {
		name  string
		terms Terms
		lang  string
		want  string // "" means conformant; else substring of the error
	}{
		{"no rule", Terms{{Element: "dcterms:title", Lang: "fr", Value: "x"}}, "", ""},
		{"tagged without required language", Terms{
			{Element: "dcterms:title", Lang: "fr", Value: "Chat"},
		}, "nl", "dcterms:title"},
		{"required language among others", Terms{
			{Element: "dcterms:title", Lang: "fr", Value: "Chat"},
			{Element: "dcterms:title", Lang: "nl", Value: "Kat"},
		}, "nl", ""},
		{"untagged values carry no rule", Terms{
			{Element: "dcterms:creator", Value: "Edmond Sacré"},
		}, "nl", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.terms.ValidateRequiredLang(tt.lang)
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
