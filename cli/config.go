package cli

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config: the CLI's operator contract, read from the
// environment. The library never sees it — embedding systems supply their
// destination and logger as data (profiles.Config), and each package's
// material — characterization report included — as a profiles.Input, not
// via env vars. Format info comes from the siegfried.json sidecar in the
// input folder (ADR-0009), not from configuration.
type config struct {
	// The submitting organization, stamped into every package's METS as a
	// CREATOR agent. `create` requires NAME for every profile and OR_ID for
	// meemoo profiles; how required each is depends on the profile, so the
	// check lives at profile resolution, not here.
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
	// Build provenance, stamped by the deployment environment. Unused by the CLI itself.
	Version struct {
		// Git branch this build was made from.
		Branch string `env:"SOURCE_BRANCH"`
		// Git commit this build was made from.
		Commit string `env:"SOURCE_COMMIT"`
		// Name of the container image carrying this build.
		Image string `env:"IMAGE_NAME"`
	}
}

func configFromEnv() (*config, error) {
	c := &config{}
	if err := env.Parse(c); err != nil {
		return nil, fmt.Errorf("parse environment config: %w", err)
	}
	return c, nil
}
