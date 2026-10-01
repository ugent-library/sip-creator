package build

import (
	"fmt"
	"log/slog"

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
	// organization added by WithSubmitter. It must name a descriptive
	// encoder.
	Profile Definition
	// Destination is the directory packages are created under.
	Destination string
	// Logger receives the build's progress messages.
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
// descriptive encoder is refused here, before any build.
func New(config *Config) (*Builder, error) {
	if config.Profile.Encoder == nil {
		return nil, fmt.Errorf("profile %q names no descriptive encoder; set the definition's Encoder", config.Profile.Name)
	}
	return &Builder{
		profile:     config.Profile,
		destination: config.Destination,
		logger:      config.Logger,
	}, nil
}

// Build validates the source package, assembles the complete package graph
// (no disk writes), then emits it in the canonical order. Failures before
// the write phase leave no partial package dir behind.
func (b *Builder) Build(source *SourcePackage) (*sip.Package, error) {
	if err := checkDescriptions(b.profile.Encoder, source); err != nil {
		return nil, fmt.Errorf("profile %q: %w", b.profile.Name, err)
	}
	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("source package: %w", err)
	}

	b.logger.Info("starting...")

	pkg, err := b.assemble(source)
	if err != nil {
		return nil, err
	}

	st := store.New(pkg.Location)
	if err := b.write(st, pkg); err != nil {
		return nil, err
	}

	b.logger.Info("finished.")
	return pkg, nil
}
