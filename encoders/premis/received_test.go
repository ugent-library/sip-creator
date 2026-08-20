package premis

import (
	"strings"
	"testing"
)

func TestValidateReceived(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string // "" means accepted; else substring of the error
	}{
		{"valid", `<?xml version="1.0"?><premis:premis xmlns:premis="http://www.loc.gov/premis/v3" version="3.0"><premis:event/></premis:premis>`, ""},
		{"valid default namespace", `<premis xmlns="http://www.loc.gov/premis/v3" version="3.0"/>`, ""},
		{"not xml", `not xml at all`, "not an XML document"},
		{"truncated", `<premis:premis xmlns:premis="http://www.loc.gov/premis/v3"><premis:event>`, "not well-formed"},
		{"empty", ``, "not an XML document"},
		{"wrong namespace", `<premis xmlns="http://www.loc.gov/premis/v2"/>`, "PREMIS 3 namespace"},
		{"wrong root", `<mets xmlns="http://www.loc.gov/METS/"/>`, "PREMIS 3 namespace"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReceived(strings.NewReader(tt.doc))
			if tt.want == "" {
				if err != nil {
					t.Fatalf("want accepted, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("want error mentioning %q, got %v", tt.want, err)
			}
		})
	}
}
