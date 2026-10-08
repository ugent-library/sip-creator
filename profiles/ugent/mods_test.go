package ugent

import (
	"bytes"
	"strings"
	"testing"
)

// encodeRecord returns r as a MODS document. It fails the test if Encode
// returns an error.
func encodeRecord(t *testing.T, r Record, schemasDir string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := (mods{}).Encode(&buf, r, schemasDir); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// goldenDocument is the document for the test record, byte for byte. It
// holds the bytes the template wrote before Record was typed by field
// (2026-09-30), so the test shows that change left the output unchanged.
const goldenDocument = `<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="3.7" xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../schemas/mods-3-7.xsd">
  <mods:identifier type="local">example-0001</mods:identifier>
  <mods:titleInfo xml:lang="nl">
    <mods:title>Fotoalbum 2026</mods:title>
  </mods:titleInfo>
  <mods:titleInfo xml:lang="en">
    <mods:title>Photo album 2026</mods:title>
  </mods:titleInfo>
  <mods:location>
    <mods:holdingSimple>
      <mods:copyInformation>
        <mods:shelfLocator>EX.0001</mods:shelfLocator>
        <mods:enumerationAndChronology>vol. 3 (1913)</mods:enumerationAndChronology>
        <mods:itemIdentifier type="barcode">000000123</mods:itemIdentifier>
      </mods:copyInformation>
      <mods:copyInformation>
        <mods:shelfLocator>EX.0002</mods:shelfLocator>
      </mods:copyInformation>
    </mods:holdingSimple>
  </mods:location>
</mods:mods>
`

func TestRecordEncodeGolden(t *testing.T) {
	if got := encodeRecord(t, testRecord(), "../../schemas"); got != goldenDocument {
		t.Errorf("document differs from the golden bytes:\n--- got\n%s--- want\n%s", got, goldenDocument)
	}
}

func TestRecordEncode(t *testing.T) {
	r := testRecord()
	// An untagged title with producer text that must be escaped.
	r.Titles = append(r.Titles, Title{Value: "R&D <scans> 'quote'"})
	out := encodeRecord(t, r, "../../schemas")

	for _, want := range []string{
		`<mods:mods xmlns:mods="http://www.loc.gov/mods/v3"`,
		`version="3.7"`,
		`xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../schemas/mods-3-7.xsd"`,
		`<mods:identifier type="` + localIdentifierType + `">example-0001</mods:identifier>`,
		"<mods:titleInfo xml:lang=\"nl\">\n    <mods:title>Fotoalbum 2026</mods:title>\n  </mods:titleInfo>",
		`<mods:titleInfo xml:lang="en">`,
		// no language tag, so no xml:lang, and producer values escaped
		"<mods:titleInfo>\n    <mods:title>R&amp;D &lt;scans&gt; &#39;quote&#39;</mods:title>",
		// the items as one location, one copyInformation per item
		"<mods:location>\n    <mods:holdingSimple>",
		`<mods:shelfLocator>EX.0001</mods:shelfLocator>`,
		`<mods:enumerationAndChronology>vol. 3 (1913)</mods:enumerationAndChronology>`,
		`<mods:itemIdentifier type="barcode">000000123</mods:itemIdentifier>`,
		`<mods:shelfLocator>EX.0002</mods:shelfLocator>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %s\n%s", want, out)
		}
	}
	for element, want := range map[string]int{
		"<mods:identifier":       1,
		"<mods:titleInfo":        3,
		"<mods:location>":        1,
		"<mods:holdingSimple>":   1,
		"<mods:copyInformation>": 2,
		// the second copy has neither, so each appears once
		"<mods:enumerationAndChronology>": 1,
		"<mods:itemIdentifier":            1,
	} {
		if got := strings.Count(out, element); got != want {
			t.Errorf("%s appears %d times, want %d\n%s", element, got, want, out)
		}
	}
	// Titles keep the record's order, and the items come after them.
	if strings.Index(out, "Fotoalbum") > strings.Index(out, "Photo album") || strings.Index(out, "Photo album") > strings.Index(out, "R&amp;D") {
		t.Error("title order not preserved")
	}
	if strings.Index(out, "<mods:location>") < strings.LastIndex(out, "</mods:titleInfo>") {
		t.Error("location must follow the titles")
	}
}

// A record without items has no location at all: an empty holdingSimple
// would claim the library holds no copy.
func TestRecordEncodeWithoutItems(t *testing.T) {
	r := testRecord()
	r.Items = nil
	if out := encodeRecord(t, r, "../../schemas"); strings.Contains(out, "<mods:location") {
		t.Errorf("location emitted without items\n%s", out)
	}
}

// Other identifiers follow the catalogue identifier, in the record's order,
// each without a type, and before the titles.
func TestRecordEncodeOtherIdentifiers(t *testing.T) {
	r := testRecord()
	r.OtherIdentifiers = []string{"9789000000000", "(RUG01)000000001 & <x>"}
	out := encodeRecord(t, r, "../../schemas")
	want := `  <mods:identifier type="local">example-0001</mods:identifier>
  <mods:identifier>9789000000000</mods:identifier>
  <mods:identifier>(RUG01)000000001 &amp; &lt;x&gt;</mods:identifier>
  <mods:titleInfo xml:lang="nl">`
	if !strings.Contains(out, want) {
		t.Errorf("output missing\n%s\n--- got\n%s", want, out)
	}
}

// Contributors follow the titles, in the record's order, each as one name
// with one namePart and no type or role, and before the copies.
func TestRecordEncodeContributors(t *testing.T) {
	r := testRecord()
	r.Contributors = []string{"Doe, Jane 1950-", "Example & Sons"}
	out := encodeRecord(t, r, "../../schemas")
	want := `  </mods:titleInfo>
  <mods:name>
    <mods:namePart>Doe, Jane 1950-</mods:namePart>
  </mods:name>
  <mods:name>
    <mods:namePart>Example &amp; Sons</mods:namePart>
  </mods:name>
  <mods:location>`
	if !strings.Contains(out, want) {
		t.Errorf("output missing\n%s\n--- got\n%s", want, out)
	}
}

// A copy without a call number has no shelfLocator rather than an empty one.
func TestRecordEncodeItemWithoutCallNumber(t *testing.T) {
	r := testRecord()
	r.Items = []Item{{Barcode: "000000456"}}
	out := encodeRecord(t, r, "../../schemas")
	if strings.Contains(out, "<mods:shelfLocator") {
		t.Errorf("shelfLocator emitted for a copy without a call number\n%s", out)
	}
	want := "<mods:copyInformation>\n        <mods:itemIdentifier type=\"barcode\">000000456</mods:itemIdentifier>\n      </mods:copyInformation>"
	if !strings.Contains(out, want) {
		t.Errorf("output missing %s\n%s", want, out)
	}
}

// A representation's record may state no identifier. The document then
// carries none rather than an empty element.
func TestRecordEncodeWithoutIdentifier(t *testing.T) {
	r := Record{Titles: []Title{{Value: "PDF-versie", Lang: "nl"}}}
	out := encodeRecord(t, r, "../../../../schemas")
	if strings.Contains(out, "<mods:identifier") {
		t.Errorf("identifier emitted for a record without one\n%s", out)
	}
	if !strings.Contains(out, "<mods:title>PDF-versie</mods:title>") {
		t.Errorf("title missing\n%s", out)
	}
}

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestRecordEncodeSchemaLocation(t *testing.T) {
	out := encodeRecord(t, testRecord(), "../../../../schemas")
	if !strings.Contains(out, `xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../../../schemas/mods-3-7.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", out)
	}
}
