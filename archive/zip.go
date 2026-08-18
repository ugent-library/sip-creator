package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/sip"
)

type Config struct {
	Destination string
	Logger      *slog.Logger
}

type Archive struct {
	BaseDir string
	Logger  *slog.Logger
}

func New(config *Config) *Archive {
	return &Archive{
		BaseDir: config.Destination,
		Logger:  config.Logger,
	}
}

func (a *Archive) Zip(pkg *sip.Package) error {
	src := pkg.Location
	dest := fmt.Sprintf("%s/%s.zip", a.BaseDir, pkg.Identifier)

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating zip %s: %w", dest, err)
	}
	defer file.Close()

	w := zip.NewWriter(file)

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		a.Logger.Info(fmt.Sprintf("Zipping: %s", path[len(a.BaseDir)+1:]))
		if info.IsDir() {
			// Directories need explicit entries (name ending in "/"):
			// readers otherwise infer them from file paths, and empty
			// directories vanish from the zip entirely.
			_, err := w.CreateHeader(&zip.FileHeader{
				Name:   path[len(a.BaseDir)+1:] + "/",
				Method: zip.Store,
			})
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Don't use any compression by setting Method to zip.Store
		f, err := w.CreateHeader(&zip.FileHeader{
			Name:   path[len(a.BaseDir)+1:],
			Method: zip.Store,
		})

		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	if err := filepath.Walk(src, walker); err != nil {
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
