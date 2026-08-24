package store

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Store writes a package's files under one root directory. Callers speak
// package-relative slash paths; the store never reads or deletes.
type Store struct {
	root string
}

// New returns a Store rooted at root.
func New(root string) *Store {
	return &Store{
		root: root,
	}
}

// Info is what the store measured about a written file: the values the
// METS and PREMIS documents declare as fixity.
type Info struct {
	// Size is the byte size, decimal.
	Size string
	// Checksum is the MD5, hex-encoded.
	Checksum string
	// Created is the modification time, RFC 3339 with nanoseconds.
	Created string
}

func (s *Store) MkdirAll(rel string) error {
	path := filepath.Join(s.root, rel)
	if err := os.MkdirAll(path, 0775); err != nil {
		return fmt.Errorf("mkdir %s: %w", rel, err)
	}
	return nil
}

// CopyFile streams src to rel, computing the MD5 and size during the copy
// so large essence files are never buffered in memory. An existing file is
// truncated.
func (s *Store) CopyFile(src, rel string) (Info, error) {
	in, err := os.Open(src)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}
	defer in.Close()

	dest := filepath.Join(s.root, rel)
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	hash := md5.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), in); err != nil {
		out.Close()
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}
	// Close is checked, not deferred: a failed close means the bytes may
	// not all be on disk, and the fixity below must describe the file as
	// written.
	if err := out.Close(); err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	file, err := os.Stat(dest)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	return Info{
		Size:     strconv.FormatInt(file.Size(), 10),
		Checksum: hex.EncodeToString(hash.Sum(nil)),
		Created:  file.ModTime().Format(time.RFC3339Nano),
	}, nil
}

// WriteMetadata renders a document to memory before writing it to rel, so
// a failed render leaves no partial file on disk. An existing file is
// truncated.
func (s *Store) WriteMetadata(rel string, fn func(io.Writer) error) (Info, error) {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return Info{}, fmt.Errorf("write metadata %s: %w", rel, err)
	}

	sum := md5.Sum(buf.Bytes())
	dest := filepath.Join(s.root, rel)
	if err := os.WriteFile(dest, buf.Bytes(), 0600); err != nil {
		return Info{}, fmt.Errorf("write metadata %s: %w", rel, err)
	}

	file, err := os.Stat(dest)
	if err != nil {
		return Info{}, fmt.Errorf("write metadata %s: %w", rel, err)
	}

	return Info{
		Size:     strconv.FormatInt(file.Size(), 10),
		Checksum: hex.EncodeToString(sum[:]),
		Created:  file.ModTime().Format(time.RFC3339Nano),
	}, nil
}
