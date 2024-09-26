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

func (a *Archive) Zip(pkg *sip.Package) {
	src := pkg.Location
	dest := fmt.Sprintf("%s/%s.zip", a.BaseDir, pkg.Identifier)

	file, err := os.Create(dest)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		a.Logger.Info(fmt.Sprintf("Zipping: %s", path[len(a.BaseDir)+1:]))
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
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
	err = filepath.Walk(src, walker)
	if err != nil {
		panic(err)
	}
}
