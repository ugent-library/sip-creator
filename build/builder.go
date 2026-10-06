package build

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Config is the builder's wiring: which profile it builds to, where
// packages land and where the build logs. What a package is built from
// is not configuration; it arrives per build as a SourcePackage.
type Config struct {
	// Profile is the definition every package this builder makes is built
	// to: one of this module's profiles (see profiles.Get) or one written
	// for another descriptive standard (ADR-0022), with the submitting
	// organization added by WithSubmitter. It must name a metadata model.
	Profile Definition
	// Destination is the directory packages are created under.
	Destination string
	// Logger receives the build's progress messages. Nil discards them.
	Logger *slog.Logger
}

// Builder builds SIP packages to one profile. It reads no input folder and
// no environment: everything one package is built from arrives in the
// SourcePackage handed to Build.
type Builder struct {
	profile     Definition
	destination string
	logger      *slog.Logger
}

// New returns a builder for the config's profile. A profile without a
// metadata model is refused here, before any build.
func New(config *Config) (*Builder, error) {
	if config.Profile.Model == nil {
		return nil, fmt.Errorf("profile %q names no metadata model; set the definition's Model", config.Profile.Name)
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Builder{
		profile:     config.Profile,
		destination: config.Destination,
		logger:      logger,
	}, nil
}

// Build validates the source package, assembles the complete package graph
// (no disk writes), then emits it in the canonical order. The package
// directory, dest/<identifier>, only ever holds a complete package: Build
// refuses one that already exists, and a failed Build leaves nothing
// behind.
func (b *Builder) Build(source *SourcePackage) (*sip.Package, error) {
	if err := b.profile.ValidateSource(source); err != nil {
		return nil, err
	}
	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("source package: %w", err)
	}

	b.logger.Info("starting...")

	pkg, err := b.assemble(source)
	if err != nil {
		return nil, err
	}

	if err := b.writePackage(pkg); err != nil {
		return nil, err
	}

	b.logger.Info("finished.")
	return pkg, nil
}

// writePackage writes pkg into a temporary directory next to its final
// one and renames it to pkg.Location once every file is written, so the
// final name never holds part of a package. A directory that already has
// the final name is refused, never written into: an update reuses the
// earlier package's identifier, and writing into that package's directory
// would leave its files beside the new ones. A failed write removes the
// temporary directory; a process killed partway leaves only that
// directory, whose name starts with a dot and ends in .tmp.
func (b *Builder) writePackage(pkg *sip.Package) (err error) {
	if _, err := os.Lstat(pkg.Location); err == nil {
		return fmt.Errorf("package directory %s already exists; move it away first", pkg.Location)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking package directory %s: %w", pkg.Location, err)
	}

	tmp := filepath.Join(filepath.Dir(pkg.Location), "."+pkg.Identifier+".tmp")
	// A temporary directory left by a killed run would mix its files into
	// this package, so it goes first.
	if err := os.RemoveAll(tmp); err != nil {
		return fmt.Errorf("removing the leftover %s: %w", tmp, err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tmp)
		}
	}()

	if err := b.write(store.New(tmp), pkg); err != nil {
		return err
	}
	if err := os.Rename(tmp, pkg.Location); err != nil {
		return fmt.Errorf("finalizing package directory %s: %w", pkg.Location, err)
	}
	return nil
}
