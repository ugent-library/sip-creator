package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// The CLI's configuration, read from the environment and an optional
// .env file. The library never reads it. A program that embeds the library
// passes a build.Config, and a build.SourcePackage per build, instead.
// Format information arrives in siegfried.json (ADR-0009), not in
// configuration.
type config struct {
	// The submitting organization, written into every package's METS as a
	// CREATOR agent.
	Submitter struct {
		// Name of the submitting organization, for example "Example
		// Organization". `create` requires it for every profile.
		Name string `env:"NAME"`
		// The organization's Meemoo OR-id (its identifier in Meemoo's
		// organization register), for example "OR-a1b2c3d". `create`
		// requires it for Meemoo profiles, where it becomes the agent's
		// IDENTIFICATIONCODE note.
		ORID string `env:"OR_ID"`
	} `envPrefix:"SIP_SUBMITTER_"`
	// Default content category for created packages (mets/@TYPE, from the
	// CSIP content-category vocabulary), for example "Photographs – Digital".
	// When it is empty, the profile's value applies. The --content-category
	// flag overrides both for one run.
	ContentCategory string `env:"SIP_CONTENT_CATEGORY"`
}

// loadConfig reads .env when present, then the environment, and returns
// the configuration. A variable already set in the environment wins over
// .env. It returns an error if .env is malformed or the environment does
// not parse.
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
