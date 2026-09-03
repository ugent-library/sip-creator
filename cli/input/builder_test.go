package input

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/sip"
)

// The embedding-caller contract: a hand-constructed profiles.Input, with
// no metadata.csv or siegfried.json anywhere on disk, must build the same
// package graph the folder convention produces. The folder is one
// transport, not the API.
func TestBuilderInputEquivalence(t *testing.T) {
	def, ok := profiles.Get("basic")
	if !ok {
		t.Fatal(`no "basic" definition registered`)
	}
	def, err := def.WithSubmitter("Test Org", "OR-test")
	if err != nil {
		t.Fatal(err)
	}
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Via the folder convention. The basic profile requires meemoo's four
	// elements, so the file carries more than the convention's minimum.
	csv := "key,value\n" +
		"identifier,ID-1\n" +
		"title,Test\n" +
		"description[nl],Testbeschrijving\n" +
		"created,2026\n"
	root := writeTree(t, map[string]string{
		"metadata.csv":                     csv,
		"representations/master/scan.tiff": "essence bytes",
	})
	pkg, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	folderPkg, err := profiles.New(&profiles.Config{Destination: t.TempDir(), Logger: discard}).
		Build(def, pkg.BuilderInput())
	if err != nil {
		t.Fatalf("Build via folder: %v", err)
	}

	// Via hand-constructed data, as an embedding system would.
	handDir := t.TempDir()
	src := filepath.Join(handDir, "staged-essence.bin") // deliberately not the folder layout
	if err := os.WriteFile(src, []byte("essence bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	handIn := &profiles.Input{
		Descriptive: metadata.Terms{
			{Element: "dcterms:identifier", Value: "ID-1"},
			{Element: "dcterms:title", Value: "Test"},
			{Element: "dcterms:description", Lang: "nl", Value: "Testbeschrijving"},
			{Element: "dcterms:created", Value: "2026"},
		},
		Representations: []profiles.SourceRepresentation{
			{Name: "master", Files: []profiles.SourceFile{
				{Source: src, Key: "irrelevant-without-report", Path: "scan.tiff"},
			}},
		},
	}
	handPkg, err := profiles.New(&profiles.Config{Destination: t.TempDir(), Logger: discard}).
		Build(def, handIn)
	if err != nil {
		t.Fatalf("Build via hand-constructed input: %v", err)
	}

	// Same graph, modulo minted UUIDs: identity, structure, and fixity.
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
	if ff.Checksum != hf.Checksum || ff.Size != hf.Size {
		t.Errorf("essence fixity differs: folder %s/%v, hand %s/%v", ff.Checksum, ff.Size, hf.Checksum, hf.Size)
	}
}

func localID(p *sip.Package) string {
	return p.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]
}
