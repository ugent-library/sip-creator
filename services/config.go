package services

import (
	"github.com/caarlos0/env/v11"
)

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config
type Config struct {
	// Format identification tool (see formats/).
	Formats struct {
		// Name of the format identification tool. Valid values: siegfried.
		Name string `env:"NAME,notEmpty"`
		// Path to the tool's binary on this system.
		Command string `env:"COMMAND,notEmpty"`
		// Extra arguments passed to the tool (e.g. "-hash md5 -json" for siegfried).
		Args string `env:"ARGS,notEmpty"`
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

func ConfigFromEnv() *Config {
	config := &Config{}
	env.Parse(config)
	return config
}
