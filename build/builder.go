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

// Config holds what a Builder needs besides the packages it builds: the
// profile, the directory packages are written to and the logger.
type Config struct {
	// Profile is the definition every package this builder makes follows.
	// It comes from profiles.Get, or is written for another descriptive
	// standard (ADR-0022). WithSubmitter adds the submitting organization to
	// it. Its Model must be set.
	Profile Definition
	// Destination is the directory packages are created under.
	Destination string
	// Logger receives the build's progress messages. Nil discards them.
	Logger *slog.Logger
}

// Builder builds SIP packages to one profile.
type Builder struct {
	profile     Definition
	destination string
	logger      *slog.Logger
}

// New checks that the config's profile has a metadata model and that the
// model's format name can be recorded in METS. It returns a Builder for
// the profile, or an error if a check fails.
func New(config *Config) (*Builder, error) {
	if config.Profile.Model == nil {
		return nil, fmt.Errorf("profile %q names no metadata model; set the definition's Model", config.Profile.Name)
	}
	if err := validateModelType(config.Profile.Model.ModelType()); err != nil {
		return nil, fmt.Errorf("profile %q: %w", config.Profile.Name, err)
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

// validateModelType checks that name, a metadata model's format name, is
// not empty, because METS requires MDTYPE, and that it is text XML can
// carry. It returns an error if a check fails.
func validateModelType(name string) error {
	if name == "" {
		return errors.New("the metadata model names no format; ModelType must return one, such as DC or MODS")
	}
	if err := ValidateXMLText(name); err != nil {
		return fmt.Errorf("the metadata model's format name: %w", err)
	}
	return nil
}

// Build validates the source package, assembles the package graph without
// writing anything, and then writes the package to the directory
// <Destination>/<identifier>. It returns the graph, with each file's size
// and checksum filled in. It returns an error if validation, assembly or
// writing fails, or if the package directory already exists. A failed
// Build leaves no package directory behind.
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
// final name never holds part of a package. It returns an error if a
// directory with the final name already exists, and never writes into it.
// An update reuses the earlier package's identifier, and writing into that
// package's directory would leave its files beside the new ones. A failed
// write removes the temporary directory. A process killed partway leaves
// only that directory, whose name starts with a dot and ends in .tmp.
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
