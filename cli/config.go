package cli

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config: the CLI's operator contract, read from the
// environment. The library never reads it: embedding systems pass a
// profiles.Config and per-build profiles.Input instead, and format info
// arrives via the siegfried.json sidecar (ADR-0009), not configuration.
type config struct {
	// The submitting organization, stamped into every package's METS as a
	// CREATOR agent. `create` requires NAME for every profile and OR_ID for
	// meemoo profiles.
	Submitter struct {
		// Name of the submitting organization, e.g. "Universiteitsbibliotheek Gent".
		Name string `env:"NAME"`
		// The organization's meemoo OR-id (its identifier in meemoo's
		// organization register), e.g. "OR-a1b2c3d". Required for meemoo
		// profiles, where it becomes the agent's IDENTIFICATIONCODE note.
		ORID string `env:"OR_ID"`
	} `envPrefix:"SIP_SUBMITTER_"`
	// Default content category for created packages (mets/@TYPE, CSIP
	// content-category vocabulary), e.g. "Photographs – Digital". Empty
	// means the profile's registry value; --content-category overrides
	// both per run.
	ContentCategory string `env:"SIP_CONTENT_CATEGORY"`
}

func configFromEnv() (*config, error) {
	c := &config{}
	if err := env.Parse(c); err != nil {
		return nil, fmt.Errorf("parse environment config: %w", err)
	}
	return c, nil
}
