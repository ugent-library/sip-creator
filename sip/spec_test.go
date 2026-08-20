package sip

import (
	"strings"
	"testing"
)

func TestValidateRecordStatus(t *testing.T) {
	for _, ok := range []string{"NEW", "SUPPLEMENT", "REPLACEMENT", "TEST", "VERSION", "DELETE"} {
		if err := ValidateRecordStatus(ok); err != nil {
			t.Errorf("ValidateRecordStatus(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"new", "UPDATE", ""} {
		if err := ValidateRecordStatus(bad); err == nil {
			t.Errorf("ValidateRecordStatus(%q) = nil, want error", bad)
		}
	}
}

func TestIsUpdateRecordStatus(t *testing.T) {
	for _, yes := range []string{"SUPPLEMENT", "REPLACEMENT", "VERSION", "DELETE", "replacement"} {
		if !IsUpdateRecordStatus(yes) {
			t.Errorf("IsUpdateRecordStatus(%q) = false, want true", yes)
		}
	}
	for _, no := range []string{"NEW", "TEST", ""} {
		if IsUpdateRecordStatus(no) {
			t.Errorf("IsUpdateRecordStatus(%q) = true, want false", no)
		}
	}
}

func TestValidateIdentifier(t *testing.T) {
	if err := ValidateIdentifier("uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e"); err != nil {
		t.Errorf("valid identifier rejected: %v", err)
	}
	for _, bad := range []string{"", "0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e", "uuid-nope", "my-id"} {
		if err := ValidateIdentifier(bad); err == nil {
			t.Errorf("ValidateIdentifier(%q) = nil, want error", bad)
		}
	}
}

func TestNewPackageIdentifier(t *testing.T) {
	minted := NewPackage("/dest", "")
	if err := ValidateIdentifier(minted.Identifier); err != nil {
		t.Errorf("minted identifier invalid: %v", err)
	}
	if !strings.HasSuffix(minted.Location, "/"+minted.Identifier) {
		t.Errorf("Location %q does not end in the identifier", minted.Location)
	}

	// An update reuses the original's identifier — and its directory name.
	reused := NewPackage("/dest", "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e")
	if reused.Identifier != "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e" {
		t.Errorf("supplied identifier not reused: %q", reused.Identifier)
	}
	if reused.Location != "/dest/uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e" {
		t.Errorf("Location = %q", reused.Location)
	}
}
