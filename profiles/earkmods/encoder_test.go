package earkmods

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

func encode(t *testing.T, r Record, schemas string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := (mods{}).Encode(&buf, r, schemas); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return buf.String()
}

func TestEncode(t *testing.T) {
	r := testRecord()
	// An untagged title with operator text that must be escaped.
	r.Terms = append(r.Terms, sip.Term{Key: "title", Value: "R&D <scans> 'quote'"})
	out := encode(t, r, "../../schemas")

	for _, want := range []string{
		`<mods:mods xmlns:mods="http://www.loc.gov/mods/v3"`,
		`version="3.7"`,
		`xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../schemas/mods-3-7.xsd"`,
		// one complete element per term, the type from the table
		`<mods:identifier type="` + mmsIDType + `">990001234560471</mods:identifier>`,
		"<mods:titleInfo xml:lang=\"nl\">\n    <mods:title>Fotoalbum Gent 1913</mods:title>\n  </mods:titleInfo>",
		`<mods:titleInfo xml:lang="en">`,
		// no language tag, no xml:lang; operator values escaped
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
	// Term order is the producer's order, and the items come after the terms.
	if strings.Index(out, "<mods:identifier") > strings.Index(out, "<mods:titleInfo") {
		t.Error("term order not preserved")
	}
	if strings.Index(out, "<mods:location>") < strings.LastIndex(out, "</mods:titleInfo>") {
		t.Error("location must follow the terms")
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

// The schema-location hint follows the document: a representation-level
// document (four levels deep) must point four levels up.
func TestEncodeSchemaLocation(t *testing.T) {
	out := encode(t, testRecord(), "../../../../schemas")
	if !strings.Contains(out, `xsi:schemaLocation="http://www.loc.gov/mods/v3 ../../../../schemas/mods-3-7.xsd"`) {
		t.Errorf("rep-level schema location hint wrong:\n%s", out)
	}
}

// The template's one guard: a key outside the vocabulary aborts the render
// before anything reaches the writer.
func TestEncodeRefusesUnknownKey(t *testing.T) {
	bad := Record{Terms: []sip.Term{{Key: "abstract", Value: "x"}}}
	var buf bytes.Buffer
	if err := (mods{}).Encode(&buf, bad, "../../schemas"); err == nil {
		t.Fatal("Encode accepted a key outside the MODS vocabulary")
	}
	if buf.Len() != 0 {
		t.Errorf("Encode wrote %d bytes despite refusing", buf.Len())
	}
}
