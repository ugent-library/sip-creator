package profiles

import (
	"fmt"
	"log/slog"

	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Config is the builder's wiring: where packages land and how the build
// narrates. The material for a package is not configuration; it arrives
// per build as an Input.
type Config struct {
	Destination string
	Logger      *slog.Logger
}

// Builder builds SIP packages from caller-supplied Input, driven by a
// profile Definition. It reads no input tree: the CLI's folder convention
// and any embedding system deliver the same data.
type Builder struct {
	OutDir string
	Logger *slog.Logger
}

func New(config *Config) *Builder {
	return &Builder{
		OutDir: config.Destination,
		Logger: config.Logger,
	}
}

// Build validates the input, assembles the complete package graph per def
// (no disk writes), then emits it in the canonical order. Failures before
// the write phase leave no partial package dir behind.
func (b *Builder) Build(def Definition, in *Input) (*sip.Package, error) {
	// Resolve the family's encodings before anything else: a bogus family
	// is an input error and must fail before any side effect.
	encodeDescriptive, err := def.Family.descriptiveEncoder()
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", def.Name, err)
	}

	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("input: %w", err)
	}
	if err := def.validateDescriptive(in); err != nil {
		return nil, fmt.Errorf("descriptive metadata does not satisfy profile %q:\n%w", def.Name, err)
	}

	b.Logger.Info("starting...")

	pkg, err := b.assemble(def, in)
	if err != nil {
		return nil, err
	}

	st := store.New(pkg.Location)
	if err := b.write(st, pkg, encodeDescriptive); err != nil {
		return nil, err
	}

	b.Logger.Info("finished.")
	return pkg, nil
}
