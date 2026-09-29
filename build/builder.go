package build

import (
	"fmt"
	"log/slog"

	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Config is the builder's wiring: where packages land and how the build
// narrates. What a package is built from is not configuration; it arrives
// per build as a SourcePackage.
type Config struct {
	// Destination is the directory packages are created under.
	Destination string
	// Logger narrates the build.
	Logger *slog.Logger
}

// Builder builds SIP packages from a caller-supplied SourcePackage, driven by a
// profile Definition. It reads no input tree; callers deliver the data.
type Builder struct {
	// Destination is the directory packages are created under.
	Destination string
	// Logger narrates the build.
	Logger *slog.Logger
}

func New(config *Config) *Builder {
	return &Builder{
		Destination: config.Destination,
		Logger:      config.Logger,
	}
}

// Build validates the source package, assembles the complete package graph per
// def (no disk writes), then emits it in the canonical order. Failures
// before the write phase leave no partial package dir behind.
func (b *Builder) Build(def Definition, source *SourcePackage) (*sip.Package, error) {
	// A definition without a descriptive encoder is refused before any
	// side effect. The encoder's check of the source package's descriptions then
	// guarantees every type assertion the encoder makes later.
	if def.Encoder == nil {
		return nil, fmt.Errorf("profile %q names no descriptive encoder; use a registered definition", def.Name)
	}
	if err := checkDescriptions(def.Encoder, source); err != nil {
		return nil, fmt.Errorf("profile %q: %w", def.Name, err)
	}

	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("source package: %w", err)
	}

	b.Logger.Info("starting...")

	pkg, err := b.assemble(def, source)
	if err != nil {
		return nil, err
	}

	st := store.New(pkg.Location)
	if err := b.write(st, pkg, def.Encoder); err != nil {
		return nil, err
	}

	b.Logger.Info("finished.")
	return pkg, nil
}
