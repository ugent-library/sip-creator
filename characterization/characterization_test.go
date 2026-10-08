package characterization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decodeFixture(t *testing.T, name string) Report {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	report, err := DecodeSiegfried(f)
	if err != nil {
		t.Fatalf("DecodeSiegfried(%s): %v", name, err)
	}
	return report
}

func TestDecodeSiegfriedMatch(t *testing.T) {
	report := decodeFixture(t, "match.json")

	rec, ok := report["cat.jpg"]
	if !ok {
		t.Fatalf("report has no entry for cat.jpg, keys: %v", keys(report))
	}
	if rec.Format == nil || rec.Format.FormatRegistry == nil {
		t.Fatalf("record Format = %+v, want a format with a registry", rec.Format)
	}
	if rec.Format.FormatRegistry.Name != "pronom" {
		t.Errorf("registry Name = %q, want %q", rec.Format.FormatRegistry.Name, "pronom")
	}
	if rec.Format.FormatRegistry.Key != "fmt/44" {
		t.Errorf("registry Key = %q, want %q", rec.Format.FormatRegistry.Key, "fmt/44")
	}
	if rec.Mime != "image/jpeg" {
		t.Errorf("Mime = %q, want %q", rec.Mime, "image/jpeg")
	}
	if rec.MD5 != "ec26c87385203b67e8ded8693ced2505" {
		t.Errorf("MD5 = %q, want the report's digest", rec.MD5)
	}
	if rec.Errors != "" {
		t.Errorf("Errors = %q, want empty", rec.Errors)
	}
}

func TestDecodeSiegfriedNoMatch(t *testing.T) {
	report := decodeFixture(t, "nomatch.json")

	rec, ok := report["mystery.bin"]
	if !ok {
		t.Fatalf("report has no entry for mystery.bin, keys: %v", keys(report))
	}
	if rec.Format != nil {
		t.Errorf("record Format = %+v on no match, want nil", rec.Format)
	}
	if rec.Mime != "" {
		t.Errorf("Mime = %q on no match, want empty", rec.Mime)
	}
	if rec.MD5 == "" {
		t.Error("MD5 is empty, want the report's digest carried")
	}
}

// The tool's error for a file is kept on its record. DecodeSiegfried does
// not reject the report for it.
func TestDecodeSiegfriedErrorsCarried(t *testing.T) {
	report := decodeFixture(t, "error.json")

	rec, ok := report["gone.jpg"]
	if !ok {
		t.Fatalf("report has no entry for gone.jpg, keys: %v", keys(report))
	}
	if rec.Errors == "" {
		t.Error("Errors is empty, want the tool's error carried verbatim")
	}
}

func TestDecodeSiegfriedMalformed(t *testing.T) {
	if _, err := DecodeSiegfried(strings.NewReader("{ not json")); err == nil {
		t.Fatal("DecodeSiegfried accepted malformed JSON")
	}
}

// DecodeSiegfried rejects a JSON object without a top-level siegfried
// version, instead of returning an empty report.
func TestDecodeSiegfriedWrongShape(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "wrongshape.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := DecodeSiegfried(f); err == nil {
		t.Fatal("DecodeSiegfried accepted JSON that is not a siegfried report")
	}
}

// Report keys are relative and cleaned, with slash separators, also for a
// path that sf recorded with backslashes on Windows.
func TestDecodeSiegfriedKeyNormalization(t *testing.T) {
	report, err := DecodeSiegfried(strings.NewReader(`{
		"siegfried": "1.11.0",
		"files": [
			{"filename": "./representation_1//cat.jpg", "filesize": 1, "errors": "", "md5": "ab", "matches": []},
			{"filename": ".\\representations\\master\\dog.jpg", "filesize": 1, "errors": "", "md5": "cd", "matches": []}
		]
	}`))
	if err != nil {
		t.Fatalf("DecodeSiegfried: %v", err)
	}

	for _, key := range []string{"representation_1/cat.jpg", "representations/master/dog.jpg"} {
		if _, ok := report[key]; !ok {
			t.Errorf("normalized key %q missing, keys: %v", key, keys(report))
		}
	}
}

func keys(r Report) []string {
	ks := make([]string, 0, len(r))
	for k := range r {
		ks = append(ks, k)
	}
	return ks
}
