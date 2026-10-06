// Package store writes the files of one package under its directory. It
// creates directories, copies content files and writes generated metadata
// documents, and measures each file's size, MD5 checksum and modification
// time as it writes it.
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
// package-relative slash paths; the store never reads or deletes a file
// in the package.
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
	// Created is the time the file was written into the package, RFC 3339
	// with nanoseconds: the creation date of the file in the package, which
	// E-ARK CSIP's CREATED attribute declares. For a copy it differs from
	// the file's modification time, which keeps the source's.
	Created string
}

// MkdirAll creates the directory rel and any missing parents.
func (s *Store) MkdirAll(rel string) error {
	path := filepath.Join(s.root, rel)
	if err := os.MkdirAll(path, 0775); err != nil {
		return fmt.Errorf("mkdir %s: %w", rel, err)
	}
	return nil
}

// CopyFile streams src to rel, so large essence files are never buffered
// in memory, and reads the size from the written file. An existing file is
// truncated. knownMD5 is the file's MD5 as the caller already holds it,
// such as from a characterization report: when set, the copy computes no
// checksum and Info reports knownMD5 as given, because MD5 on one core is
// slower than the disk; when empty, the MD5 is computed during the copy.
// The copy keeps the source's modification time, as cp -p does: it is the
// one date the producer's file system holds about the file, and it cannot
// be recovered once lost. Info still reports the time the copy was written
// as Created.
func (s *Store) CopyFile(src, rel, knownMD5 string) (Info, error) {
	in, err := os.Open(src)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}
	defer in.Close()
	source, err := in.Stat()
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	dest := filepath.Join(s.root, rel)
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	checksum, err := copyBytes(out, in, knownMD5)
	if err != nil {
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
	// Read the copy's own time first: restoring the source's below
	// replaces it. A zero access time leaves the access time as it is.
	created := file.ModTime()
	if err := os.Chtimes(dest, time.Time{}, source.ModTime()); err != nil {
		return Info{}, fmt.Errorf("copy %s: keeping the modification time: %w", rel, err)
	}

	return Info{
		Size:     strconv.FormatInt(file.Size(), 10),
		Checksum: checksum,
		Created:  created.Format(time.RFC3339Nano),
	}, nil
}

// copyBytes copies in to out and returns the MD5 of the bytes copied, or
// knownMD5 as given when it is set, without computing one.
func copyBytes(out io.Writer, in io.Reader, knownMD5 string) (string, error) {
	if knownMD5 != "" {
		_, err := io.Copy(out, in)
		return knownMD5, err
	}
	hash := md5.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), in); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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
