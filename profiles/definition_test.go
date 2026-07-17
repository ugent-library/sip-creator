package profiles

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The registry's basic entry must declare its family — a blanked family
// would otherwise only surface as a Build error at run time.
func TestBasicFamily(t *testing.T) {
	if def := basicDef(t); def.Family != FamilyMeemoo {
		t.Errorf(`basic Family = %q, want %q`, def.Family, FamilyMeemoo)
	}
}

// An unknown (or empty) family fails before assembly: a clean input error,
// nothing written to the destination.
func TestBuildUnknownFamily(t *testing.T) {
	for _, family := range []Family{"", "klingon"} {
		b, _, outDir := newTestBuilder(t)

		def := basicDef(t)
		def.Family = family

		_, err := b.Build(def)
		if err == nil {
			t.Fatalf("Build succeeded with family %q", family)
		}
		if !strings.Contains(err.Error(), `"basic"`) || !strings.Contains(err.Error(), string(family)) {
			t.Errorf("error %q does not name the definition and the family", err)
		}

		entries, err := os.ReadDir(outDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("failed Build wrote to the destination: %v", entries)
		}
	}
}

func TestGetUnknown(t *testing.T) {
	if _, ok := Get("nope"); ok {
		t.Error(`Get("nope") reported an unregistered profile as known`)
	}
}

func TestNames(t *testing.T) {
	want := []string{"basic", "eark"}
	if got := Names(); !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

// TestBuildBasic drives the basic profile end to end and pins its meemoo
// SIP 1.2 face: profile URLs, the 1.2 descriptive namespace, and the two
// required agent notes (docs/plans/meemoo-12.md).
func TestBuildBasic(t *testing.T) {
	b, _, outDir := newTestBuilder(t)

	pkg, err := b.Build(basicDef(t))
	if err != nil {
		t.Fatalf("Build(basic): %v", err)
	}
	root := filepath.Join(outDir, pkg.Identifier)

	mets, err := os.ReadFile(filepath.Join(root, "METS.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`PROFILE="https://earksip.dilcis.eu/profile/E-ARK-SIP.xml"`,
		`csip:OTHERCONTENTINFORMATIONTYPE="https://data.hetarchief.be/id/sip/1.2/basic"`,
		`csip:NOTETYPE="SOFTWARE VERSION"`,
		`csip:NOTETYPE="IDENTIFICATIONCODE"`,
	} {
		if !strings.Contains(string(mets), want) {
			t.Errorf("package METS missing %s", want)
		}
	}
	if strings.Contains(string(mets), "OTHERROLE") {
		t.Error("package METS still carries the stray OTHERROLE")
	}

	desc, err := os.ReadFile(filepath.Join(root, "metadata", "descriptive", "dc+schema.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desc), `xmlns="https://data.hetarchief.be/id/sip/1.2/basic"`) {
		t.Errorf("descriptive metadata is not in the 1.2 namespace:\n%s", desc)
	}
}

func earkDef(t *testing.T) Definition {
	t.Helper()
	def, ok := Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}
	return def
}

// TestBuildEark drives the eark profile end to end: premis-less output,
// simple-DC descriptive metadata, documentation copied and inventoried.
func TestBuildEark(t *testing.T) {
	b, inDir, outDir := newTestBuilder(t)

	if err := os.MkdirAll(filepath.Join(inDir, "documentation", "sub"), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "documentation", "manual.txt"), []byte("read me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "documentation", "sub", "notes.txt"), []byte("notes"), 0600); err != nil {
		t.Fatal(err)
	}

	pkg, err := b.Build(earkDef(t))
	if err != nil {
		t.Fatalf("Build(eark): %v", err)
	}
	root := filepath.Join(outDir, pkg.Identifier)

	// Premis-less: the Emit flags are off, so no premis.xml may exist.
	err = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "premis.xml" {
			t.Errorf("premis-less profile wrote %s", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// No meemoo layer.
	if _, ok := pkg.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
		t.Error("MEEMOO-LOCAL-ID extracted for the eark profile")
	}

	// Descriptive metadata is simple DC, not the meemoo dc+schema document.
	desc, err := os.ReadFile(filepath.Join(root, "metadata", "descriptive", "dc+schema.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(desc), "<simpledc") || strings.Contains(string(desc), "dcterms:") {
		t.Errorf("descriptive output is not a simple-DC document:\n%s", desc)
	}

	// Documentation copied with its structure.
	for rel, want := range map[string]string{
		"documentation/manual.txt":    "read me",
		"documentation/sub/notes.txt": "notes",
	} {
		bts, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("documentation not copied: %v", err)
		}
		if string(bts) != want {
			t.Errorf("%s = %q, want %q", rel, bts, want)
		}
	}

	// Package METS: E-ARK SIP profile, typed descriptive mdRef, a
	// documentation fileGrp, and no PREMIS references at all.
	mets, err := os.ReadFile(filepath.Join(root, "METS.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`PROFILE="https://earksip.dilcis.eu/profile/E-ARK-SIP-v2-2-0.xml"`,
		`MDTYPEVERSION="SimpleDC20021212"`,
		`USE="Documentation"`,
		`csip:CONTENTINFORMATIONTYPE="MIXED"`,
	} {
		if !strings.Contains(string(mets), want) {
			t.Errorf("package METS missing %s", want)
		}
	}
	for _, forbidden := range []string{"<amdSec>", "ADMID", "OTHERCONTENTINFORMATIONTYPE"} {
		if strings.Contains(string(mets), forbidden) {
			t.Errorf("package METS contains %s for a premis-less eark package", forbidden)
		}
	}
}

// An empty LocalIdentifierScheme disables local-identifier extraction —
// the first definition-driven behavior variation.
func TestAssembleWithoutLocalIdentifierScheme(t *testing.T) {
	b, _, _ := newTestBuilder(t)

	def := basicDef(t)
	def.LocalIdentifierScheme = ""

	pkg, err := b.assemble(def)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if _, ok := pkg.Root.AdditionalIdentifiers["MEEMOO-LOCAL-ID"]; ok {
		t.Error("MEEMOO-LOCAL-ID extracted despite an empty scheme")
	}
}
