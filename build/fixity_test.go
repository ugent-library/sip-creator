package build_test

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/profiles/mods"
	"github.com/ugent-library/sip-creator/profiles/simpledc"
	"github.com/ugent-library/sip-creator/sip"
)

// Every reference a built package makes names a file the package holds,
// with that file's size and checksum: each href in the package METS and in
// every representation METS, and each file object in the representation
// PREMIS. The METS documents carry the fixity of files written before
// them, so a change to the write order, or a file whose fixity the writer
// does not record, shows up here. The input uses every kind of file a
// package can hold: nested essence with a characterization report,
// documentation and received PREMIS at both levels, and a representation
// description where the profile allows one.
func TestBuildReferencesMatchDisk(t *testing.T) {
	cases := []struct {
		name           string
		def            build.Definition
		description    sip.Description
		repDescription sip.Description
	}{
		{"meemoo/basic", basicDef(t), testDescription(), nil},
		{"ugent/basic", ugentBasicDef(t), identityTerms(), simpledc.Terms{{Key: "rights", Value: "CC BY 4.0"}}},
		{"ugent/bibliographic", bibliographicDef(t), identityRecord(), mods.Record{Titles: []mods.Title{{Value: "PDF-versie", Lang: "nl"}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, in, _ := newTestBuilder(t, c.def)
			dir := t.TempDir()
			in.Description = c.description
			rep := &in.Representations[0]
			rep.Description = c.repDescription
			rep.Files = []build.SourceFile{
				writeEssence(t, dir, "scan-001.tif", "first scan"),
				writeEssence(t, dir, "sub/scan-002.tif", "second scan, a longer one"),
			}
			rep.Documentation = []build.SourceFile{writeEssence(t, dir, "rep-notes.txt", "capture notes")}
			rep.Premis = []build.SourceFile{writeEssence(t, dir, "capture.xml", validPremis)}
			in.Documentation = []build.SourceFile{writeEssence(t, dir, "manual.pdf", "manual")}
			in.Premis = []build.SourceFile{writeEssence(t, dir, "events.xml", validPremis)}
			in.Characterization = report(t, rep.Files...)

			pkg, err := b.Build(in)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			requireReferencesMatchDisk(t, filepath.Join(pkg.Location, "METS.xml"))
			for _, r := range pkg.Root.Representations {
				metsPath := filepath.Join(pkg.Location, "representations", r.Name, "METS.xml")
				requireReferencesMatchDisk(t, metsPath)
				if r.PremisFile != nil {
					premisPath := filepath.Join(pkg.Location, "representations", r.Name, filepath.FromSlash(r.PremisFile.Path))
					requirePremisFixityMatchesDisk(t, premisPath, metsPath, len(r.Files))
				}
			}
		})
	}
}

// reference is one xlink:href in a METS document with the fixity METS
// declares for it: a file's FLocat with the file's SIZE and CHECKSUM, an
// mdRef with its own, or an mptr, which declares none (declaresFixity
// false).
type reference struct {
	id, href, size, checksum string
	declaresFixity           bool
}

// metsReferences returns every reference in the METS document at path,
// failing the test unless the document is well-formed.
func metsReferences(t *testing.T, path string) []reference {
	t.Helper()
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	attr := func(start xml.StartElement, local string) string {
		for _, a := range start.Attr {
			if a.Name.Local == local {
				return a.Value
			}
		}
		return ""
	}

	var refs []reference
	var file reference // the file element whose FLocat comes next
	dec := xml.NewDecoder(bytes.NewReader(doc))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return refs
		}
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "file":
			file = reference{id: attr(start, "ID"), size: attr(start, "SIZE"), checksum: attr(start, "CHECKSUM"), declaresFixity: true}
		case "FLocat":
			file.href = attr(start, "href")
			refs = append(refs, file)
		case "mdRef":
			refs = append(refs, reference{href: attr(start, "href"), size: attr(start, "SIZE"), checksum: attr(start, "CHECKSUM"), declaresFixity: true})
		case "mptr":
			refs = append(refs, reference{href: attr(start, "href")})
		}
	}
}

