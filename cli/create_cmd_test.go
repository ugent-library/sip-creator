package cli

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/ugent-library/sip-creator/profiles"
	"github.com/ugent-library/sip-creator/sip"
)

// --status and --updates come as a pair: an update-class status names an
// earlier package, and naming one needs an update-class status. A status
// is read in any case; one outside the SIP3 vocabulary is refused. The
// identifier's form is the library's rule (SourcePackage.Validate), not
// checked here.
func TestRecordStatusFromFlags(t *testing.T) {
	const earlier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"
	tests := []struct {
		name            string
		status, updates string
		wantStatus      sip.RecordStatus
		wantErr         string // "" means accepted; else substring of the error
	}{
		{"neither flag", "", "", "", ""},
		{"a new package", "new", "", sip.RecordStatusNew, ""},
		{"a test package", "TEST", "", sip.RecordStatusTest, ""},
		{"an update with the earlier package", "Replacement", earlier, sip.RecordStatusReplacement, ""},
		{"delete, also an update", "delete", earlier, sip.RecordStatusDelete, ""},
		{"a status outside the vocabulary", "update", earlier, "", "SIP3 vocabulary"},
		{"an update without the earlier package", "supplement", "", "", "--status supplement updates an earlier package; pass its identifier with --updates"},
		{"an earlier package without a status", "", earlier, "", "--updates names an earlier package"},
		{"an earlier package for a new package", "new", earlier, "", "--updates names an earlier package"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, updates, err := recordStatusFromFlags(commandWithFlags(t, tt.status, tt.updates))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one mentioning %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v, want the flags accepted", err)
			}
			if status != tt.wantStatus || updates != tt.updates {
				t.Errorf("got %q, %q; want %q, %q", status, updates, tt.wantStatus, tt.updates)
			}
		})
	}
}

// commandWithFlags returns a command with create's --status and --updates
// flags set to the given values; an empty value leaves the flag unset. A
// fresh command per case, so no test changes the flags of createCmd.
func commandWithFlags(t *testing.T, status, updates string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("updates", "", "")
	for name, value := range map[string]string{"status": status, "updates": updates} {
		if value == "" {
			continue
		}
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	return cmd
}

// packageMETS holds what the create tests read from a package METS: the
// identifier, the content category, the record status and the agents.
type packageMETS struct {
	ObjID string `xml:"OBJID,attr"`
	Type  string `xml:"TYPE,attr"`
	Hdr   struct {
		RecordStatus string `xml:"RECORDSTATUS,attr"`
		Agents       []struct {
			Type string `xml:"TYPE,attr"`
			Name string `xml:"name"`
			Note string `xml:"note"`
		} `xml:"agent"`
	} `xml:"metsHdr"`
}

// create carries the operator's flags and configuration into the package:
// the submitter from the environment, the record status and the earlier
// package's identifier from --status and --updates, the content category
// from --content-category before SIP_CONTENT_CATEGORY before the profile's
// default, and a zip unless --no-zip. A package that loses --status would
// be ingested as new beside the one it was meant to replace.
func TestCreateCarriesFlagsAndConfiguration(t *testing.T) {
	const earlier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"
	basicDefault := profileDefaultType(t, "meemoo/basic")
	tests := []struct {
		name            string
		profile         string
		contentCategory string // SIP_CONTENT_CATEGORY
		flags           []string
		wantObjID       string // "" means a minted identifier
		wantStatus      string
		wantType        string
		wantZip         bool
	}{
		{"the profile's values and a zip", "meemoo/basic", "", nil, "", "", basicDefault, true},
		{"an update of an earlier package", "ugent/basic", "",
			[]string{"--status", "replacement", "--updates", earlier}, earlier, "REPLACEMENT", "Mixed", true},
		{"the configured content category", "ugent/basic", "Textual works – Print", nil, "", "", "Textual works – Print", true},
		{"the flag before the configured content category", "ugent/basic", "Textual works – Print",
			[]string{"--content-category", "Maps"}, "", "", "Maps", true},
		{"no zip", "ugent/basic", "", []string{"--no-zip"}, "", "", "Mixed", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SIP_SUBMITTER_NAME", "Example Organization")
			t.Setenv("SIP_SUBMITTER_OR_ID", "OR-a1b2c3d")
			t.Setenv("SIP_CONTENT_CATEGORY", tt.contentCategory)
			dest := t.TempDir()

			args := append([]string{"create", "--profile", tt.profile}, tt.flags...)
			args = append(args, filepath.Join("..", "examples", filepath.FromSlash(tt.profile)), dest)
			if _, stderr, err := runCLI(t, args...); err != nil {
				t.Fatalf("create: %v\n%s", err, stderr)
			}

			mets := readPackageMETS(t, dest)
			if tt.wantObjID != "" && mets.ObjID != tt.wantObjID {
				t.Errorf("OBJID = %q, want %q", mets.ObjID, tt.wantObjID)
			}
			if err := sip.ValidateIdentifier(mets.ObjID); err != nil {
				t.Errorf("OBJID: %v", err)
			}
			if mets.Hdr.RecordStatus != tt.wantStatus {
				t.Errorf("RECORDSTATUS = %q, want %q", mets.Hdr.RecordStatus, tt.wantStatus)
			}
			if mets.Type != tt.wantType {
				t.Errorf("TYPE = %q, want %q", mets.Type, tt.wantType)
			}
			if !hasSubmitter(mets, "Example Organization") {
				t.Errorf("no ORGANIZATION agent named %q: %+v", "Example Organization", mets.Hdr.Agents)
			}
			_, err := os.Stat(filepath.Join(dest, mets.ObjID+".zip"))
			if gotZip := err == nil; gotZip != tt.wantZip {
				t.Errorf("zip written = %v, want %v", gotZip, tt.wantZip)
			}
		})
	}
}

