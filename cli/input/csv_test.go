package input

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/metadata"
)

// readCSV runs Read over a minimal flat tree carrying the given
// metadata.csv, so the decoder is exercised through the real entry point.
func readCSV(t *testing.T, csv string) (*Package, error) {
	t.Helper()
	root := writeTree(t, map[string]string{
		"metadata.csv": csv,
		"scan.tiff":    "x",
	})
	return Read(root)
}

func TestMetadataCSVHappy(t *testing.T) {
	// BOM, CRLF, RFC 4180 quoting, repeated keys, [lang] tags, the renamed
	// plain-key mappings, and the schema.org keys — in one file.
	csv := "\ufeffkey,value\r\n" +
		"identifier,BIB.FA.2026.001\r\n" +
		"title[nl],Fotoalbum Gent 1913\r\n" +
		"description[nl],\"Album met 48 foto's, zwart-wit\"\r\n" +
		"subject[nl],stadsgezichten\r\n" +
		"subject[nl],wereldtentoonstellingen\r\n" +
		"ispartof,Collectie Sacré\r\n" +
		"rightsholder,Universiteitsbibliotheek Gent\r\n" +
		"abstract[en],A photo album\r\n" +
		"artmedium[nl],zilvergelatinedruk\r\n"

	pkg, err := readCSV(t, csv)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := metadata.Terms{
		{Element: "dcterms:identifier", Value: "BIB.FA.2026.001"},
		{Element: "dcterms:title", Lang: "nl", Value: "Fotoalbum Gent 1913"},
		{Element: "dcterms:description", Lang: "nl", Value: "Album met 48 foto's, zwart-wit"},
		{Element: "dcterms:subject", Lang: "nl", Value: "stadsgezichten"},
		{Element: "dcterms:subject", Lang: "nl", Value: "wereldtentoonstellingen"},
		{Element: "dcterms:isPartOf", Value: "Collectie Sacré"},
		{Element: "dcterms:rightsHolder", Value: "Universiteitsbibliotheek Gent"},
		{Element: "dcterms:abstract", Lang: "en", Value: "A photo album"},
		{Element: "schema:artMedium", Lang: "nl", Value: "zilvergelatinedruk"},
	}
	if len(pkg.Descriptive) != len(want) {
		t.Fatalf("got %d terms, want %d:\n%v", len(pkg.Descriptive), len(want), pkg.Descriptive)
	}
	for i, w := range want {
		if pkg.Descriptive[i] != w {
			t.Errorf("term %d = %+v, want %+v (order must be preserved)", i, pkg.Descriptive[i], w)
		}
	}
}

func TestMetadataCSVViolations(t *testing.T) {
	tests := []struct {
		name string
		csv  string
		want string // substring of the expected violation
	}{
		{"missing header", "identifier,ID-1\ntitle,T\n", `header "key,value"`},
		{"unknown key", minimalCSV + "titel,Oeps\n", `unknown key "titel"`},
		{"dcterms outside the profile", minimalCSV + "accrualpolicy,x\n", `unknown key "accrualpolicy"`},
		{"prefixed dcterms key", minimalCSV + "dcterms:abstract,x\n", "prefixed keys"},
		{"prefixed schema key", minimalCSV + "schema:artMedium,x\n", "prefixed keys"},
		{"unknown prefix", minimalCSV + "foo:bar,x\n", "prefixed keys"},
		{"empty value", minimalCSV + "subject,\n", "empty value"},
		{"missing identifier", "key,value\ntitle,T\n", "identifier is missing"},
		{"missing title", "key,value\nidentifier,ID-1\n", "title is missing"},
		{"duplicate identifier", minimalCSV + "identifier,ID-2\n", "exactly one"},
		{"single-valued key repeated", minimalCSV + "created,1913\ncreated,1914\n", "exactly one"},
		{"per-language key repeated in one language", minimalCSV + "abstract[nl],a\nabstract[nl],b\n", `language "nl"`},
		{"per-language key repeated untagged", minimalCSV + "abstract,a\nabstract,b\n", "distinct language tags"},
		{"empty lang tag", minimalCSV + "subject[],x\n", "malformed language tag"},
		{"bad lang tag", minimalCSV + "subject[nl!],x\n", "not a language tag"},
		{"three columns", minimalCSV + "subject,a,b\n", "exactly two columns"},
		{"not utf-8", "key,value\nidentifier,ID\ntitle,\xff\xfe\n", "not valid UTF-8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readCSV(t, tt.csv)
			assertViolation(t, err, tt.want)
		})
	}
}

func TestMetadataCSVMissingHeaderStillDecodes(t *testing.T) {
	// Collect-all: the header violation must not hide findings in the rows.
	_, err := readCSV(t, "identifier,ID-1\ntitel,Oeps\n")
	assertViolation(t, err, `header "key,value"`)
	assertViolation(t, err, `unknown key "titel"`)
}

func TestMetadataCSVLineNumbers(t *testing.T) {
	_, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle,T\ntitel,Oeps\n")
	assertViolation(t, err, "line 4")
}

// A cardinality violation is a cross-row finding: no line number, but the
// element and language it names locate the rows in a keyed file.
func TestMetadataCSVRepeatNamesElementAndLanguage(t *testing.T) {
	_, err := readCSV(t, minimalCSV+"abstract[nl],a\nabstract[nl],b\n")
	assertViolation(t, err, `dcterms:abstract appears more than once in language "nl"`)
}

// Per-language keys repeat freely across languages (title[nl] + title[en]);
// only a same-language repeat is a violation.
func TestMetadataCSVPerLanguageRepeat(t *testing.T) {
	if _, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle[nl],Kat\ntitle[en],Cat\n"); err != nil {
		t.Fatalf("distinct languages must be accepted: %v", err)
	}
	_, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle[nl],Kat\ntitle[nl],Poes\n")
	assertViolation(t, err, `language "nl"`)
}

func TestRepresentationCSVNeedsNoIdentity(t *testing.T) {
	root := writeTree(t, map[string]string{
		"metadata.csv":                        minimalCSV,
		"representations/master/scan.tiff":    "x",
		"representations/master/metadata.csv": "key,value\nlicense,publiek domein\n",
	})
	pkg, err := Read(root)
	if err != nil {
		t.Fatalf("rep-level metadata.csv must not require identifier/title: %v", err)
	}
	got := pkg.Representations[0].Descriptive
	if len(got) != 1 || got[0].Element != "dcterms:license" {
		t.Errorf("rep descriptive = %v", got)
	}
}

func TestRepresentationCSVDuplicateIdentifier(t *testing.T) {
	// Identity is optional at rep level, but two identifiers stay ambiguous.
	root := writeTree(t, map[string]string{
		"metadata.csv":                        minimalCSV,
		"representations/master/scan.tiff":    "x",
		"representations/master/metadata.csv": "key,value\nidentifier,A\nidentifier,B\n",
	})
	_, err := Read(root)
	assertViolation(t, err, "exactly one")
}

func TestMetadataCSVQuotedNewline(t *testing.T) {
	// A quoted value may span lines (RFC 4180); line numbers must survive.
	csv := "key,value\nidentifier,ID-1\ndescription,\"two\nlines\"\ntitel,Oeps\n"
	_, err := readCSV(t, csv)
	assertViolation(t, err, "line 5")
	if !strings.Contains(err.Error(), `unknown key "titel"`) {
		t.Errorf("multiline value swallowed the following row: %v", err)
	}
}
