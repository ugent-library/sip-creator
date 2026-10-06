package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
