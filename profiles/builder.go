package profiles

import (
	"fmt"
	"log/slog"

	"github.com/ugent-library/sip-creator/characterization"
	"github.com/ugent-library/sip-creator/sip"
	"github.com/ugent-library/sip-creator/store"
)

// Config carries the operator's inputs for building one package.
type Config struct {
	Source      string
	Destination string
	Logger      *slog.Logger
	// Characterization optionally supplies a pre-decoded characterization
	// report as data; nil means assemble discovers the profile's sidecar
	// file in the input tree (ADR-0009). Strictness is the same either way.
	Characterization characterization.Report
}

// Builder builds SIP packages from an input tree, driven by a profile
// Definition.
type Builder struct {
	OutDir string
	InDir  string
	Logger *slog.Logger
	// Characterization optionally enriches essence files with format
	// information; nil falls back to the profile's sidecar file, and the
	// build proceeds without format info when neither exists (ADR-0009).
	Characterization characterization.Report
}

func New(config *Config) *Builder {
	return &Builder{
		OutDir:           config.Destination,
		InDir:            config.Source,
		Logger:           config.Logger,
		Characterization: config.Characterization,
	}
}

// Build assembles the complete package graph per def (no disk writes),
// then emits it in the canonical order. Assembly failures happen before
// the store exists, so a bad input leaves no partial package dir behind.
func (b *Builder) Build(def Definition) (*sip.Package, error) {
	// Resolve the family's encodings before anything else: a bogus family
	// is an input error and must fail before any side effect.
	encodeDescriptive, err := def.Family.descriptiveEncoder()
	if err != nil {
		return nil, fmt.Errorf("profile %q: %w", def.Name, err)
	}

	b.Logger.Info("starting...")

	pkg, err := b.assemble(def)
	if err != nil {
		return nil, err
	}

	st := store.New(pkg.Location)
	if err := b.write(st, pkg, def, encodeDescriptive); err != nil {
		return nil, err
	}

	b.Logger.Info("finished.")
	return pkg, nil
}
