package input

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/eark"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// readCSV runs Read over a minimal flat tree carrying the given
// description.csv under the basic profile, so the decoder is exercised
// through the real entry point.
func readCSV(t *testing.T, csv string) (*build.SourcePackage, error) {
	t.Helper()
	root := writeTree(t, map[string]string{
		"description.csv": csv,
		"scan.tiff":       "x",
	})
	return Read(root, meemooVocab{})
}

func TestRowsHappy(t *testing.T) {
	// BOM, CRLF, RFC 4180 quoting, repeated keys, [lang] tags, a
	// capitalized key, and a schema.org key, all in one file.
	csv := "\ufeffkey,value\r\n" +
		"identifier,example-0001\r\n" +
		"Title[nl],Fotoalbum 2026\r\n" +
		"description[nl],\"Album met 48 foto's, zwart-wit\"\r\n" +
		"created,1913\r\n" +
		"subject[nl],voorbeelden\r\n" +
		"subject[nl],fotografie\r\n" +
		"ispartof,Collectie Één\r\n" +
		"rightsholder,Example Organization\r\n" +
		"abstract[nl],Een fotoalbum\r\n" +
		"abstract[en],A photo album\r\n" +
		"artmedium[nl],zilvergelatinedruk\r\n"

	pkg, err := readCSV(t, csv)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	want := meemoo.Terms{
		{Key: "identifier", Value: "example-0001"},
		{Key: "title", Lang: "nl", Value: "Fotoalbum 2026"},
		{Key: "description", Lang: "nl", Value: "Album met 48 foto's, zwart-wit"},
		{Key: "created", Value: "1913"},
		{Key: "subject", Lang: "nl", Value: "voorbeelden"},
		{Key: "subject", Lang: "nl", Value: "fotografie"},
		{Key: "ispartof", Value: "Collectie Één"},
		{Key: "rightsholder", Value: "Example Organization"},
		{Key: "abstract", Lang: "nl", Value: "Een fotoalbum"},
		{Key: "abstract", Lang: "en", Value: "A photo album"},
		{Key: "artmedium", Lang: "nl", Value: "zilvergelatinedruk"},
	}
	got, ok := pkg.Description.(meemoo.Terms)
	if !ok || len(got) != len(want) {
		t.Fatalf("got %T with %d terms, want %d Meemoo terms:\n%v", pkg.Description, len(got), len(want), pkg.Description)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("term %d = %+v, want %+v (order must be preserved)", i, got[i], w)
		}
	}
}

// Under the basic profile the rows are Meemoo's: a language-tagged key
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
		{"unknown key", minimalCSV + "titel,Oeps\n", `unknown key "titel"`},
		{"dcterms outside the profile", minimalCSV + "accrualpolicy,x\n", `unknown key "accrualpolicy"`},
		{"empty value", minimalCSV + "subject,\n", "empty value"},
		{"missing identifier", "key,value\ntitle,T\n", "identifier is required"},
		{"missing title", "key,value\nidentifier,ID-1\n", "title is required"},
		{"duplicate identifier", minimalCSV + "identifier,ID-2\n", "exactly one"},
		{"single-valued key repeated", minimalCSV + "created,1913\ncreated,1914\n", "exactly one"},
		{"per-language key repeated in one language", minimalCSV + "abstract[nl],a\nabstract[nl],b\n", `language "nl"`},
		{"per-language key repeated untagged", minimalCSV + "abstract,a\nabstract,b\n", "distinct language tags"},
		{"bad lang tag", minimalCSV + "subject[nl!],x\n", "not a language tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readCSV(t, tt.csv)
			assertViolation(t, err, tt.want)
		})
	}
}

// A finding about one row is reported with the file and the row's line,
// whether the parser, the vocabulary or the description's rules found it.
func TestRowsLineNumbers(t *testing.T) {
	_, err := readCSV(t, "key,value\nidentifier,ID-1\ntitle,T\ntitel,Oeps\n")
	assertViolation(t, err, `description.csv line 4: unknown key "titel"`)
	_, err = readCSV(t, minimalCSV+"subject[nl!],x\nsubject,\nsubject[],x\n")
	assertViolation(t, err, `description.csv line 6: "nl!" is not a language tag`)
	assertViolation(t, err, "description.csv line 7: subject has an empty value")
	assertViolation(t, err, `description.csv line 8: malformed language tag in "subject[]"`)
}

// Content that cannot be read as CSV is one violation: the rows are not
// decoded, so the description's rules do not add findings about keys the
// file may well contain.
func TestRowsNotUTF8IsOneViolation(t *testing.T) {
	_, err := readCSV(t, "key,value\nidentifier,ID\ntitle,\xff\xfe\n")
	v, ok := errors.AsType[Violations](err)
	if !ok || len(v) != 1 || !strings.Contains(v[0], "description.csv: not valid UTF-8") {
		t.Fatalf("want one UTF-8 violation, got %v", err)
	}
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
	pkg, err := Read(root, meemooVocab{})
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
	_, err := Read(root, meemooVocab{})
	assertViolation(t, err, "exactly one")
}

