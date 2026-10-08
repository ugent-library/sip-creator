package sip

import (
	"strings"
	"testing"
)

func TestParseRecordStatus(t *testing.T) {
	for _, text := range []string{"REPLACEMENT", "replacement", "Replacement"} {
		got, err := ParseRecordStatus(text)
		if err != nil || got != RecordStatusReplacement {
			t.Errorf("ParseRecordStatus(%q) = %q, %v; want REPLACEMENT", text, got, err)
		}
	}
	for _, bad := range []string{"UPDATE", ""} {
		if _, err := ParseRecordStatus(bad); err == nil {
			t.Errorf("ParseRecordStatus(%q) = nil error, want error", bad)
		}
	}
}

func TestRecordStatusIsValid(t *testing.T) {
	for _, ok := range []RecordStatus{RecordStatusNew, RecordStatusSupplement, RecordStatusReplacement, RecordStatusTest, RecordStatusVersion, RecordStatusDelete} {
		if !ok.IsValid() {
			t.Errorf("%q.IsValid() = false, want true", ok)
		}
	}
	// Case matters.
	for _, bad := range []RecordStatus{"new", "UPDATE", ""} {
		if bad.IsValid() {
			t.Errorf("%q.IsValid() = true, want false", bad)
		}
	}
}

func TestRecordStatusIsUpdate(t *testing.T) {
	for _, yes := range []RecordStatus{RecordStatusSupplement, RecordStatusReplacement, RecordStatusVersion, RecordStatusDelete} {
		if !yes.IsUpdate() {
			t.Errorf("%q.IsUpdate() = false, want true", yes)
		}
	}
	for _, no := range []RecordStatus{RecordStatusNew, RecordStatusTest, "replacement", ""} {
		if no.IsUpdate() {
			t.Errorf("%q.IsUpdate() = true, want false", no)
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

	// An update reuses the original's identifier and its directory name.
	reused := NewPackage("/dest", "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e")
	if reused.Identifier != "uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e" {
		t.Errorf("supplied identifier not reused: %q", reused.Identifier)
	}
	if reused.Location != "/dest/uuid-0e7a2c4f-3f6e-4f3f-8f4b-2f8a9d3c1b5e" {
		t.Errorf("Location = %q", reused.Location)
	}
}