// Without a submitter, create stops before reading the folder and names
// the settings to set; a folder that breaks the input rules gets its
// violations on stderr; an update whose earlier zip is still in dest is
// refused before the build. None of them writes anything.
func TestCreateRefusesBeforeWriting(t *testing.T) {
	t.Run("no submitter", func(t *testing.T) {
		t.Setenv("SIP_SUBMITTER_NAME", "")
		t.Setenv("SIP_SUBMITTER_OR_ID", "")
		dest := t.TempDir()
		_, _, err := runCLI(t, "create", "--profile", "ugent/basic", filepath.Join("..", "examples", "ugent", "basic"), dest)
		if want := "(set SIP_SUBMITTER_NAME and SIP_SUBMITTER_OR_ID)"; err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error = %v, want one ending %q", err, want)
		}
		requireNothingWritten(t, dest)
	})
	t.Run("input violations", func(t *testing.T) {
		t.Setenv("SIP_SUBMITTER_NAME", "Example Organization")
		src := writeFolder(t, map[string]string{"scan.tif": "x"}) // no description
		dest := t.TempDir()
		_, stderr, err := runCLI(t, "create", "--profile", "ugent/basic", src, dest)
		if want := src + ": 1 problem(s) found"; err == nil || err.Error() != want {
			t.Errorf("error = %v, want %q", err, want)
		}
		if !strings.Contains(stderr, "descriptive metadata is missing") {
			t.Errorf("stderr = %q, want the violation", stderr)
		}
		requireNothingWritten(t, dest)
	})
	t.Run("the earlier package's zip in dest", func(t *testing.T) {
		const earlier = "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"
		t.Setenv("SIP_SUBMITTER_NAME", "Example Organization")
		t.Setenv("SIP_SUBMITTER_OR_ID", "OR-a1b2c3d")
		dest := t.TempDir()
		oldZip := filepath.Join(dest, earlier+".zip")
		if err := os.WriteFile(oldZip, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, err := runCLI(t, "create", "--profile", "ugent/basic", "--status", "replacement", "--updates", earlier,
			filepath.Join("..", "examples", "ugent", "basic"), dest)
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error = %v, want one saying the zip already exists", err)
		}
		entries, err := os.ReadDir(dest)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != earlier+".zip" {
			t.Errorf("%s holds %v, want only the old zip", dest, entries)
		}
	})
}

// profileDefaultType returns the content category the registered profile
// declares.
func profileDefaultType(t *testing.T, name string) string {
	t.Helper()
	def, ok := profiles.Get(name)
	if !ok {
		t.Fatalf("no %q profile registered", name)
	}
	return def.Declaration.Type
}

// readPackageMETS reads the METS of the one package create built under dest.
func readPackageMETS(t *testing.T, dest string) packageMETS {
	t.Helper()
	dirs, err := filepath.Glob(filepath.Join(dest, "uuid-*", "METS.xml"))
	if err != nil || len(dirs) != 1 {
		t.Fatalf("want one package under %s, found %v (%v)", dest, dirs, err)
	}
	data, err := os.ReadFile(dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	var mets packageMETS
	if err := xml.Unmarshal(data, &mets); err != nil {
		t.Fatalf("%s: %v", dirs[0], err)
	}
	return mets
}

// hasSubmitter reports whether the METS lists an ORGANIZATION agent with
// the given name.
func hasSubmitter(mets packageMETS, name string) bool {
	for _, a := range mets.Hdr.Agents {
		if a.Type == "ORGANIZATION" && a.Name == name {
			return true
		}
	}
	return false
}

// requireNothingWritten fails the test unless dir is empty.
func requireNothingWritten(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("%s holds %v, want nothing written", dir, entries)
	}
}
