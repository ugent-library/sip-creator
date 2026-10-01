package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config: the CLI's operator contract, read from the
// environment. The library never reads it: embedding systems pass a
// build.Config and per-build build.SourcePackage instead, and format info
// arrives via the siegfried.json sidecar (ADR-0009), not configuration.
type config struct {
	// The submitting organization, stamped into every package's METS as a
	// CREATOR agent. `create` requires NAME for every profile and OR_ID for
	// Meemoo profiles.
	Submitter struct {
		// Name of the submitting organization, e.g. "Example Organization".
		Name string `env:"NAME"`
		// The organization's Meemoo OR-id (its identifier in Meemoo's
		// organization register), e.g. "OR-a1b2c3d". Required for Meemoo
		// profiles, where it becomes the agent's IDENTIFICATIONCODE note.
		ORID string `env:"OR_ID"`
	} `envPrefix:"SIP_SUBMITTER_"`
	// Default content category for created packages (mets/@TYPE, CSIP
	// content-category vocabulary), e.g. "Photographs – Digital". Empty
	// means the profile's registry value; --content-category overrides
	// both per run.
	ContentCategory string `env:"SIP_CONTENT_CATEGORY"`
}

// loadConfig reads .env when present and then the environment. A missing
// .env is fine, a malformed one is an error.
func loadConfig() (*config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}
	c := &config{}
	if err := env.Parse(c); err != nil {
		return nil, fmt.Errorf("parse environment config: %w", err)
	}
	return c, nil
}
