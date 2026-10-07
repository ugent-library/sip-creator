package build_test

import (
	stdzip "archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ugent-library/sip-creator/archive"
	"github.com/ugent-library/sip-creator/build"
	"github.com/ugent-library/sip-creator/sip"
)

// Under the UGent profiles a package may hold no representation: it
// carries the description and the schemas, and its METS references no
// representation (CSIP58). The representations/ directory is still written,
// empty, and the zip carries it.
func TestBuildMetadataOnly(t *testing.T) {
	cases := []struct {
		profile     string
		def         build.Definition
		description sip.Description
	}{
		{"ugent/basic", ugentBasicDef(t), identityTerms()},
		{"ugent/bibliographic", bibliographicDef(t), identityRecord()},
	}
	for _, tt := range cases {
		t.Run(tt.profile, func(t *testing.T) {
			b, in, outDir := newTestBuilder(t, tt.def)
			in.Description = tt.description
			in.Representations = nil

			pkg, err := b.Build(in)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}

			mets, err := os.ReadFile(filepath.Join(pkg.Location, "METS.xml"))
			if err != nil {
				t.Fatal(err)
			}
			for _, absent := range []string{`USE="Representations/`, `LABEL="Representations/`} {
				if strings.Contains(string(mets), absent) {
					t.Errorf("package METS has %s, want no representation", absent)
				}
			}
			for _, present := range []string{`USE="Schemas"`, `LABEL="Metadata"`, `LABEL="Schemas"`} {
				if !strings.Contains(string(mets), present) {
					t.Errorf("package METS lacks %s", present)
				}
			}
			requireReferencesMatchDisk(t, filepath.Join(pkg.Location, "METS.xml"))

			entries, err := os.ReadDir(filepath.Join(pkg.Location, "representations"))
			if err != nil || len(entries) != 0 {
				t.Errorf("representations/ = %v (%v), want an empty directory", entries, err)
			}

			if err := archive.New(&archive.Config{Destination: outDir}).Zip(pkg); err != nil {
				t.Fatalf("Zip: %v", err)
			}
			zr, err := stdzip.OpenReader(filepath.Join(outDir, pkg.Identifier+".zip"))
			if err != nil {
				t.Fatal(err)
			}
			defer zr.Close()
			want := pkg.Identifier + "/representations/"
			found := false
			for _, f := range zr.File {
				found = found || f.Name == want
			}
			if !found {
				t.Errorf("zip has no entry %q", want)
			}
		})
	}
}
