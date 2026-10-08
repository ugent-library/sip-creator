package premis

import (
	"bytes"
	"encoding/xml"
	"io"
	"testing"

	"github.com/ugent-library/sip-creator/sip"
)

// Values from producers reach the documents escaped. A local identifier
// and an original file name that carry the XML-active characters leave
// both documents well-formed. An XML reader gets each value back as it was
// given.
func TestEscapesGraphValues(t *testing.T) {
	const (
		localID = `R&D <001> "a"`
		name    = `R&D "1" <a>.tif`
	)
	entity := sip.NewEntity()
	entity.AdditionalIdentifiers["MEEMOO-LOCAL-ID"] = localID
	rep := sip.NewRepresentation("master")
	rep.Entity = entity
	essence := sip.NewFile()
	essence.Name = name
	essence.Representation = rep
	rep.Files = []*sip.File{essence}
	entity.Representations = []*sip.Representation{rep}

	var entityDoc, repDoc bytes.Buffer
	if err := EncodeEntity(&entityDoc, entity); err != nil {
		t.Fatalf("package PREMIS: %v", err)
	}
	if err := EncodeRepresentation(&repDoc, rep); err != nil {
		t.Fatalf("representation PREMIS: %v", err)
	}

	if !textNodes(t, entityDoc.Bytes())[localID] {
		t.Errorf("package PREMIS does not carry the local identifier %q\n%s", localID, entityDoc.String())
	}
	if !textNodes(t, repDoc.Bytes())[name] {
		t.Errorf("representation PREMIS does not carry the original name %q\n%s", name, repDoc.String())
	}
}

// textNodes reads the whole document and returns every text node as an XML
// reader decodes it. It fails the test if the document is not well-formed.
func textNodes(t *testing.T, doc []byte) map[string]bool {
	t.Helper()
	texts := map[string]bool{}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return texts
		}
		if err != nil {
			t.Fatalf("not well-formed: %v\n%s", err, doc)
		}
		if text, ok := tok.(xml.CharData); ok {
			texts[string(text)] = true
		}
	}
}
