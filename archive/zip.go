// Package archive zips a built package directory into uuid-<uuid>.zip,
// with every entry stored uncompressed.
package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/sip"
)

// Config is the archiver's wiring: where zips land and how the run logs.
type Config struct {
	// Destination is the directory the zip is written to.
	Destination string
	// Logger receives a message per zipped entry. Nil discards them.
	Logger *slog.Logger
}

// Archive zips built package directories. New returns one.
type Archive struct {
	destination string
	logger      *slog.Logger
}

// New returns an Archive that writes to config.Destination.
func New(config *Config) *Archive {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Archive{
		destination: config.Destination,
		logger:      logger,
	}
}

// Zip writes the package directory to dest/uuid-<uuid>.zip with every
// entry stored uncompressed. The final name only ever holds a complete
// zip: Zip refuses a zip that already exists, writes to a temporary file
// next to it, and renames that file once every entry is written. A failed
// zip leaves nothing behind; a process killed partway leaves only the
// temporary file, whose name does not end in .zip.
func (a *Archive) Zip(pkg *sip.Package) (err error) {
	dest := filepath.Join(a.destination, pkg.Identifier+".zip")
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("zip %s already exists; move it away first", dest)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking zip %s: %w", dest, err)
	}

	// The temporary file sits in the same directory, so the rename stays on
	// one file system, where it is atomic. It is opened with the mode
	// os.Create uses, so the zip gets the permissions it always had.
	tmp := filepath.Join(a.destination, "."+pkg.Identifier+".zip.tmp")
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("creating zip %s: %w", dest, err)
	}
	defer func() {
		if err != nil {
			file.Close() // after a failed Close this fails too; the first error is returned
			os.Remove(tmp)
		}
	}()

	if err := a.writeEntries(file, pkg.Location); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing zip %s: %w", dest, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("finalizing zip %s: %w", dest, err)
	}
	return nil
}

// writeEntries writes every entry under src to out as a zip, entry names
// relative to the archive's destination.
func (a *Archive) writeEntries(out io.Writer, src string) error {
	w := zip.NewWriter(out)

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Entry names are relative to the destination dir, so the package
		// dir (uuid-<uuid>/) stays the top-level entry; zip names are
		// slash-separated regardless of platform.
		rel, err := filepath.Rel(a.destination, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		a.logger.Info("zipping", slog.String("path", name))
		if d.IsDir() {
			// Directories need explicit entries (name ending in "/"):
			// readers otherwise infer them from file paths, and empty
			// directories vanish from the zip entirely.
			_, err := w.CreateHeader(&zip.FileHeader{
				Name:   name + "/",
				Method: zip.Store,
			})
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		// Method zip.Store keeps the entry uncompressed. The entry is
		// written with CreateRaw rather than CreateHeader so the size
		// and CRC land in the local file header: CreateHeader streams,
		// leaves them zero and sets general-purpose flag bit 3 ("sizes
		// follow the data in a descriptor"), and Java's ZipInputStream
		// throws "only DEFLATED entries can have EXT descriptor" on a
		// stored entry with that flag. RODA reads the SIP through
		// ZipInputStream, so it rejected such zips with "Error
		// unzipping file". Filling the header costs one extra streamed
		// pass over the file to checksum it.
		crc := crc32.NewIEEE()
		size, err := io.Copy(crc, in)
		if err != nil {
			return err
		}
		if _, err := in.Seek(0, io.SeekStart); err != nil {
			return err
		}
		// For a stored entry the raw bytes are the file bytes.
		f, err := w.CreateRaw(&zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			CRC32:              crc.Sum32(),
			CompressedSize64:   uint64(size),
			UncompressedSize64: uint64(size),
		})
		if err != nil {
			return err
		}

		_, err = io.Copy(f, in)
		return err
	}
	if err := filepath.WalkDir(src, walker); err != nil {
		w.Close()
		return fmt.Errorf("zipping %s: %w", src, err)
	}

	// The zip writer buffers the central directory until Close: an error
	// here means the file on disk is truncated and unreadable, so it must
	// not be discarded via defer.
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalizing zip of %s: %w", src, err)
	}
	return nil
}
