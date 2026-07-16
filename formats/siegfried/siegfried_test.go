package siegfried

import (
	"testing"
)

// stubTool builds an identificator whose "tool" is sh dumping a canned sf
// JSON fixture; the file path Identify appends lands in $0 and is ignored.
func stubTool(t *testing.T, fixture string) *siegfried {
	t.Helper()
	id, err := New("sh", []string{"-c", "cat testdata/" + fixture})
	if err != nil {
		t.Fatal(err)
	}
	return id.(*siegfried)
}

func TestIdentify(t *testing.T) {
	s := stubTool(t, "match.json")

	format, err := s.Identify("cat.jpg")
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if format == nil || format.FormatRegistry == nil {
		t.Fatalf("Identify returned %+v, want a format with a registry", format)
	}
	if format.FormatRegistry.Name != "pronom" {
		t.Errorf("registry Name = %q, want %q", format.FormatRegistry.Name, "pronom")
	}
	if format.FormatRegistry.Key != "fmt/44" {
		t.Errorf("registry Key = %q, want %q", format.FormatRegistry.Key, "fmt/44")
	}
}

// Regression test for the receiver-mutation bug: Identify used to append
// the path to s.args, so call N ran sf against files 1..N.
func TestIdentifyDoesNotMutateArgs(t *testing.T) {
	s := stubTool(t, "match.json")
	argsLen := len(s.args)

	for _, path := range []string{"a.jpg", "b.jpg"} {
		if _, err := s.Identify(path); err != nil {
			t.Fatalf("Identify(%s): %v", path, err)
		}
	}

	if len(s.args) != argsLen {
		t.Errorf("Identify mutated the receiver args: len %d, want %d", len(s.args), argsLen)
	}
}

func TestIdentifyNoMatch(t *testing.T) {
	s := stubTool(t, "nomatch.json")

	format, err := s.Identify("mystery.bin")
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if format != nil {
		t.Errorf("Identify returned %+v on no match, want nil", format)
	}
}

func TestIdentifyToolReportedError(t *testing.T) {
	s := stubTool(t, "error.json")

	if _, err := s.Identify("gone.jpg"); err == nil {
		t.Fatal("Identify swallowed the tool-reported file error")
	}
}

// A missing binary must surface as an error, not a panic on empty output.
func TestIdentifyMissingBinary(t *testing.T) {
	id, err := New("sip-creator-test-no-such-binary", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := id.Identify("cat.jpg"); err == nil {
		t.Fatal("Identify succeeded with a nonexistent binary")
	}
}