// BOM, CRLF, RFC 4180 quoting (a value spanning two lines included), a
// capitalized key and a language tag, all in one file.
func TestParseStatements(t *testing.T) {
	data := "\ufeffKey,Value\r\n" +
		"Identifier,ID-1\r\n" +
		"description[nl],\"two\r\nlines, quoted\"\r\n" +
		"title,T\r\n"

	statements, errs, err := parseStatements([]byte(data))
	if err != nil || len(errs) > 0 {
		t.Fatalf("parseStatements: err %v, findings %v", err, errs)
	}
	want := []Statement{
		{Key: "identifier", Value: "ID-1", Line: 2},
		{Key: "description", Lang: "nl", Value: "two\nlines, quoted", Line: 3},
		{Key: "title", Value: "T", Line: 5},
	}
	if len(statements) != len(want) {
		t.Fatalf("got %d statements, want %d: %+v", len(statements), len(want), statements)
	}
	for i, w := range want {
		if statements[i] != w {
			t.Errorf("statement %d = %#v, want %#v", i, statements[i], w)
		}
	}
}

func TestParseStatementsFindings(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		wantLine int    // line of the *StatementError; 0 for a finding about the file
		want     string // substring of the finding
	}{
		{"missing header", "identifier,ID-1\n", 0, `header "key,value"`},
		{"three columns", "key,value\nsubject,a,b\n", 2, "exactly two columns"},
		{"one column", "key,value\nsubject\n", 2, "exactly two columns"},
		{"empty lang tag", "key,value\nsubject[],x\n", 2, "malformed language tag"},
		{"unclosed lang tag", "key,value\nsubject[nl,x\n", 2, "malformed language tag"},
		{"prefixed dcterms key", "key,value\ndcterms:abstract,x\n", 2, "prefixed keys"},
		{"prefixed schema key", "key,value\nschema:artMedium,x\n", 2, "prefixed keys"},
		{"broken quote", "key,value\ntitle,\"open\n", 0, "extraneous or missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs, err := parseStatements([]byte(tt.data))
			if err != nil {
				t.Fatalf("parseStatements: %v", err)
			}
			assertFinding(t, errs, tt.want)
			if tt.wantLine == 0 {
				return
			}
			se, ok := errors.AsType[*StatementError](errs[0])
			if !ok || se.Line != tt.wantLine {
				t.Errorf("finding %v, want a *StatementError at line %d", errs[0], tt.wantLine)
			}
		})
	}
}

// A first row that is data, not the header, is still parsed, so its own
// findings are reported next to the missing header.
func TestParseStatementsMissingHeaderKeepsTheRow(t *testing.T) {
	statements, errs, _ := parseStatements([]byte("identifier,ID-1\n"))
	assertFinding(t, errs, `header "key,value"`)
	if len(statements) != 1 || statements[0].Key != "identifier" || statements[0].Line != 1 {
		t.Errorf("statements = %+v, want the first row kept", statements)
	}
}

func TestParseStatementsNotUTF8(t *testing.T) {
	_, _, err := parseStatements([]byte("key,value\ntitle,\xff\xfe\n"))
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("want the content refused as not UTF-8, got %v", err)
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
	pkg, err := Read(writeTree(t, tree), earkVocab{})
	if err != nil {
		t.Fatalf("Read under eark: %v", err)
	}
	got, ok := pkg.Description.(eark.Terms)
	if !ok || len(got) != 3 || got[0].Key != "identifier" || got[2].Key != "coverage" {
		t.Errorf("descriptive = %#v, want three Simple DC terms", pkg.Description)
	}

	_, err = Read(writeTree(t, tree), meemooVocab{})
	assertViolation(t, err, `unknown key "coverage"`)
}

// Under eark only the fifteen Simple DC elements are keys: Meemoo's keys
// are unknown there, at both levels.
func TestRowsEarkRefusesMeemooKeys(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalDC + "abstract,x\n",
		"representations/master/scan.tiff":       "x",
		"representations/master/description.csv": "key,value\nlicense,publiek domein\n",
	})
	_, err := Read(root, earkVocab{})
	assertViolation(t, err, `unknown key "abstract"`)
	assertViolation(t, err, `unknown key "license"`)
}

// Without a vocabulary Read cannot say what the rows mean; it is
// refused before the folder is touched.
func TestReadRequiresAVocabulary(t *testing.T) {
	root := writeTree(t, map[string]string{"description.csv": minimalCSV, "scan.tiff": "x"})
	_, err := Read(root, nil)
	if err == nil || !strings.Contains(err.Error(), "no vocabulary") {
		t.Fatalf("want the missing vocabulary refused, got %v", err)
	}
}

// A vocabulary's own errors are reported at the row's line when they are
// about one statement, or against the file otherwise, next to the
// description's rules.
func TestVocabularyErrorsNameTheLine(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv": "key,value\nidentifier,ID-1\ntitle,T\n",
		"scan.tiff":       "x",
	})
	_, err := Read(root, placesNothing{})
	assertViolation(t, err, "description.csv line 3: no place for title")
	assertViolation(t, err, "description.csv: nothing fits")
	assertViolation(t, err, "identifier is required")
}

// placesNothing is a vocabulary that refuses every statement at its line,
// adds one error about the file, and returns an empty eark description, so
// the description's own required-keys rule still runs on the result.
type placesNothing struct{}

func (placesNothing) Description(statements []Statement) (sip.Description, []error) {
	errs := []error{errors.New("nothing fits")}
	for _, s := range statements {
		errs = append(errs, &StatementError{Line: s.Line, Err: fmt.Errorf("no place for %s", s.Key)})
	}
	return eark.Terms(nil), errs
}
