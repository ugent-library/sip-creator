package archive

import (
	"archive/zip"
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
	// Logger narrates the zipping.
	Logger *slog.Logger
}

// Archive zips built package directories.
type Archive struct {
	// Destination is the directory the zip is written to.
	Destination string
	// Logger narrates the zipping.
	Logger *slog.Logger
}

func New(config *Config) *Archive {
	return &Archive{
		Destination: config.Destination,
		Logger:      config.Logger,
	}
}

// Zip writes the package directory to dest/uuid-<uuid>.zip with every
// entry stored uncompressed.
func (a *Archive) Zip(pkg *sip.Package) error {
	src := pkg.Location
	dest := filepath.Join(a.Destination, pkg.Identifier+".zip")

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating zip %s: %w", dest, err)
	}
	defer file.Close()

	w := zip.NewWriter(file)

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Entry names are relative to the destination dir, so the package
		// dir (uuid-<uuid>/) stays the top-level entry; zip names are
		// slash-separated regardless of platform.
		rel, err := filepath.Rel(a.Destination, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		a.Logger.Info("zipping", slog.String("path", name))
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
		return fmt.Errorf("finalizing zip %s: %w", dest, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing zip %s: %w", dest, err)
	}
	return nil
}
