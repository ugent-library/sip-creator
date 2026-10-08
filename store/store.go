// Package store writes the files of one package under its directory. It
// creates directories, copies content files and writes generated metadata
// documents. It records each file's size, MD5 checksum and the time it was
// written as it writes it.
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

// Store writes a package's files under one root directory. Its methods take
// paths relative to that root, with slash separators.
type Store struct {
	root string
}

// New returns a Store rooted at root.
func New(root string) *Store {
	return &Store{
		root: root,
	}
}

// Info holds what the store measured about a written file, for the METS
// and PREMIS documents to declare.
type Info struct {
	// Size is the size in bytes, as a decimal number.
	Size string
	// Checksum is the MD5 checksum, hex-encoded.
	Checksum string
	// Created is the time the file was written into the package, in
	// RFC 3339 with nanoseconds. E-ARK CSIP's CREATED attribute declares
	// this date. For a copy it differs from the file's modification time,
	// which keeps the source's.
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

// CopyFile streams src to rel, so large essence files are never held in
// memory. An existing file at rel is truncated. knownMD5 is the file's MD5
// as the caller already holds it, such as from a characterization report.
// When knownMD5 is set, CopyFile computes no checksum and reports knownMD5
// as given (ADR-0032). When it is empty, CopyFile computes the MD5 during
// the copy. The copy keeps the
// source's modification time, as cp -p does, because it is the one date
// the producer's file system holds about the file and it cannot be
// recovered once lost. CopyFile returns the size read from the written
// file, the checksum, and the time the copy was written as Created.
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
	// A failed Close means the bytes may not all be on disk. Close is
	// checked here so the returned Info describes the file as written.
	if err := out.Close(); err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}

	file, err := os.Stat(dest)
	if err != nil {
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}
	// The copy's own modification time is read first, because restoring
	// the source's time below replaces it. A zero access time leaves the
	// access time unchanged.
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

// WriteMetadata renders a document with fn into memory, then writes it to
// rel, so a failed render leaves no partial file on disk. An existing file
// at rel is truncated. It returns the size, MD5 checksum and modification
// time of the written file.
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
