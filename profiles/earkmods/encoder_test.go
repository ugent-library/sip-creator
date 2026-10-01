package earkmods

import (
	"bytes"
	"strings"
	"testing"
)

func encode(t *testing.T, r Record, schemasDir string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := (mods{}).Encode(&buf, r, schemasDir); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

// The document for the test record, byte for byte: the bytes the template
// emitted before the record was typed by field (2026-09-30), so the model
// change leaves the output unchanged.
const goldenDocument = `<?xml version='1.0' encoding='UTF-8'?>
<mods:mods xmlns:mods="http://www.loc.gov/mods/v3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="3.7" xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../schemas/mods-3-7.xsd">
  <mods:identifier type="local">990001234560471</mods:identifier>
  <mods:titleInfo xml:lang="nl">
    <mods:title>Fotoalbum Gent 1913</mods:title>
  </mods:titleInfo>
  <mods:titleInfo xml:lang="en">
    <mods:title>Photo album Ghent 1913</mods:title>
  </mods:titleInfo>
  <mods:location>
    <mods:holdingSimple>
      <mods:copyInformation>
        <mods:shelfLocator>BIB.FA.001</mods:shelfLocator>
        <mods:enumerationAndChronology>vol. 3 (1913)</mods:enumerationAndChronology>
        <mods:itemIdentifier type="barcode">000000123</mods:itemIdentifier>
      </mods:copyInformation>
      <mods:copyInformation>
        <mods:shelfLocator>BIB.FA.002</mods:shelfLocator>
      </mods:copyInformation>
    </mods:holdingSimple>
  </mods:location>
</mods:mods>
`

func TestEncodeGolden(t *testing.T) {
	if got := encode(t, testRecord(), "../../schemas"); got != goldenDocument {
		t.Errorf("document differs from the golden bytes:\n--- got\n%s--- want\n%s", got, goldenDocument)
	}
}

func TestEncode(t *testing.T) {
	r := testRecord()
	// An untagged title with producer text that must be escaped.
	r.Titles = append(r.Titles, Title{Value: "R&D <scans> 'quote'"})
	out := encode(t, r, "../../schemas")

	for _, want := range []string{
		`<mods:mods xmlns:mods="http://www.loc.gov/mods/v3"`,
		`version="3.7"`,
		`xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../schemas/mods-3-7.xsd"`,
		`<mods:identifier type="` + localIdentifierType + `">990001234560471</mods:identifier>`,
		"<mods:titleInfo xml:lang=\"nl\">\n    <mods:title>Fotoalbum Gent 1913</mods:title>\n  </mods:titleInfo>",
		`<mods:titleInfo xml:lang="en">`,
		// no language tag, no xml:lang; producer values escaped
		"<mods:titleInfo>\n    <mods:title>R&amp;D &lt;scans&gt; &#39;quote&#39;</mods:title>",
		// the items as one location, one copyInformation per item
		"<mods:location>\n    <mods:holdingSimple>",
		`<mods:shelfLocator>BIB.FA.001</mods:shelfLocator>`,
		`<mods:enumerationAndChronology>vol. 3 (1913)</mods:enumerationAndChronology>`,
		`<mods:itemIdentifier type="barcode">000000123</mods:itemIdentifier>`,
		`<mods:shelfLocator>BIB.FA.002</mods:shelfLocator>`,
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
func TestEncodeWithoutItems(t *testing.T) {
	r := testRecord()
	r.Items = nil
	if out := encode(t, r, "../../schemas"); strings.Contains(out, "<mods:location") {
		t.Errorf("location emitted without items\n%s", out)
	}
}

// A representation's record may state no identifier; the document then
// carries none rather than an empty element.
func TestEncodeWithoutIdentifier(t *testing.T) {
	r := Record{Titles: []Title{{Value: "PDF-versie", Lang: "nl"}}}
	out := encode(t, r, "../../../../schemas")
	if strings.Contains(out, "<mods:identifier") {
		t.Errorf("identifier emitted for a record without one\n%s", out)
	}
	if !strings.Contains(out, "<mods:title>PDF-versie</mods:title>") {
		t.Errorf("title missing\n%s", out)
	}
}

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestEncodeSchemaLocation(t *testing.T) {
	out := encode(t, testRecord(), "../../../../schemas")
	if !strings.Contains(out, `xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../../../schemas/mods-3-7.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", out)
	}
}
