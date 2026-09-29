package build

import (
	"fmt"
	"log/slog"

	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Config is the builder's wiring: which profile it builds to, where
// packages land and how the build narrates. What a package is built from
// is not configuration; it arrives per build as a SourcePackage.
type Config struct {
	// Profile is the definition every package this builder makes is built
	// to: a registered definition, completed with WithSubmitter. It must
	// name a descriptive encoder.
	Profile Definition
	// Destination is the directory packages are created under.
	Destination string
	// Logger narrates the build.
	Logger *slog.Logger
}

// Builder builds SIP packages to one profile from caller-supplied
// SourcePackage values. It reads no input tree; callers deliver the data.
type Builder struct {
	profile     Definition
	destination string
	logger      *slog.Logger
}

// New returns a builder for the config's profile. A profile without a
// descriptive encoder is refused here, before any build: the encoder's
// check of each source package is what makes every later type assertion
// safe.
func New(config *Config) (*Builder, error) {
	if config.Profile.Encoder == nil {
		return nil, fmt.Errorf("profile %q names no descriptive encoder; use a registered definition", config.Profile.Name)
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
	// The encoder's check of the source package's descriptions guarantees
	// every type assertion the encoder makes later.
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
