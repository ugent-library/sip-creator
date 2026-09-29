package input

import (
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
)

// readCSV runs Read over a minimal flat tree carrying the given
// description.csv under the basic profile, so the decoder is exercised
// through the real entry point.
func readCSV(t *testing.T, csv string) (*build.Material, error) {
	t.Helper()
	root := writeTree(t, map[string]string{
		"description.csv": csv,
		"scan.tiff":       "x",
	})
	return basicReader.Read(root)
}

func TestRowsHappy(t *testing.T) {
	// BOM, CRLF, RFC 4180 quoting, repeated keys, [lang] tags, a
	// capitalized key, and the schema.org keys, all in one file.
	csv := "\ufeffkey,value\r\n" +
		"identifier,BIB.FA.2026.001\r\n" +
		"Title[nl],Fotoalbum Gent 1913\r\n" +
		"description[nl],\"Album met 48 foto's, zwart-wit\"\r\n" +
		"created,1913\r\n" +
		"subject[nl],stadsgezichten\r\n" +
		"subject[nl],wereldtentoonstellingen\r\n" +
		"ispartof,Collectie Sacré\r\n" +
		"rightsholder,Universiteitsbibliotheek Gent\r\n" +
		"abstract[nl],Een fotoalbum\r\n" +
		"abstract[en],A photo album\r\n" +
		"artmedium[nl],zilvergelatinedruk\r\n"

	pkg, err := readCSV(t, csv)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := meemoo.Terms{
		{Key: "identifier", Value: "BIB.FA.2026.001"},
		{Key: "title", Lang: "nl", Value: "Fotoalbum Gent 1913"},
		{Key: "description", Lang: "nl", Value: "Album met 48 foto's, zwart-wit"},
		{Key: "created", Value: "1913"},
		{Key: "subject", Lang: "nl", Value: "stadsgezichten"},
		{Key: "subject", Lang: "nl", Value: "wereldtentoonstellingen"},
		{Key: "ispartof", Value: "Collectie Sacré"},
		{Key: "rightsholder", Value: "Universiteitsbibliotheek Gent"},
		{Key: "abstract", Lang: "nl", Value: "Een fotoalbum"},
		{Key: "abstract", Lang: "en", Value: "A photo album"},
		{Key: "artmedium", Lang: "nl", Value: "zilvergelatinedruk"},
	}
	got, ok := pkg.Description.(meemoo.Terms)
	if !ok || len(got) != len(want) {
		t.Fatalf("got %T with %d terms, want %d meemoo terms:\n%v", pkg.Description, len(got), len(want), pkg.Description)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("term %d = %+v, want %+v (order must be preserved)", i, got[i], w)
		}
	}
}

// Under the basic profile the rows are meemoo's: a language-tagged key
// without a Dutch entry is a violation at check time, not only at build.
func TestRowsCSVRequiresDutch(t *testing.T) {
	_, err := readCSV(t, minimalCSV+"abstract[en],A photo album\n")
	assertViolation(t, err, `none in "nl"`)
}

func TestRowsViolations(t *testing.T) {
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
		{"missing identifier", "key,value\ntitle,T\n", "identifier is required"},
		{"missing title", "key,value\nidentifier,ID-1\n", "title is required"},
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

func TestRowsMissingHeaderStillDecodes(t *testing.T) {
	// Collect-all: the header violation must not hide findings in the rows.
	_, err := readCSV(t, "identifier,ID-1\ntitel,Oeps\n")
	assertViolation(t, err, `header "key,value"`)
	assertViolation(t, err, `unknown key "titel"`)
}

// A finding about one term is reported at the row's line: the vocabulary
// names the term by position, the decoder maps that back to the line.
func TestRowsLineNumbers(t *testing.T) {
	_, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle,T\ntitel,Oeps\n")
	assertViolation(t, err, `line 4: unknown key "titel"`)
	_, err = readCSV(t, minimalCSV+"subject[nl!],x\nsubject,\n")
	assertViolation(t, err, `line 6: "nl!" is not a language tag`)
	assertViolation(t, err, "line 7: subject has an empty value")
}

// A cardinality violation is a cross-row finding: no line number, but the
// key and language it names locate the rows in a keyed file.
func TestRowsRepeatNamesKeyAndLanguage(t *testing.T) {
	_, err := readCSV(t, minimalCSV+"abstract[nl],a\nabstract[nl],b\n")
	assertViolation(t, err, `abstract appears more than once in language "nl"`)
}

// Per-language keys repeat freely across languages (title[nl] + title[en]);
// only a same-language repeat is a violation.
func TestRowsPerLanguageRepeat(t *testing.T) {
	if _, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle[nl],Kat\ntitle[en],Cat\ndescription,x\ncreated,2026\n"); err != nil {
		t.Fatalf("distinct languages must be accepted: %v", err)
	}
	_, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle[nl],Kat\ntitle[nl],Poes\n")
	assertViolation(t, err, `language "nl"`)
}

