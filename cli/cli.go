// Package cli is the sip-creator command line: the create and check
// commands, the --profile flag that selects a profile and its vocabulary,
// and the configuration read from the environment and an optional .env
// file.
package cli

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	cfg    *config
	logger *slog.Logger

	rootCmd = &cobra.Command{
		Use:   "sip-creator",
		Short: "SIP Creator CLI",
		// Execute would print a RunE error and then CheckErr prints it
		// again; silence the first so every error appears exactly once.
		SilenceErrors: true,
	}
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// Run executes the CLI: it loads .env when present, reads the environment
// config, and dispatches the root command.
func Run() {
	// .env is optional: a missing file is fine, a malformed one is an
	// error. Each command checks the variables it needs.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		cobra.CheckErr(err)
	}

	var err error
	cfg, err = configFromEnv()
	cobra.CheckErr(err)

	logger = newLogger()

	cobra.CheckErr(rootCmd.Execute())
}
