// Package archive zips a built package directory.
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

// Config sets the directory an Archive writes zips to and the logger it
// reports to.
type Config struct {
	// Destination is the directory the zip is written to.
	Destination string
	// Logger receives one message per zipped entry. When it is nil, the
	// messages are discarded.
	Logger *slog.Logger
}

// Archive zips built package directories.
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

// ValidateDestination checks that the destination does not yet hold the
// zip for the package with this identifier. It returns an error if the zip
// exists or if the check fails. Zip never replaces an existing zip
// (ADR-0031). Calling ValidateDestination before a build keeps a refused
// zip from leaving a newly built package directory behind.
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

// Zip writes the package directory to <identifier>.zip in the destination
// directory, with every entry stored uncompressed. It returns an error if
// that zip already exists. Zip writes to a temporary file next to the zip
// and renames that file once every entry is written, so the final name
// only ever holds a complete zip (ADR-0031). A failed zip leaves nothing
// behind. A process killed partway leaves only the temporary file, whose
// name does not end in .zip. The next Zip for that identifier writes over
// it.
func (a *Archive) Zip(pkg *sip.Package) (err error) {
	if err := a.ValidateDestination(pkg.Identifier); err != nil {
		return err
	}
	dest := a.zipPath(pkg.Identifier)

	// The temporary file sits in the same directory, so the rename stays on
	// one file system, where it is atomic. It is opened with mode 0o666, as
	// os.Create does, so the zip gets the usual permissions after the umask.
	tmp := filepath.Join(a.destination, "."+pkg.Identifier+".zip.tmp")
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("creating zip %s: %w", dest, err)
	}
	defer func() {
		if err != nil {
			// The error from this Close is ignored. The file may already be
			// closed, and err already holds the error that stopped Zip.
			file.Close()
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

// writeEntries writes every file and directory under src to out as a zip.
func (a *Archive) writeEntries(out io.Writer, src string) error {
	w := zip.NewWriter(out)

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Entry names are relative to the destination directory, so the
		// package directory is the top-level entry. The zip format requires
		// forward slashes in entry names on every platform.
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
			// Each directory gets an entry of its own, with a name ending
			// in "/". Without one, a directory exists only as part of the
			// file paths, and an empty directory is missing from the zip.
			// CreateHeader sets the MS-DOS date fields from Modified itself,
			// so a directory entry needs no setModified.
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
		// written with CreateRaw rather than CreateHeader so the size and
		// CRC go into the local file header. CreateHeader streams the
		// entry. It leaves both zero and sets general purpose bit 3, which
		// says they follow the data in a data descriptor. Java's
		// ZipInputStream throws "only DEFLATED entries can have EXT
		// descriptor" on a stored entry with that flag. RODA reads the SIP
		// through ZipInputStream, so it rejects such a zip with "Error
		// unzipping file".
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

	// The zip writer writes the central directory only on Close, so an
	// error from Close means the zip is incomplete and unreadable.
	if err := w.Close(); err != nil {
		return fmt.Errorf("finalizing zip of %s: %w", src, err)
	}
	return nil
}

// setModified records t as the entry's modification time. CreateRaw
// writes the header as given, so setModified does what CreateHeader does
// with FileHeader.Modified. It sets the MS-DOS date fields in t's own time
// zone, because those fields carry no time zone and unzip tools read them
// as local time. It adds an Info-ZIP extended timestamp with the exact
// instant, which zip readers that know that field read instead. Go's
// SetModTime would write the MS-DOS fields in UTC, and extracted files
// would then be off by the local offset.
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
