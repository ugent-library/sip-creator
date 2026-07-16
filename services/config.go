package services

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config
type Config struct {
	// Format identification tool (see formats/). Optional: leaving NAME
	// empty disables format identification — premis:format is a SHOULD,
	// and essence fixity is computed natively during the copy.
	Formats struct {
		// Name of the format identification tool; empty disables format
		// identification. Valid values: siegfried.
		Name string `env:"NAME"`
		// Path to the tool's binary on this system. Required when NAME is set.
		Command string `env:"COMMAND"`
		// Extra arguments passed to the tool (e.g. "-json" for siegfried).
		Args string `env:"ARGS"`
	} `envPrefix:"SIP_FILE_FORMAT_"`
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

func ConfigFromEnv() (*Config, error) {
	config := &Config{}
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("parse environment config: %w", err)
	}
	if config.Formats.Name != "" && config.Formats.Command == "" {
		return nil, fmt.Errorf("SIP_FILE_FORMAT_NAME is set (%q) but SIP_FILE_FORMAT_COMMAND is empty", config.Formats.Name)
	}
	return config, nil
}
