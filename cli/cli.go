// Package cli is the sip-creator command line: the create and check
// commands, the --profile flag that selects a profile and its vocabulary,
// and the configuration read from the environment and an optional .env
// file.
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
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

// Run executes the CLI. Configuration is read by the commands that need
// it, so check runs without it (ADR-0010).
func Run() {
	logger = newLogger()

	cobra.CheckErr(rootCmd.Execute())
}
