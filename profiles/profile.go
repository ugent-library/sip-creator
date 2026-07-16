package profiles

import (
	"log/slog"

	"github.com/ugent-library/sip-creator/formats"
)

// Config carries the operator's inputs for building one package.
type Config struct {
	Source      string
	Destination string
	Logger      *slog.Logger
	Formats     formats.Identificator
}

// Profile builds SIP packages from an input tree: see Basic.
type Profile struct {
	OutDir  string
	InDir   string
	Logger  *slog.Logger
	Formats formats.Identificator
}

func New(config *Config) *Profile {
	return &Profile{
		OutDir:  config.Destination,
		InDir:   config.Source,
		Logger:  config.Logger,
		Formats: config.Formats,
	}
}