func TestRepresentationCSVNeedsNoIdentity(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalCSV,
		"representations/master/scan.tiff":       "x",
		"representations/master/description.csv": "key,value\nlicense,publiek domein\n",
	})
	pkg, err := basicReader.Read(root)
	if err != nil {
		t.Fatalf("rep-level description.csv must not require identifier/title: %v", err)
	}
	got, ok := pkg.Representations[0].Description.(meemoo.Terms)
	if !ok || len(got) != 1 || got[0].Key != "license" {
		t.Errorf("rep descriptive = %#v", pkg.Representations[0].Description)
	}
}

func TestRepresentationCSVDuplicateIdentifier(t *testing.T) {
	// Identity is optional at rep level, but two identifiers stay ambiguous.
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalCSV,
		"representations/master/scan.tiff":       "x",
		"representations/master/description.csv": "key,value\nidentifier,A\nidentifier,B\n",
	})
	_, err := basicReader.Read(root)
	assertViolation(t, err, "exactly one")
}

func TestRowsQuotedNewline(t *testing.T) {
	// A quoted value may span lines (RFC 4180); line numbers must survive.
	csv := "key,value\nidentifier,ID-1\ndescription,\"two\nlines\"\ntitel,Oeps\n"
	_, err := readCSV(t, csv)
	assertViolation(t, err, "line 5")
	if !strings.Contains(err.Error(), `unknown key "titel"`) {
		t.Errorf("multiline value swallowed the following row: %v", err)
	}
}

// The profile, not the file, says which vocabulary the rows are in: the
// same description.csv is Simple Dublin Core under eark and refused under
// basic, where coverage is not a key.
func TestRowsProfileDecidesTheVocabulary(t *testing.T) {
	tree := map[string]string{
		"description.csv": minimalDC + "coverage,Gent\n",
		"scan.tiff":       "x",
	}
	pkg, err := earkReader.Read(writeTree(t, tree))
	if err != nil {
		t.Fatalf("Read under eark: %v", err)
	}
	got, ok := pkg.Description.(eark.Terms)
	if !ok || len(got) != 3 || got[0].Key != "identifier" || got[2].Key != "coverage" {
		t.Errorf("descriptive = %#v, want three Simple DC terms", pkg.Description)
	}

	_, err = basicReader.Read(writeTree(t, tree))
	assertViolation(t, err, `unknown key "coverage"`)
}

// Under eark only the fifteen Simple DC elements are keys: meemoo's keys
// are unknown there, at both levels.
func TestRowsEarkRefusesMeemooKeys(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalDC + "abstract,x\n",
		"representations/master/scan.tiff":       "x",
		"representations/master/description.csv": "key,value\nlicense,publiek domein\n",
	})
	_, err := earkReader.Read(root)
	assertViolation(t, err, `unknown key "abstract"`)
	assertViolation(t, err, `unknown key "license"`)
}

// Without a description builder the reader cannot say what the rows mean;
// it is refused before the folder is touched.
func TestReadRequiresABuilder(t *testing.T) {
	root := writeTree(t, map[string]string{"description.csv": minimalCSV, "scan.tiff": "x"})
	_, err := New(nil).Read(root)
	if err == nil || !strings.Contains(err.Error(), "no description builder") {
		t.Fatalf("want the missing builder refused, got %v", err)
	}
}
