// Package archive zips a built package directory into uuid-<uuid>.zip,
// with every entry stored uncompressed.
package archive

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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

// ValidateDestination returns an error when the destination already holds
// the zip for the package with this identifier. Zip never replaces one: it
// may be one a transfer has picked up, or another build's. A program that
// knows the identifier before building calls it first, so a refused zip
// does not leave a freshly built package directory behind.
func (a *Archive) ValidateDestination(identifier string) error {
	dest := a.zipPath(identifier)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("zip %s already exists; move it away first", dest)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking zip %s: %w", dest, err)
	}
	return nil
}

func (a *Archive) zipPath(identifier string) string {
	return filepath.Join(a.destination, identifier+".zip")
}

// Zip writes the package directory to dest/uuid-<uuid>.zip with every
// entry stored uncompressed. The final name only ever holds a complete
// zip: Zip refuses a zip that already exists, writes to a temporary file
// next to it, and renames that file once every entry is written. A failed
// zip leaves nothing behind; a process killed partway leaves only the
// temporary file, whose name does not end in .zip.
func (a *Archive) Zip(pkg *sip.Package) (err error) {
	if err := a.ValidateDestination(pkg.Identifier); err != nil {
		return err
	}
	dest := a.zipPath(pkg.Identifier)

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
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Directories need explicit entries (name ending in "/"):
			// readers otherwise infer them from file paths, and empty
			// directories vanish from the zip entirely.
			// CreateHeader encodes Modified itself.
			_, err := w.CreateHeader(&zip.FileHeader{
				Name:     name + "/",
				Method:   zip.Store,
				Modified: info.ModTime(),
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
		header := &zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			CRC32:              crc.Sum32(),
			CompressedSize64:   uint64(size),
			UncompressedSize64: uint64(size),
		}
		setModified(header, info.ModTime())
		f, err := w.CreateRaw(header)
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

// setModified records t as the entry's modification time, so the entry
// carries the date of its file in the package directory instead of the
// zero date. CreateHeader encodes FileHeader.Modified itself, but
// CreateRaw writes the header as given, so this does what CreateHeader
// does: the MS-DOS date fields in t's own time zone, which most unzip
// tools read as local time, and an Info-ZIP extended timestamp with the
// exact instant, which zip readers that know it prefer. Go's SetModTime
// would write the MS-DOS fields in UTC, and extracted files would then be
// off by the local offset.
func setModified(header *zip.FileHeader, t time.Time) {
	header.Modified = t
	header.ModifiedDate = uint16(t.Day() + int(t.Month())<<5 + (t.Year()-1980)<<9)
	header.ModifiedTime = uint16(t.Second()/2 + t.Minute()<<5 + t.Hour()<<11)

	// The extended timestamp extra field (ID 0x5455): five bytes of data,
	// a flags byte saying only the modification time follows, then that
	// time in Unix seconds.
	var extra [9]byte
	binary.LittleEndian.PutUint16(extra[0:], 0x5455)
	binary.LittleEndian.PutUint16(extra[2:], 5)
	extra[4] = 1
	binary.LittleEndian.PutUint32(extra[5:], uint32(t.Unix()))
	header.Extra = append(header.Extra, extra[:]...)
}
