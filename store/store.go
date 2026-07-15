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

type Store struct {
	root string
}

func New(root string) *Store {
	return &Store{
		root: root,
	}
}

type Info struct {
	Size     string
	Checksum string
	Created  string
}

func (s *Store) MkdirAll(rel string) error {
	path := filepath.Join(s.root, rel)
	if err := os.MkdirAll(path, 0775); err != nil {
		return fmt.Errorf("mkdir %s: %w", rel, err)
	}
	return nil
}

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
	defer out.Close()

	hash := md5.New()
	if _, err := io.Copy(io.MultiWriter(out, hash), in); err != nil {
		out.Close()
		return Info{}, fmt.Errorf("copy %s: %w", rel, err)
	}
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