// requireReferencesMatchDisk fails the test for every reference in the
// METS document at metsPath that, percent-decoded and taken relative to
// the document, names no file, or whose declared size or checksum is not
// the file's. A reference without declared fixity (an mptr) must still
// name a file.
func requireReferencesMatchDisk(t *testing.T, metsPath string) {
	t.Helper()
	refs := metsReferences(t, metsPath)
	if len(refs) == 0 {
		t.Fatalf("%s references no file", metsPath)
	}
	for _, ref := range refs {
		target, ok := resolveHref(t, metsPath, ref.href)
		if !ok || !ref.declaresFixity {
			continue
		}
		requireFixity(t, target, ref.size, ref.checksum, metsPath+" href "+ref.href)
	}
}

// resolveHref returns the file href names, relative to the METS document
// at metsPath, failing the test when it names none.
func resolveHref(t *testing.T, metsPath, href string) (string, bool) {
	t.Helper()
	rel, err := url.PathUnescape(href)
	if err != nil {
		t.Errorf("%s: href %q does not decode: %v", metsPath, href, err)
		return "", false
	}
	target := filepath.Join(filepath.Dir(metsPath), filepath.FromSlash(rel))
	if _, err := os.Stat(target); err != nil {
		t.Errorf("%s: href %q names no file: %v", metsPath, href, err)
		return "", false
	}
	return target, true
}

// requireFixity fails the test unless the file at path has the declared
// size and MD5 checksum; where names the declaration in the message.
func requireFixity(t *testing.T, path, size, checksum, where string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := strconv.FormatInt(info.Size(), 10); size != want {
		t.Errorf("%s: size %q, the file has %s bytes", where, size, want)
	}
	if want := fileMD5(t, path); checksum != want {
		t.Errorf("%s: checksum %q, the file's MD5 is %s", where, checksum, want)
	}
}

// premisDocument holds what the fixity check reads from a representation
// PREMIS document: each object's type, identifiers and fixity.
type premisDocument struct {
	Objects []struct {
		Type        string   `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		Identifiers []string `xml:"objectIdentifier>objectIdentifierValue"`
		Digest      string   `xml:"objectCharacteristics>fixity>messageDigest"`
		Size        string   `xml:"objectCharacteristics>size"`
	} `xml:"object"`
}

// requirePremisFixityMatchesDisk fails the test unless the representation
// PREMIS document at premisPath describes wantFiles files, each one the
// file the representation METS at metsPath lists under the same
// identifier, with that file's size and checksum.
func requirePremisFixityMatchesDisk(t *testing.T, premisPath, metsPath string, wantFiles int) {
	t.Helper()
	data, err := os.ReadFile(premisPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc premisDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("%s: %v", premisPath, err)
	}

	hrefByID := map[string]string{}
	for _, ref := range metsReferences(t, metsPath) {
		if ref.id != "" {
			hrefByID[ref.id] = ref.href
		}
	}

	files := 0
	for _, obj := range doc.Objects {
		if obj.Type != "premis:file" {
			continue
		}
		files++
		if len(obj.Identifiers) == 0 {
			t.Errorf("%s: a file object carries no identifier", premisPath)
			continue
		}
		id := obj.Identifiers[0]
		href, ok := hrefByID[id]
		if !ok {
			t.Errorf("%s: file object %s is not a file of %s", premisPath, id, metsPath)
			continue
		}
		if target, ok := resolveHref(t, metsPath, href); ok {
			requireFixity(t, target, obj.Size, obj.Digest, premisPath+" object "+id)
		}
	}
	if files != wantFiles {
		t.Errorf("%s describes %d files, want %d", premisPath, files, wantFiles)
	}
}
