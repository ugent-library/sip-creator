package store

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// md5 of "hello world", computed independently; the tests must not
// reimplement the store's own hashing to check it.
const helloWorldMD5 = "5eb63bbbe01eeed093cb22bb8f5acdc3"

func readFile(t *testing.T, path string) string {
	t.Helper()
	bts, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(bts)
}

func TestWriteMetadata(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	info, err := s.WriteMetadata("premis.xml", func(w io.Writer) error {
		_, err := w.Write([]byte("hello world"))
		return err
	})
	if err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	if info.Checksum != helloWorldMD5 {
		t.Errorf("Checksum = %q, want %q", info.Checksum, helloWorldMD5)
	}
	if info.Size != "11" {
		t.Errorf("Size = %q, want %q", info.Size, "11")
	}
	if info.Created == "" {
		t.Error("Created is empty")
	}
	if got := readFile(t, filepath.Join(root, "premis.xml")); got != "hello world" {
		t.Errorf("file content = %q, want %q", got, "hello world")
	}
}

func TestWriteMetadataTruncatesOnRewrite(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	write := func(content string) Info {
		info, err := s.WriteMetadata("METS.xml", func(w io.Writer) error {
			_, err := w.Write([]byte(content))
			return err
		})
		if err != nil {
			t.Fatalf("WriteMetadata: %v", err)
		}
		return info
	}

	write("a much longer first run of content")
	info := write("short")

	// A re-run must replace the file, not append to it (the O_APPEND defect
	// in the old profile primitives).
	if got := readFile(t, filepath.Join(root, "METS.xml")); got != "short" {
		t.Errorf("file content after rewrite = %q, want %q", got, "short")
	}
	if info.Size != "5" {
		t.Errorf("Size after rewrite = %q, want %q", info.Size, "5")
	}
}

func TestWriteMetadataRenderErrorLeavesNoFile(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	renderErr := errors.New("template exploded")
	_, err := s.WriteMetadata("premis.xml", func(w io.Writer) error {
		_, _ = w.Write([]byte("partial out"))
		return renderErr
	})
	if !errors.Is(err, renderErr) {
		t.Fatalf("error = %v, want wrapped %v", err, renderErr)
	}

	if _, err := os.Stat(filepath.Join(root, "premis.xml")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a failed render left a file on disk")
	}
}

func TestCopyFile(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	src := filepath.Join(t.TempDir(), "cat.jpg")
	if err := os.WriteFile(src, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	info, err := s.CopyFile(src, "cat.jpg")
	if err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	if info.Checksum != helloWorldMD5 {
		t.Errorf("Checksum = %q, want %q", info.Checksum, helloWorldMD5)
	}
	if info.Size != "11" {
		t.Errorf("Size = %q, want %q", info.Size, "11")
	}
	if got := readFile(t, filepath.Join(root, "cat.jpg")); got != "hello world" {
		t.Errorf("copied content = %q, want %q", got, "hello world")
	}
}

func TestCopyFileTruncatesOnRewrite(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	srcDir := t.TempDir()
	long := filepath.Join(srcDir, "long")
	short := filepath.Join(srcDir, "short")
	if err := os.WriteFile(long, []byte("a much longer first run of content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(short, []byte("short"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := s.CopyFile(long, "essence.bin"); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if _, err := s.CopyFile(short, "essence.bin"); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	if got := readFile(t, filepath.Join(root, "essence.bin")); got != "short" {
		t.Errorf("content after re-copy = %q, want %q", got, "short")
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	_, err := s.CopyFile(filepath.Join(t.TempDir(), "nope.jpg"), "nope.jpg")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want wrapped os.ErrNotExist", err)
	}

	if _, err := os.Stat(filepath.Join(root, "nope.jpg")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a failed copy left a destination file on disk")
	}
}

func TestMkdirAll(t *testing.T) {
	root := t.TempDir()
	s := New(root)

	if err := s.MkdirAll("metadata/preservation"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Idempotent: an existing directory is not an error.
	if err := s.MkdirAll("metadata/preservation"); err != nil {
		t.Fatalf("MkdirAll on existing dir: %v", err)
	}

	fi, err := os.Stat(filepath.Join(root, "metadata/preservation"))
	if err != nil {
		t.Fatalf("stat created dir: %v", err)
	}
	if !fi.IsDir() {
		t.Error("created path is not a directory")
	}
}
