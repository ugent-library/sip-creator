package input

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/cli/input/mapping"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// A build.SourcePackage built directly in Go, with no description.csv or
// siegfried.json anywhere on disk, must build the same package graph as
// the input folder produces. The folder is one way to supply a package,
// not the API.
func TestSourcePackageEquivalence(t *testing.T) {
	def, ok := profiles.Get("meemoo/basic")
	if !ok {
		t.Fatal(`no "meemoo/basic" definition registered`)
	}
	def, err := def.WithSubmitter("Test Org", "OR-test")
	if err != nil {
		t.Fatal(err)
	}
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Through an input folder. meemoo/basic requires four Meemoo
	// elements, so the file carries all four.
	csv := "key,value\n" +
		"identifier,ID-1\n" +
		"title,Test\n" +
		"description[nl],Testbeschrijving\n" +
		"created,2026\n"
	root := writeTree(t, map[string]string{
		"description.csv":                  csv,
		"representations/master/scan.tiff": "essence bytes",
	})
	pkg, err := Read(root, mapping.Meemoo{}, meemooDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	folderBuilder, err := build.New(&build.Config{Profile: def, Destination: t.TempDir(), Logger: discard})
	if err != nil {
		t.Fatal(err)
	}
	folderPkg, err := folderBuilder.Build(pkg)
	if err != nil {
		t.Fatalf("Build via folder: %v", err)
	}

	// Via hand-constructed data, as an embedding system would.
	handDir := t.TempDir()
	src := filepath.Join(handDir, "staged-essence.bin") // deliberately not the folder layout
	if err := os.WriteFile(src, []byte("essence bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	handIn := &build.SourcePackage{
		Description: meemoo.Terms{
			{Key: "dcterms:identifier", Value: "ID-1"},
			{Key: "dcterms:title", Value: "Test"},
			{Key: "dcterms:description", Lang: "nl", Value: "Testbeschrijving"},
			{Key: "dcterms:created", Value: "2026"},
		},
		Representations: []build.SourceRepresentation{
			{Name: "master", Files: []build.SourceFile{
				{Source: src, Key: "irrelevant-without-report", Path: "scan.tiff"},
			}},
		},
	}
	handBuilder, err := build.New(&build.Config{Profile: def, Destination: t.TempDir(), Logger: discard})
	if err != nil {
		t.Fatal(err)
	}
	handPkg, err := handBuilder.Build(handIn)
	if err != nil {
		t.Fatalf("Build via hand-constructed input: %v", err)
	}

	// The same graph apart from the minted UUIDs: identity, structure and
	// fixity.
	if a, b := localID(folderPkg), localID(handPkg); a != b || a != "ID-1" {
		t.Errorf("MEEMOO-LOCAL-ID differs: folder %q, hand %q", a, b)
	}
	fr, hr := folderPkg.Root.Representations, handPkg.Root.Representations
	if len(fr) != 1 || len(hr) != 1 || fr[0].Name != hr[0].Name {
		t.Fatalf("representations differ: folder %v, hand %v", fr, hr)
	}
	ff, hf := fr[0].Files[0], hr[0].Files[0]
	if ff.Path != hf.Path {
		t.Errorf("essence Path differs: folder %q, hand %q", ff.Path, hf.Path)
	}
	// Both are the fixity of "essence bytes", as md5(1) computes it.
	const wantChecksum, wantSize = "b04fc2b4b05b5c78a2a4fac253cdc66a", "13"
	for name, f := range map[string]*sip.File{"folder": ff, "hand": hf} {
		if f.Checksum != wantChecksum || f.Size != wantSize {
			t.Errorf("%s essence fixity = %s/%s, want %s/%s", name, f.Checksum, f.Size, wantChecksum, wantSize)
		}
	}
}

func localID(p *sip.Package) string {
	return p.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]
}

// A representation without its own description.csv must carry a nil
// Description. A typed nil (a nil meemoo.Terms) stored in the interface
// field would read as a present, empty description and earn the
// representation a descriptive document it never had.
func TestSourcePackageRepresentationWithoutDescriptive(t *testing.T) {
	root := writeTree(t, map[string]string{
		"description.csv":                        minimalCSV,
		"representations/access/description.csv": "key,value\ntitle,Access copy\n",
		"representations/access/scan.jpg":        "access bytes",
		"representations/master/scan.tiff":       "master bytes",
	})
	in, err := Read(root, mapping.Meemoo{}, meemooDocumentSpec)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(in.Representations) != 2 {
		t.Fatalf("got %d representations, want 2", len(in.Representations))
	}
	for _, r := range in.Representations {
		switch r.Name {
		case "access":
			if r.Description == nil {
				t.Error("access: Description is nil, want its title term")
			}
		case "master":
			if r.Description != nil {
				t.Errorf("master: Description = %#v, want nil (no description.csv)", r.Description)
			}
		}
	}
}
