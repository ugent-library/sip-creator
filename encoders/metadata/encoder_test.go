package metadata

import (
	"bytes"
	"strings"
	"testing"
)

const testSource = `{
	"dcterms:identifier": "local-id-001",
	"dcterms:title": {"@value": "Catus Testus", "@language": "nl"},
	"dcterms:creator": ["Matthias Vandermaesen"],
	"dcterms:created": "2020"
}`

// EncodeDC must produce the dc_SimpleDC20021212 shape RODA renders
// natively: a simpledc root, unqualified elements, no meemoo namespaces.
func TestEncodeDC(t *testing.T) {
	d, err := Decode(strings.NewReader(testSource))
	if err != nil {
		t.Fatal(err)
	}
	d.SetObjectIdentifier("uuid-test-entity")

	var buf bytes.Buffer
	if err := EncodeDC(&buf, d); err != nil {
		t.Fatalf("EncodeDC: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"<simpledc",
		"<identifier>uuid-test-entity</identifier>",
		"<identifier>local-id-001</identifier>", // the source's own identifier survives
		"<title>Catus Testus</title>",
		"<creator>Matthias Vandermaesen</creator>",
		"<date>2020</date>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("EncodeDC output missing %s\n%s", want, out)
		}
	}
	if strings.Contains(out, "dcterms:") || strings.Contains(out, "hetarchief") {
		t.Errorf("EncodeDC output carries meemoo vocabulary:\n%s", out)
	}
}
