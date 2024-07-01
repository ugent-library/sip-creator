package services

import "github.com/caarlos0/env/v11"

//go:generate go run github.com/g4s8/envdoc@v0.2.4 --output ../CONFIG.md --all

// Application config
type Config struct {
	Version struct {
		Branch string `env:"SOURCE_BRANCH"`
		Commit string `env:"SOURCE_COMMIT"`
		Image  string `env:"IMAGE_NAME"`
	}
}

func ConfigFromEnv() *Config {
	config := &Config{}
	env.Parse(config)
	return config
}
