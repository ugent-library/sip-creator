package services

import (
	"log/slog"
	"os"
)

func LogLevel(c *Config) slog.Level {
	return slog.LevelInfo
}

func NewLogger(c *Config) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// TODO Add a generic service which yields a file characterisation
//  processor (could be siegfried or droid, conforms to a common interface)
